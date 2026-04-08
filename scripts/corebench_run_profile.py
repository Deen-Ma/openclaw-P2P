#!/usr/bin/env python3
import argparse
import csv
import json
import math
import shlex
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path


def run(cmd, cwd=None):
    subprocess.run(cmd, cwd=cwd, check=True)


def capture(cmd, cwd=None):
    return subprocess.check_output(cmd, cwd=cwd, text=True).strip()


def normalize_bool_text(raw):
    text = str(raw).strip().lower()
    if text in {"1", "true", "on", "yes"}:
        return "on"
    if text in {"0", "false", "off", "no"}:
        return "off"
    return "on"


def resolve_profile_path(raw_profile, profiles_dir):
    p = Path(raw_profile)
    if p.exists():
        return p.resolve()
    candidate = profiles_dir / f"{raw_profile}.json"
    if candidate.exists():
        return candidate.resolve()
    raise FileNotFoundError(f"profile not found: {raw_profile}")


def apply_overrides(profile, args):
    out = dict(profile)
    if args.node_count is not None:
        out["node_count"] = int(args.node_count)
    if args.request_count is not None:
        out["request_count"] = int(args.request_count)
    if args.payload_size is not None:
        out["payload_size"] = int(args.payload_size)
    if args.topic_mode:
        out["topic_mode"] = args.topic_mode
    if args.payload_mode:
        out["payload_mode"] = args.payload_mode
    if args.seq_rule:
        out["seq_rule"] = normalize_bool_text(args.seq_rule)
    return out


def resolve_variant(profile):
    if profile.get("variant"):
        return profile["variant"], None
    topic_mode = str(profile.get("topic_mode", "topic_aware")).strip()
    payload_mode = str(profile.get("payload_mode", "bodyless")).strip()
    seq_rule = normalize_bool_text(profile.get("seq_rule", "on"))
    if topic_mode == "public" and payload_mode == "full":
        return "baseline_full_payload_real", None
    if topic_mode == "public" and payload_mode == "bodyless":
        return "ablation_no_semantic_public_topic", None
    if topic_mode == "topic_aware" and payload_mode == "bodyless":
        if seq_rule == "off":
            return "openagent_semantic_on", "seq_rule=off is metadata-only in current implementation (effective rule is still on)"
        return "openagent_semantic_on", None
    raise ValueError(f"unsupported profile combination: topic_mode={topic_mode} payload_mode={payload_mode}")


def compute_rounds_per_shard(node_count, request_count):
    shard_count = max(1, node_count // 20)
    if request_count <= 0:
        return 2
    return max(1, math.ceil(request_count / shard_count))


def ssh_cmd(remote, port, password, command):
    return [
        "sshpass", "-p", password,
        "ssh", "-o", "StrictHostKeyChecking=no", "-p", str(port), remote, command,
    ]


def scp_to_remote(remote, port, password, local_path, remote_path):
    run([
        "sshpass", "-p", password,
        "scp", "-P", str(port), "-o", "StrictHostKeyChecking=no",
        str(local_path), f"{remote}:{remote_path}",
    ])


def scp_from_remote(remote, port, password, remote_path, local_path):
    local_path.parent.mkdir(parents=True, exist_ok=True)
    run([
        "sshpass", "-p", password,
        "scp", "-r", "-P", str(port), "-o", "StrictHostKeyChecking=no",
        f"{remote}:{remote_path}", str(local_path),
    ])


def summarize_profiles(run_root, manifest):
    rows = []
    for item in manifest:
        summary_path = Path(item["local_bench_dir"]) / "summary.json"
        if not summary_path.exists():
            continue
        summary = json.loads(summary_path.read_text())
        rows.append({
            "profile": item["profile_name"],
            "variant": summary.get("variant"),
            "node_count": summary.get("node_count"),
            "request_count": summary.get("request_count"),
            "topic_mode": summary.get("topic_mode"),
            "payload_mode": summary.get("payload_mode"),
            "seq_rule": summary.get("seq_rule"),
            "visible_median": (summary.get("core_signal_stats", {}).get("visible", {}) or {}).get("median"),
            "coarse_match_median": (summary.get("core_signal_stats", {}).get("coarse_match", {}) or {}).get("median"),
            "full_match_median": (summary.get("core_signal_stats", {}).get("full_match", {}) or {}).get("median"),
            "candidate_return_median": (summary.get("core_signal_stats", {}).get("candidate_return", {}) or {}).get("median"),
            "state_terminal_median": (summary.get("core_signal_stats", {}).get("state_terminal", {}) or {}).get("median"),
            "cover95_target_ms_median": (summary.get("target_cover95_ms", {}) or {}).get("median"),
            "cover99_target_ms_median": (summary.get("target_cover99_ms", {}) or {}).get("median"),
            "cover95_global_ms_median": (summary.get("global_cover95_ms", {}) or {}).get("median"),
            "cover99_global_ms_median": (summary.get("global_cover99_ms", {}) or {}).get("median"),
            "seed": item.get("seed"),
            "case_label": item.get("case_label", ""),
            "local_bench_dir": item.get("local_bench_dir", ""),
            "result_dir": summary.get("result_dir"),
        })
    summary_csv = run_root / "profile_summary.csv"
    if rows:
        fieldnames = list(rows[0].keys())
        with summary_csv.open("w", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(rows)
    return summary_csv


def main():
    ap = argparse.ArgumentParser(description="Run OpenAgent core-loop benchmark profiles")
    ap.add_argument("--repo", default=str(Path(__file__).resolve().parents[1]))
    ap.add_argument("--profiles-dir", default="")
    ap.add_argument("--profile", action="append", default=[], help="profile file path or profile name (repeatable)")
    ap.add_argument("--profiles", default="", help="comma-separated profile names")
    ap.add_argument("--output-root", default="", help="local results root")
    ap.add_argument("--remote", default="ma@172.31.100.142")
    ap.add_argument("--port", default="30002")
    ap.add_argument("--password", default="ma123")
    ap.add_argument("--remote-sudo-password", default="ma123")
    ap.add_argument("--remote-tools-dir", default="~/corebench_tools")
    ap.add_argument("--remote-base", default="/home/ma/openagent-corebench")
    ap.add_argument("--node-count", type=int, default=None)
    ap.add_argument("--request-count", type=int, default=None)
    ap.add_argument("--payload-size", type=int, default=None)
    ap.add_argument("--topic-mode", choices=["public", "topic_aware"], default="")
    ap.add_argument("--payload-mode", choices=["full", "bodyless"], default="")
    ap.add_argument("--seq-rule", choices=["on", "off"], default="")
    ap.add_argument("--run-id", default="", help="fixed run id under output-root; defaults to current UTC timestamp")
    ap.add_argument("--case-label", default="", help="case label appended to profile output path")
    ap.add_argument("--seed", default="", help="run seed metadata")
    ap.add_argument("--observe-sec", type=int, default=60)
    ap.add_argument("--round-gap-sec", type=float, default=10.0)
    args = ap.parse_args()

    repo = Path(args.repo).resolve()
    profiles_dir = Path(args.profiles_dir).resolve() if args.profiles_dir else repo / "configs" / "core_profiles"
    requested = list(args.profile)
    if args.profiles:
        requested.extend([x.strip() for x in args.profiles.split(",") if x.strip()])
    if not requested:
        raise SystemExit("missing --profile/--profiles")

    timestamp = args.run_id.strip() if args.run_id.strip() else datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    output_root = Path(args.output_root).resolve() if args.output_root else repo / "results" / "corebench_profiles"
    run_root = output_root / timestamp
    run_root.mkdir(parents=True, exist_ok=True)
    manifest_path = run_root / "manifest.json"
    manifest = []
    if manifest_path.exists():
        try:
            manifest = json.loads(manifest_path.read_text())
        except json.JSONDecodeError:
            manifest = []

    scripts = [
        repo / "scripts" / "scalebench_remote_cluster.sh",
        repo / "scripts" / "scalebench_remote_bench.py",
    ]
    run(ssh_cmd(args.remote, args.port, args.password, f"mkdir -p {shlex.quote(args.remote_tools_dir)}"))
    for script in scripts:
        scp_to_remote(args.remote, args.port, args.password, script, f"{args.remote_tools_dir}/{script.name}")
    run(ssh_cmd(args.remote, args.port, args.password, f"chmod +x {args.remote_tools_dir}/scalebench_remote_cluster.sh {args.remote_tools_dir}/scalebench_remote_bench.py"))

    for raw_profile in requested:
        profile_path = resolve_profile_path(raw_profile, profiles_dir)
        profile_data = json.loads(profile_path.read_text())
        effective = apply_overrides(profile_data, args)
        if not effective.get("name"):
            effective["name"] = profile_path.stem

        node_count = int(effective.get("node_count", 40))
        if node_count < 20 or (node_count % 20) != 0:
            raise SystemExit(f"profile {effective['name']} has invalid node_count={node_count}, must be >=20 and multiple of 20")
        request_count = int(effective.get("request_count", 20))
        if request_count <= 0:
            raise SystemExit(f"profile {effective['name']} has invalid request_count={request_count}")
        payload_size = int(effective.get("payload_size", 0))
        if payload_size < 0:
            raise SystemExit(f"profile {effective['name']} has invalid payload_size={payload_size}")
        effective["seq_rule"] = normalize_bool_text(effective.get("seq_rule", "on"))
        variant, warning = resolve_variant(effective)
        rounds_per_shard = compute_rounds_per_shard(node_count, request_count)

        profile_name = effective["name"]
        case_label = args.case_label.strip()
        profile_key = profile_name if not case_label else f"{profile_name}__{case_label}"
        local_profile_root = run_root / profile_key
        local_profile_root.mkdir(parents=True, exist_ok=True)
        remote_root = f"{args.remote_base}/{timestamp}/{profile_key}"

        effective_out = {
            **effective,
            "variant": variant,
            "rounds_per_shard": rounds_per_shard,
            "remote_root": remote_root,
            "profile_path": str(profile_path),
            "case_label": case_label,
            "seed": args.seed.strip(),
        }
        if warning:
            effective_out["warning"] = warning
        (local_profile_root / "profile.effective.json").write_text(json.dumps(effective_out, indent=2))

        print(f"[corebench] build cluster profile={profile_name} variant={variant} nodes={node_count}")
        run(ssh_cmd(
            args.remote,
            args.port,
            args.password,
            " ".join([
                f"SCALEBENCH_SUDO_PW={shlex.quote(args.remote_sudo_password)}",
                f"{args.remote_tools_dir}/scalebench_remote_cluster.sh",
                f"{shlex.quote(variant)}",
                f"{shlex.quote(remote_root)}",
                f"{node_count}",
            ]),
        ))

        topic_mode = str(effective.get("topic_mode", "topic_aware")).strip()
        payload_mode = str(effective.get("payload_mode", "bodyless")).strip()
        seq_rule = effective["seq_rule"]
        print(f"[corebench] run bench profile={profile_name} requests={request_count} topic_mode={topic_mode} payload_mode={payload_mode} seq_rule={seq_rule}")
        run(ssh_cmd(
            args.remote,
            args.port,
            args.password,
            " ".join([
                f"SCALEBENCH_SUDO_PW={shlex.quote(args.remote_sudo_password)}",
                f"{args.remote_tools_dir}/scalebench_remote_bench.py",
                f"--root {shlex.quote(remote_root)}",
                f"--label {shlex.quote(profile_name)}",
                f"--variant {shlex.quote(variant)}",
                f"--node-count {node_count}",
                f"--rounds-per-shard {rounds_per_shard}",
                f"--request-count {request_count}",
                f"--topic-mode {shlex.quote(topic_mode)}",
                f"--payload-mode {shlex.quote(payload_mode)}",
                f"--seq-rule {shlex.quote(seq_rule)}",
                f"--payload-size {payload_size}",
                f"--event-log-root {shlex.quote(remote_root)}",
                f"--observe-sec {int(args.observe_sec)}",
                f"--round-gap-sec {float(args.round_gap_sec)}",
                f"--seed {shlex.quote(args.seed.strip())}",
            ])
        ))

        remote_bench_dir = capture(ssh_cmd(args.remote, args.port, args.password, f"ls -td {shlex.quote(remote_root)}/bench_shards_* | head -1"))
        scp_from_remote(args.remote, args.port, args.password, f"{remote_root}/observability", local_profile_root / "observability")
        scp_from_remote(args.remote, args.port, args.password, remote_bench_dir, local_profile_root / Path(remote_bench_dir).name)
        local_bench_dir = local_profile_root / Path(remote_bench_dir).name

        manifest.append({
            "profile_name": profile_name,
            "profile_file": str(profile_path),
            "variant": variant,
            "node_count": node_count,
            "request_count": request_count,
            "payload_size": payload_size,
            "topic_mode": topic_mode,
            "payload_mode": payload_mode,
            "seq_rule": seq_rule,
            "rounds_per_shard": rounds_per_shard,
            "remote_root": remote_root,
            "remote_bench_dir": remote_bench_dir,
            "local_root": str(local_profile_root),
            "local_bench_dir": str(local_bench_dir),
            "warning": warning or "",
            "seed": args.seed.strip(),
            "case_label": case_label,
            "profile_key": profile_key,
        })
        manifest_path.write_text(json.dumps(manifest, indent=2))

    summary_csv = summarize_profiles(run_root, manifest)
    print(f"[corebench] done run_root={run_root}")
    print(f"[corebench] manifest={run_root / 'manifest.json'}")
    print(f"[corebench] summary_csv={summary_csv}")


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as exc:
        print(f"[corebench] command failed: {exc}", file=sys.stderr)
        sys.exit(exc.returncode)
