#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "usage: $0 <variant> <root> <total>" >&2
  exit 1
fi

VARIANT="$1"
ROOT="$2"
TOTAL="$3"
OBS="$ROOT/observability"
GENERAL_TOPIC="openagent/v1/general"
RENDEZVOUS="openagent/v1/dev"
P2P_BASE=5500
API_BASE=8500
SUDO_PW="${SCALEBENCH_SUDO_PW:-ma123}"
MSG_SIZE_LIMIT="${SCALEBENCH_MSG_SIZE_LIMIT:-4096}"

if [ "$TOTAL" -lt 20 ] || [ $((TOTAL % 20)) -ne 0 ]; then
  echo "total must be a multiple of 20 and >= 20" >&2
  exit 1
fi

SEEDS=16
if [ "$TOTAL" -lt "$SEEDS" ]; then
  SEEDS="$TOTAL"
fi
SHARDS=$((TOTAL / 20))
PER_SHARD_SEEDS=4
MIN_REQUIRED_SEEDS=$((SHARDS * PER_SHARD_SEEDS))
if [ "$SEEDS" -lt "$MIN_REQUIRED_SEEDS" ] && [ "$MIN_REQUIRED_SEEDS" -le "$TOTAL" ]; then
  SEEDS="$MIN_REQUIRED_SEEDS"
fi

if [ "$TOTAL" -le 100 ]; then
  BATCH_SIZE=20
  BATCH_SLEEP=10
elif [ "$TOTAL" -le 200 ]; then
  BATCH_SIZE=25
  BATCH_SLEEP=15
elif [ "$TOTAL" -le 300 ]; then
  BATCH_SIZE=30
  BATCH_SLEEP=20
elif [ "$TOTAL" -le 400 ]; then
  BATCH_SIZE=30
  BATCH_SLEEP=30
else
  BATCH_SIZE=30
  BATCH_SLEEP=45
fi

mkdir -p "$OBS"

wait_healthz() {
  local api="$1"
  for _ in $(seq 1 180); do
    if curl -sf "http://127.0.0.1:${api}/healthz" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}

shard_topic() {
  local idx="$1"
  local shard=$(( (idx - 1) / 20 + 1 ))
  printf 'openagent/v1/bench/shard-%03d' "$shard"
}

mkboot() {
  local api="$1"
  local p2p="$2"
  wait_healthz "$api"
  curl -sf "http://127.0.0.1:${api}/healthz" | python3 -c "import sys,json; d=json.load(sys.stdin); print('/ip4/127.0.0.1/tcp/${p2p}/p2p/' + d['peer_id'])"
}

start_node() {
  local idx="$1"; shift || true
  local p2p=$((P2P_BASE + idx - 1))
  local api=$((API_BASE + idx - 1))
  local prof="formal${idx}"
  local data="$ROOT/node${idx}"
  local shard
  shard="$(shard_topic "$idx")"
  mkdir -p "$data"
  # Guard against stale or duplicate container names from interrupted runs.
  printf '%s\n' "$SUDO_PW" | sudo -S docker rm -f "oa-formal-${idx}" >/dev/null 2>&1 || true
  printf '%s\n' "$SUDO_PW" | sudo -S docker run -d \
    --name "oa-formal-${idx}" \
    --network host \
    -v "$data:/data" \
    openagent-node:formal \
    --profile "$prof" \
    --data-dir /data \
    --port "$p2p" \
    --api-port "$api" \
    --rendezvous "$RENDEZVOUS" \
    --msg-size-limit "$MSG_SIZE_LIMIT" \
    --experiment-variant "$VARIANT" \
    --topic "$GENERAL_TOPIC" \
    --topic "$shard" \
    "$@" >/dev/null
}

printf '%s\n' "$SUDO_PW" | sudo -S bash -lc 'docker ps -aq --filter name=oa-formal- | xargs -r docker rm -f'
printf '%s\n' "$SUDO_PW" | sudo -S rm -rf "$ROOT"
mkdir -p "$OBS"

BOOTSTRAPS=()
SEED_INDEXES=()
for offset in $(seq 0 $((PER_SHARD_SEEDS - 1))); do
  for shard in $(seq 0 $((SHARDS - 1))); do
    idx=$((shard * 20 + offset + 1))
    if [ "$idx" -le "$TOTAL" ]; then
      SEED_INDEXES+=("$idx")
      if [ "${#SEED_INDEXES[@]}" -ge "$SEEDS" ]; then
        break 2
      fi
    fi
  done
done
for offset in $(seq "$PER_SHARD_SEEDS" 19); do
  for shard in $(seq 0 $((SHARDS - 1))); do
    idx=$((shard * 20 + offset + 1))
    if [ "$idx" -le "$TOTAL" ]; then
      SEED_INDEXES+=("$idx")
      if [ "${#SEED_INDEXES[@]}" -ge "$SEEDS" ]; then
        break 2
      fi
    fi
  done
done

SEEDED=()
for idx in "${SEED_INDEXES[@]}"; do
  args=()
  if [ "${#BOOTSTRAPS[@]}" -gt 0 ]; then
    for b in "${BOOTSTRAPS[@]}"; do
      args+=(--bootstrap "$b")
    done
  fi
  start_node "$idx" "${args[@]}"
  SEEDED+=("$idx")
  boot="$(mkboot $((API_BASE + idx - 1)) $((P2P_BASE + idx - 1)))"
  BOOTSTRAPS+=("$boot")
  sleep 1
done
printf '%s\n' "${BOOTSTRAPS[@]}" > "$OBS/bootstrap_multiaddrs.txt"

NON_SEEDS=()
for idx in $(seq 1 "$TOTAL"); do
  is_seed=0
  for seeded in "${SEEDED[@]}"; do
    if [ "$seeded" -eq "$idx" ]; then
      is_seed=1
      break
    fi
  done
  if [ "$is_seed" -eq 0 ]; then
    NON_SEEDS+=("$idx")
  fi
done

for ((offset=0; offset<${#NON_SEEDS[@]}; offset+=BATCH_SIZE)); do
  batch_end=$((offset + BATCH_SIZE))
  if [ "$batch_end" -gt "${#NON_SEEDS[@]}" ]; then
    batch_end="${#NON_SEEDS[@]}"
  fi
  for ((i=offset; i<batch_end; i++)); do
    idx="${NON_SEEDS[$i]}"
    args=()
    for b in "${BOOTSTRAPS[@]}"; do
      args+=(--bootstrap "$b")
    done
    start_node "$idx" "${args[@]}"
    sleep 0.2
  done
  sleep "$BATCH_SLEEP"
done

health_ok=0
for api in $(seq "$API_BASE" $((API_BASE + TOTAL - 1))); do
  if wait_healthz "$api"; then
    health_ok=$((health_ok + 1))
  fi
done

printf '%s\n' "$health_ok" > "$OBS/healthz_ok_count.txt"
printf '%s\n' "$SUDO_PW" | sudo -S docker ps --format '{{.Names}}' | grep -c '^oa-formal-' > "$OBS/running_count.txt"

python3 - <<PY > "$OBS/summary.json"
import json
import pathlib

base = pathlib.Path(r"$OBS")
running = int((base / "running_count.txt").read_text().strip())
health = int((base / "healthz_ok_count.txt").read_text().strip())
print(json.dumps({
    "variant": "$VARIANT",
    "node_count": $TOTAL,
    "running_count": running,
    "healthz_ok_count": health,
    "missing_count": max(0, $TOTAL - running),
    "missing_nodes": []
}, indent=2))
PY
