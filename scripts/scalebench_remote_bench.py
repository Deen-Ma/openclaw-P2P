#!/usr/bin/env python3
import argparse
import csv
import json
import math
import os
import re
import statistics
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path

GENERAL_TOPIC = "openagent/v1/general"
SUDO_PW = os.getenv("SCALEBENCH_SUDO_PW", "ma123")

EVENT_TYPE_TO_SIGNAL = {
    "coarse_match_true": "coarse_match",
    "full_match_true": "full_match",
    "candidate_return_received": "candidate_return",
    "candidate_return_sent": "candidate_return_sent",
    "state_active": "state_active",
    "state_terminal": "state_terminal",
    "state_expired": "state_expired",
    "state_replay_or_old": "state_replay_or_old",
    "state_conflict": "state_conflict",
    "payload_fetch_invalid": "payload_fetch_invalid",
    "payload_fetch_parse_error": "payload_fetch_parse_error",
}

CORE_SIGNAL_FIELDS = [
    "visible",
    "coarse_match",
    "full_match",
    "candidate_return",
    "candidate_return_sent",
    "state_active",
    "state_terminal",
    "state_expired",
    "state_replay_or_old",
    "state_conflict",
    "payload_fetch_invalid",
    "payload_fetch_parse_error",
]


def get_json(port, path):
    with urllib.request.urlopen(f"http://127.0.0.1:{port}{path}", timeout=5) as r:
        return json.loads(r.read().decode())


def post_json(port, path, body):
    req = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}",
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def shell(cmd):
    return subprocess.check_output(cmd, shell=True, text=True)


def read_iface_bytes(iface):
    base = Path(f"/sys/class/net/{iface}/statistics")
    return {
        "rx_bytes": int((base / "rx_bytes").read_text().strip()),
        "tx_bytes": int((base / "tx_bytes").read_text().strip()),
    }


def detect_phys_iface(explicit_iface=""):
    candidate = (explicit_iface or "").strip() or os.getenv("SCALEBENCH_NET_IFACE", "").strip()
    if candidate and (Path(f"/sys/class/net/{candidate}") / "statistics").exists():
        return candidate
    try:
        route = shell("ip route show default").strip().splitlines()
        if route:
            parts = route[0].split()
            if "dev" in parts:
                idx = parts.index("dev") + 1
                if idx < len(parts):
                    iface = parts[idx]
                    if (Path(f"/sys/class/net/{iface}") / "statistics").exists():
                        return iface
    except Exception:
        pass
    for path in sorted(Path("/sys/class/net").iterdir()):
        if path.name == "lo":
            continue
        if (path / "statistics").exists():
            return path.name
    raise RuntimeError("failed to detect physical network interface")


def parse_load1():
    return float(Path("/proc/loadavg").read_text().split()[0])


def parse_event_line(line):
    line = line.strip()
    if not line:
        return None
    try:
        return json.loads(line)
    except json.JSONDecodeError:
        return None


class EventTailCollector:
    def __init__(self, event_paths):
        self.event_paths = list(event_paths)
        self.offsets = {}

    def prime(self):
        for path in self.event_paths:
            try:
                self.offsets[path] = path.stat().st_size
            except (FileNotFoundError, PermissionError, OSError):
                self.offsets[path] = None

    def consume_task(self, task_id):
        counters = {field: 0 for field in CORE_SIGNAL_FIELDS}
        for path in self.event_paths:
            prev = self.offsets.get(path, 0)
            if prev is None:
                continue
            if not path.exists():
                self.offsets[path] = None
                continue
            try:
                with path.open("r", encoding="utf-8", errors="ignore") as fh:
                    fh.seek(prev)
                    for raw_line in fh:
                        row = parse_event_line(raw_line)
                        if not row:
                            continue
                        if str(row.get("task_id", "")).strip() != task_id:
                            continue
                        event_type = str(row.get("type", "")).strip()
                        signal_key = EVENT_TYPE_TO_SIGNAL.get(event_type)
                        if signal_key:
                            counters[signal_key] += 1
                    self.offsets[path] = fh.tell()
            except (FileNotFoundError, PermissionError, OSError):
                self.offsets[path] = None
        return counters


def task_visible(port, task_id):
    try:
        rows = get_json(port, "/v1/tasks")
    except Exception:
        return False
    for row in rows:
        if row.get("task_id") == task_id:
            return True
        env = row.get("envelope") or {}
        if env.get("task_id") == task_id:
            return True
    return False


def stats(values):
    cleaned = [v for v in values if v is not None]
    if not cleaned:
        return {"count": 0}
    cleaned = sorted(cleaned)
    return {
        "count": len(cleaned),
        "min": round(cleaned[0], 2),
        "median": round(statistics.median(cleaned), 2),
        "p95": round(cleaned[max(0, math.ceil(len(cleaned) * 0.95) - 1)], 2),
        "max": round(cleaned[-1], 2),
        "avg": round(sum(cleaned) / len(cleaned), 2),
    }


def cover_latency(first_seen_ms, threshold_count):
    if len(first_seen_ms) < threshold_count:
        return None
    return round(sorted(first_seen_ms)[threshold_count - 1], 2)


def sample_state(base_dir, name, sample_ports, node_count):
    topo_dir = base_dir / "sample_topology"
    pubsub_dir = base_dir / "sample_pubsub"
    topo_dir.mkdir(parents=True, exist_ok=True)
    pubsub_dir.mkdir(parents=True, exist_ok=True)
    for port in sample_ports:
        try:
            (topo_dir / f"{name}_{port}.json").write_text(json.dumps(get_json(port, "/v1/topology"), indent=2))
        except Exception as e:
            (topo_dir / f"{name}_{port}.err.txt").write_text(str(e))
        try:
            (pubsub_dir / f"{name}_{port}.json").write_text(json.dumps(get_json(port, "/v1/pubsub"), indent=2))
        except Exception as e:
            (pubsub_dir / f"{name}_{port}.err.txt").write_text(str(e))


def resource_snapshot(base_dir, name, node_count):
    snap = base_dir / "resource_snapshots"
    snap.mkdir(parents=True, exist_ok=True)
    (snap / f"{name}_uptime.txt").write_text(shell("uptime"))
    (snap / f"{name}_free.txt").write_text(shell("free -h"))
    (snap / f"{name}_ss_established.txt").write_text(shell("ss -tan state established"))
    stats_cmd = (
        f"printf '{SUDO_PW}\\n' | sudo -S docker stats --no-stream "
        "--format '{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.PIDs}}' "
        f"$(printf 'oa-formal-%s ' $(seq 1 {node_count}))"
    )
    (snap / f"{name}_docker_stats.txt").write_text(shell(stats_cmd))


def target_ports_for_shard(api_base, shard):
    start = api_base + (shard - 1) * 20
    return list(range(start, start + 20))


def target_cover_metrics(first_target_ms, target_total):
    return {
        "cover90_ms_target": cover_latency(first_target_ms, math.ceil(target_total * 0.90)),
        "cover95_ms_target": cover_latency(first_target_ms, math.ceil(target_total * 0.95)),
        "cover99_ms_target": cover_latency(first_target_ms, math.ceil(target_total * 0.99)),
        "cover100_ms_target": cover_latency(first_target_ms, target_total),
    }


def global_cover_metrics(first_global_ms, node_count):
    return {
        "cover90_ms_global": cover_latency(first_global_ms, math.ceil(node_count * 0.90)),
        "cover95_ms_global": cover_latency(first_global_ms, math.ceil(node_count * 0.95)),
        "cover99_ms_global": cover_latency(first_global_ms, math.ceil(node_count * 0.99)),
        "cover100_ms_global": cover_latency(first_global_ms, node_count),
    }


def round_plan(shard_count, rounds_per_shard, request_count):
    if request_count and request_count > 0:
        return list(range(1, request_count + 1))
    return list(range(1, shard_count * rounds_per_shard + 1))


def shard_for_round(round_idx, shard_count, topic_mode):
    if topic_mode == "public":
        return 1
    return ((round_idx - 1) % shard_count) + 1


def repetition_for_round(round_idx, shard_count):
    if shard_count <= 0:
        return round_idx
    return ((round_idx - 1) // shard_count) + 1


def build_detail(label, shard, repetition, payload_size):
    detail = {"label": label, "shard": shard, "repetition": repetition}
    if payload_size and payload_size > 0:
        detail["payload_blob"] = "x" * payload_size
        detail["payload_size"] = payload_size
    return detail


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True)
    ap.add_argument("--label", required=True)
    ap.add_argument("--variant", required=True)
    ap.add_argument("--node-count", required=True, type=int)
    ap.add_argument("--rounds-per-shard", type=int, default=2)
    ap.add_argument("--request-count", type=int, default=0)
    ap.add_argument("--topic-mode", choices=["public", "topic_aware"], default="topic_aware")
    ap.add_argument("--payload-mode", choices=["full", "bodyless"], default="bodyless")
    ap.add_argument("--seq-rule", choices=["on", "off"], default="on")
    ap.add_argument("--payload-size", type=int, default=0)
    ap.add_argument("--publish-port", type=int, default=8500)
    ap.add_argument("--api-base", type=int, default=8500)
    ap.add_argument("--observe-sec", type=int, default=60)
    ap.add_argument("--poll-sec", type=float, default=1.0)
    ap.add_argument("--event-log-root", default="")
    ap.add_argument("--round-gap-sec", type=float, default=10.0)
    ap.add_argument("--seed", default="")
    ap.add_argument("--net-iface", default="")
    args = ap.parse_args()

    root = Path(args.root).expanduser()
    shard_count = args.node_count // 20
    if args.topic_mode == "topic_aware" and shard_count <= 0:
        raise SystemExit("topic_aware mode requires node_count >= 20")
    out = root / f"bench_shards_{time.strftime('%Y%m%dT%H%M%SZ', time.gmtime())}_{args.label}"
    out.mkdir(parents=True, exist_ok=True)
    phys_iface = detect_phys_iface(args.net_iface)

    ports = list(range(args.api_base, args.api_base + args.node_count))
    health_ok = 0
    for p in ports:
        try:
            get_json(p, "/healthz")
            health_ok += 1
        except Exception:
            pass
    if health_ok != args.node_count:
        raise SystemExit(f"health gate failed: {health_ok}/{args.node_count}")

    sample_ports = sorted(set([args.api_base, min(args.api_base + 1, ports[-1]), ports[min(len(ports)-1, 99)], ports[len(ports)//2], ports[-1]]))
    resource_snapshot(out, "baseline", args.node_count)
    sample_state(out, "baseline", sample_ports, args.node_count)
    event_log_root = Path(args.event_log_root).expanduser() if args.event_log_root else root
    collector = EventTailCollector([
        event_log_root / f"node{i}" / "state" / "events.jsonl"
        for i in range(1, args.node_count + 1)
    ])
    collector.prime()

    rounds = []
    matrix_rows = []
    plan = round_plan(shard_count, args.rounds_per_shard, args.request_count)
    total_rounds = len(plan)
    mid_round = max(1, (total_rounds + 1) // 2)
    default_publish_port = args.publish_port

    for round_index, plan_idx in enumerate(plan, start=1):
        shard = shard_for_round(plan_idx, max(1, shard_count), args.topic_mode)
        repetition = repetition_for_round(plan_idx, max(1, shard_count))
        shard_topic = f"openagent/v1/bench/shard-{shard:03d}"
        target_ports = ports if args.topic_mode == "public" else target_ports_for_shard(args.api_base, shard)
        publish_topic = GENERAL_TOPIC if args.topic_mode == "public" else shard_topic
        publisher_port = default_publish_port if args.topic_mode == "public" or args.variant == "ablation_no_semantic_public_topic" else target_ports[0]
        task_id = f"scalebench-{args.label}-s{shard:03d}-r{repetition:02d}"
        body = {
            "task_id": task_id,
            "taxonomy": "crowd.data_labeling",
            "topics": [publish_topic],
            "summary": f"{args.label} shard {shard} round {repetition}",
            "detail": build_detail(args.label, shard, repetition, args.payload_size),
            "conf": 900,
            "ttl_sec": 3600,
        }
        lo_before = read_iface_bytes("lo")
        phys_before = read_iface_bytes(phys_iface)
        start = time.time()
        pub = post_json(publisher_port, "/v1/tasks/publish", body)

        first_global = {}
        first_target = {}
        deadline = start + args.observe_sec
        last_new_global = start
        while time.time() < deadline:
            for p in ports:
                if p not in first_global and task_visible(p, task_id):
                    ts = time.time()
                    first_global[p] = ts
                    last_new_global = ts
                    if p in target_ports:
                        first_target[p] = ts
            now = time.time()
            if len(first_target) == len(target_ports) and (now - start) >= 3 and (now - last_new_global) >= 2:
                break
            time.sleep(args.poll_sec)

        lo_after = read_iface_bytes("lo")
        phys_after = read_iface_bytes(phys_iface)
        first_target_ms = [round((ts - start) * 1000.0, 2) for ts in first_target.values()]
        first_target_excl_publisher_ms = [round((ts - start) * 1000.0, 2) for p, ts in first_target.items() if p != publisher_port]
        first_global_ms = [round((ts - start) * 1000.0, 2) for ts in first_global.values()]
        target_metrics = target_cover_metrics(first_target_ms, len(target_ports))
        signal_counts = collector.consume_task(task_id)
        row = {
            "round_index": round_index,
            "task_id": task_id,
            "variant": args.variant,
            "label": args.label,
            "topic_mode": args.topic_mode,
            "payload_mode": args.payload_mode,
            "seq_rule": args.seq_rule,
            "payload_size": args.payload_size,
            "seed": str(args.seed).strip(),
            "request_count": args.request_count if args.request_count > 0 else None,
            "detail_ref": pub.get("detail_ref"),
            "publisher_port": publisher_port,
            "topic": publish_topic,
            "shard": shard,
            "target_ports": target_ports,
            "first_seen_ms_target_excl_publisher": min(first_target_excl_publisher_ms) if first_target_excl_publisher_ms else None,
            "seen_count_target": len(first_target),
            "missing_target_ports": [p for p in target_ports if p not in first_target],
            "seen_count_global": len(first_global),
            "seen_ratio_global": round(len(first_global) / args.node_count, 4),
            "spillover_count_global": len([p for p in first_global if p not in target_ports]),
            "spillover_ratio_global": round(len([p for p in first_global if p not in target_ports]) / max(1, args.node_count - len(target_ports)), 4),
            "visible": len(first_target),
            "load1": parse_load1(),
            "net_lo_rx_bytes_delta": lo_after["rx_bytes"] - lo_before["rx_bytes"],
            "net_lo_tx_bytes_delta": lo_after["tx_bytes"] - lo_before["tx_bytes"],
            "net_lo_bytes_delta_total": (lo_after["rx_bytes"] - lo_before["rx_bytes"]) + (lo_after["tx_bytes"] - lo_before["tx_bytes"]),
            "net_eno1np0_rx_bytes_delta": phys_after["rx_bytes"] - phys_before["rx_bytes"],
            "net_eno1np0_tx_bytes_delta": phys_after["tx_bytes"] - phys_before["tx_bytes"],
            "net_eno1np0_bytes_delta_total": (phys_after["rx_bytes"] - phys_before["rx_bytes"]) + (phys_after["tx_bytes"] - phys_before["tx_bytes"]),
            "bytes_per_target_delivered_lo": round(((lo_after["rx_bytes"] - lo_before["rx_bytes"]) + (lo_after["tx_bytes"] - lo_before["tx_bytes"])) / max(1, len(first_target)), 2),
            "bytes_per_global_delivered_lo": round(((lo_after["rx_bytes"] - lo_before["rx_bytes"]) + (lo_after["tx_bytes"] - lo_before["tx_bytes"])) / max(1, len(first_global)), 2),
            **target_metrics,
            **global_cover_metrics(first_global_ms, args.node_count),
            **signal_counts,
        }
        rounds.append(row)
        for p in ports:
            matrix_rows.append({
                "round_index": round_index,
                "task_id": task_id,
                "variant": args.variant,
                "shard": shard,
                "target": p in target_ports,
                "api_port": p,
                "seen": p in first_global,
                "first_seen_ms": round((first_global[p] - start) * 1000.0, 2) if p in first_global else None,
            })
        with (out / "per_round.jsonl").open("a") as f:
            f.write(json.dumps(row) + "\n")
        if round_index in {mid_round, total_rounds}:
            tag = "mid" if round_index == mid_round else "final"
            resource_snapshot(out, tag, args.node_count)
            sample_state(out, tag, sample_ports, args.node_count)
        if args.round_gap_sec > 0:
            time.sleep(args.round_gap_sec)

    with (out / "first_seen_matrix.csv").open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=["round_index", "task_id", "variant", "shard", "target", "api_port", "seen", "first_seen_ms"])
        writer.writeheader()
        writer.writerows(matrix_rows)

    core_csv_fields = [
        "round_index", "task_id", "variant", "topic_mode", "payload_mode", "seq_rule",
        "payload_size", "publisher_port", "topic", "shard",
        "visible", "seen_count_target", "seen_count_global",
        "cover95_ms_target", "cover99_ms_target", "cover100_ms_target",
        "cover95_ms_global", "cover99_ms_global", "cover100_ms_global",
        "coarse_match", "full_match", "candidate_return", "candidate_return_sent",
        "state_active", "state_terminal", "state_expired", "state_replay_or_old", "state_conflict",
        "payload_fetch_invalid", "payload_fetch_parse_error",
    ]
    with (out / "core_metrics.csv").open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=core_csv_fields)
        writer.writeheader()
        writer.writerows([{k: row.get(k) for k in core_csv_fields} for row in rounds])

    target_cover100 = [r["cover100_ms_target"] for r in rounds if r["cover100_ms_target"] is not None]
    target_cover99 = [r["cover99_ms_target"] for r in rounds if r["cover99_ms_target"] is not None]
    target_cover95 = [r["cover95_ms_target"] for r in rounds if r["cover95_ms_target"] is not None]
    target_cover90 = [r["cover90_ms_target"] for r in rounds if r["cover90_ms_target"] is not None]
    summary = {
        "label": args.label,
        "variant": args.variant,
        "topic_mode": args.topic_mode,
        "payload_mode": args.payload_mode,
        "seq_rule": args.seq_rule,
        "payload_size": args.payload_size,
        "seed": str(args.seed).strip(),
        "request_count": args.request_count if args.request_count > 0 else len(rounds),
        "node_count": args.node_count,
        "shard_count": shard_count,
        "rounds_per_shard": args.rounds_per_shard,
        "round_count": len(rounds),
        "publish_port": default_publish_port,
        "target_cover90_ms": stats(target_cover90),
        "target_cover95_ms": stats(target_cover95),
        "target_cover99_ms": stats(target_cover99),
        "target_cover100_ms": stats(target_cover100),
        "first_seen_ms_target_excl_publisher": stats([r["first_seen_ms_target_excl_publisher"] for r in rounds if r["first_seen_ms_target_excl_publisher"] is not None]),
        "bytes_per_target_delivered_lo": stats([r["bytes_per_target_delivered_lo"] for r in rounds]),
        "bytes_per_global_delivered_lo": stats([r["bytes_per_global_delivered_lo"] for r in rounds]),
        "seen_count_target": stats([r["seen_count_target"] for r in rounds]),
        "seen_count_global": stats([r["seen_count_global"] for r in rounds]),
        "seen_ratio_global": stats([r["seen_ratio_global"] for r in rounds]),
        "spillover_count_global": stats([r["spillover_count_global"] for r in rounds]),
        "spillover_ratio_global": stats([r["spillover_ratio_global"] for r in rounds]),
        "load1": stats([r["load1"] for r in rounds]),
        "global_cover90_ms": stats([r["cover90_ms_global"] for r in rounds if r["cover90_ms_global"] is not None]),
        "global_cover95_ms": stats([r["cover95_ms_global"] for r in rounds if r["cover95_ms_global"] is not None]),
        "global_cover99_ms": stats([r["cover99_ms_global"] for r in rounds if r["cover99_ms_global"] is not None]),
        "global_cover100_ms": stats([r["cover100_ms_global"] for r in rounds if r["cover100_ms_global"] is not None]),
        "core_signal_stats": {
            field: stats([r.get(field) for r in rounds if r.get(field) is not None])
            for field in CORE_SIGNAL_FIELDS
        },
        "result_dir": str(out),
    }
    core_metrics = {
        "profile": {
            "label": args.label,
            "variant": args.variant,
            "topic_mode": args.topic_mode,
            "payload_mode": args.payload_mode,
            "seq_rule": args.seq_rule,
            "payload_size": args.payload_size,
            "seed": str(args.seed).strip(),
            "node_count": args.node_count,
            "request_count": args.request_count if args.request_count > 0 else len(rounds),
        },
        "round_count": len(rounds),
        "core_signal_stats": summary["core_signal_stats"],
        "rounds": [
            {k: row.get(k) for k in ["round_index", "task_id"] + CORE_SIGNAL_FIELDS}
            for row in rounds
        ],
    }
    (out / "summary.json").write_text(json.dumps(summary, indent=2))
    (out / "core_metrics.json").write_text(json.dumps(core_metrics, indent=2))
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
