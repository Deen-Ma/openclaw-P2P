#!/usr/bin/env python3
import argparse
import json
import os
import subprocess
import time
from pathlib import Path


def run(cmd, cwd=None, timeout=None, quiet=False):
    kwargs = {}
    if quiet:
        kwargs["stdout"] = subprocess.DEVNULL
        kwargs["stderr"] = subprocess.DEVNULL
    subprocess.run(cmd, cwd=cwd, check=True, timeout=timeout, **kwargs)


def load_manifest(path):
    if not path.exists():
        return []
    return json.loads(path.read_text())


def build_cases(nodes, seeds, profiles):
    out = []
    for n in nodes:
        for idx, seed in enumerate(seeds, start=1):
            case = f"formal_n{n}_r{idx}"
            for profile in profiles:
                out.append((case, profile, n, seed))
    return out


def done_set(manifest_rows):
    return {
        (row.get("case_label"), row.get("profile_name"))
        for row in manifest_rows
        if str(row.get("case_label", "")).startswith("formal_")
    }


def log_line(path, msg):
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    line = f"[{ts}] {msg}"
    print(line, flush=True)
    with path.open("a", encoding="utf-8") as fh:
        fh.write(line + "\n")


def remote_cleanup(repo, remote, port, password, remote_sudo_password):
    cmd = [
        "sshpass",
        "-p",
        password,
        "ssh",
        "-o",
        "StrictHostKeyChecking=no",
        "-p",
        str(port),
        remote,
        "printf '"
        + remote_sudo_password
        + "\\n' | sudo -S bash -lc 'docker ps -aq --filter name=oa-formal- | xargs -r docker rm -f >/dev/null'",
    ]
    try:
        run(cmd, cwd=repo, timeout=90, quiet=True)
        return True
    except Exception:
        return False


def main():
    ap = argparse.ArgumentParser(description="Resume missing formal corebench runs (single-instance)")
    ap.add_argument("--repo", default=str(Path(__file__).resolve().parents[1]))
    ap.add_argument("--run-id", required=True)
    ap.add_argument("--remote", default="ma@172.31.100.142")
    ap.add_argument("--port", default="30002")
    ap.add_argument("--password", default="ma123")
    ap.add_argument("--remote-sudo-password", default="ma123")
    ap.add_argument("--remote-tools-dir", default="/home/ma/corebench_tools")
    ap.add_argument("--remote-base", default="/home/ma/openagent-corebench")
    ap.add_argument("--request-count", type=int, default=3)
    ap.add_argument("--payload-size", type=int, default=1024)
    ap.add_argument("--observe-sec", type=int, default=15)
    ap.add_argument("--round-gap-sec", type=float, default=2.0)
    ap.add_argument("--attempts", type=int, default=2)
    ap.add_argument("--timeout-sec", type=int, default=2700)
    args = ap.parse_args()

    repo = Path(args.repo).resolve()
    run_root = repo / "results" / "corebench_profiles" / args.run_id
    manifest_path = run_root / "manifest.json"
    log_path = run_root / "resume_runner.log"
    fail_path = run_root / "resume_failures.log"
    lock_path = run_root / ".resume_missing.lock"

    profiles = ["public_full_payload", "public_bodyless", "topic_aware_bodyless"]
    nodes = [40, 60, 80, 100, 120]
    seeds = ["20260411", "20260412", "20260413"]

    fd = None
    try:
        fd = os.open(str(lock_path), os.O_CREAT | os.O_EXCL | os.O_WRONLY)
        os.write(fd, str(os.getpid()).encode())
    except FileExistsError:
        raise SystemExit(f"lock exists: {lock_path}")

    try:
        manifest = load_manifest(manifest_path)
        missing = [c for c in build_cases(nodes, seeds, profiles) if (c[0], c[1]) not in done_set(manifest)]
        log_line(log_path, f"resume_missing start missing={len(missing)}")

        for idx, (case, profile, node, seed) in enumerate(missing, start=1):
            manifest = load_manifest(manifest_path)
            if (case, profile) in done_set(manifest):
                log_line(log_path, f"skip already-done {case} {profile}")
                continue

            success = False
            for attempt in range(1, args.attempts + 1):
                log_line(
                    log_path,
                    f"[{idx}/{len(missing)}] run {case} {profile} node={node} seed={seed} attempt={attempt}",
                )
                remote_cleanup(repo, args.remote, args.port, args.password, args.remote_sudo_password)

                cmd = [
                    "python3",
                    "scripts/corebench_run_profile.py",
                    "--run-id",
                    args.run_id,
                    "--case-label",
                    case,
                    "--seed",
                    seed,
                    "--remote",
                    args.remote,
                    "--port",
                    str(args.port),
                    "--password",
                    args.password,
                    "--remote-sudo-password",
                    args.remote_sudo_password,
                    "--remote-tools-dir",
                    args.remote_tools_dir,
                    "--remote-base",
                    args.remote_base,
                    "--profile",
                    profile,
                    "--node-count",
                    str(node),
                    "--request-count",
                    str(args.request_count),
                    "--payload-size",
                    str(args.payload_size),
                    "--observe-sec",
                    str(args.observe_sec),
                    "--round-gap-sec",
                    str(args.round_gap_sec),
                ]
                try:
                    run(cmd, cwd=repo, timeout=args.timeout_sec)
                    log_line(log_path, f"ok {case} {profile} attempt={attempt}")
                    success = True
                    break
                except subprocess.TimeoutExpired:
                    log_line(log_path, f"timeout {case} {profile} attempt={attempt}")
                except subprocess.CalledProcessError as exc:
                    log_line(log_path, f"fail rc={exc.returncode} {case} {profile} attempt={attempt}")

            if not success:
                with fail_path.open("a", encoding="utf-8") as fh:
                    fh.write(f"{case},{profile},{node},{seed},failed_after_retries_resume_missing\n")

        log_line(log_path, "resume_missing finished")
    finally:
        try:
            if fd is not None:
                os.close(fd)
            if lock_path.exists():
                lock_path.unlink()
        except Exception:
            pass


if __name__ == "__main__":
    main()
