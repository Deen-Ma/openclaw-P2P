#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/host-config.sh"

CONNECT_HOST_SCRIPT="$SCRIPT_DIR/connect-host.sh"
HOSTS_FILE="${HOSTS_FILE:-$SCRIPT_DIR/.local/hosts.json}"
OUTPUT_ROOT="${OUTPUT_ROOT:-$SCRIPT_DIR/.local/evidence/p2p-cli}"
MODE="auto"

usage() {
  cat <<'USAGE'
Usage: ./collect-p2p-cli-evidence.sh [--mode auto|direct|ssh] [--hosts-file path] [--output-dir path] <host-key> [host-key...]

Collects OpenAgent API snapshots for each host key:
  /healthz
  /v1/tasks
  /v1/facts
  /v1/peers
  /v1/sessions

Modes:
  auto   Try direct HTTP first, then SSH fallback via ./connect-host.sh (default)
  direct Use direct HTTP only (http://<host>:<openagentApiPort>)
  ssh    Use SSH only and curl remote 127.0.0.1:<openagentApiPort>

Outputs:
  .local/evidence/p2p-cli/<timestamp>/
USAGE
}

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    host_config_fail "Required command not found: $name"
  fi
}

endpoint_to_file() {
  local endpoint="$1"
  printf '%s.json' "$(echo "${endpoint#/}" | tr '/' '-')"
}

fetch_direct() {
  local host_addr="$1"
  local api_port="$2"
  local endpoint="$3"
  curl -fsS --max-time 10 "http://${host_addr}:${api_port}${endpoint}"
}

fetch_ssh() {
  local host_key="$1"
  local api_port="$2"
  local endpoint="$3"
  local remote_cmd

  remote_cmd="$(printf "curl -fsS --max-time 10 'http://127.0.0.1:%s%s'" "$api_port" "$endpoint")"
  "$CONNECT_HOST_SCRIPT" "$host_key" "$remote_cmd"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      [[ $# -ge 2 ]] || host_config_fail "--mode requires a value"
      MODE="$2"
      shift 2
      ;;
    --hosts-file)
      [[ $# -ge 2 ]] || host_config_fail "--hosts-file requires a value"
      HOSTS_FILE="$2"
      shift 2
      ;;
    --output-dir)
      [[ $# -ge 2 ]] || host_config_fail "--output-dir requires a value"
      OUTPUT_ROOT="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      host_config_fail "Unknown option: $1"
      ;;
    *)
      break
      ;;
  esac
done

case "$MODE" in
  auto|direct|ssh)
    ;;
  *)
    host_config_fail "Unsupported mode '$MODE'. Use auto, direct, or ssh."
    ;;
esac

require_cmd jq
require_cmd curl

if [[ "$MODE" != "direct" && ! -x "$CONNECT_HOST_SCRIPT" ]]; then
  host_config_fail "SSH collector mode requires executable script: $CONNECT_HOST_SCRIPT"
fi

host_config_validate_file "$HOSTS_FILE"
export HOSTS_FILE

declare -a HOST_KEYS=()
if [[ $# -gt 0 ]]; then
  HOST_KEYS=("$@")
else
  default_host="$(host_config_default_host "$HOSTS_FILE")"
  if [[ -z "${default_host//[[:space:]]/}" ]]; then
    usage
    host_config_fail "No host keys provided and defaultHost is empty in $HOSTS_FILE."
  fi
  HOST_KEYS=("$default_host")
fi

timestamp="$(date '+%Y%m%d-%H%M%S')"
run_dir="$OUTPUT_ROOT/$timestamp"
mkdir -p "$run_dir"

{
  printf '%s\n' "${HOST_KEYS[@]}" | jq -R . | jq -s \
    --arg created_at "$(date '+%Y-%m-%d %H:%M:%S %z')" \
    --arg mode "$MODE" \
    --arg hosts_file "$HOSTS_FILE" \
    --arg output_dir "$run_dir" \
    '{created_at:$created_at,mode:$mode,hosts_file:$hosts_file,output_dir:$output_dir,hosts:.}'
} >"$run_dir/metadata.json"

summary_file="$run_dir/summary.tsv"
printf 'host\tendpoint\tstatus\tmethod\toutput\terror\n' >"$summary_file"

endpoints=(
  "/healthz"
  "/v1/tasks"
  "/v1/facts"
  "/v1/peers"
  "/v1/sessions"
)

failed_fetches=0

for host_key in "${HOST_KEYS[@]}"; do
  host_config_require_host "$HOSTS_FILE" "$host_key"

  host_addr="$(host_config_read_field "$HOSTS_FILE" "$host_key" "host")"
  api_port="$(host_config_read_field "$HOSTS_FILE" "$host_key" "openagentApiPort" "openagent_api_port")"
  profile="$(host_config_read_field "$HOSTS_FILE" "$host_key" "profile")"

  host_dir="$run_dir/$host_key"
  err_dir="$host_dir/errors"
  mkdir -p "$host_dir" "$err_dir"

  jq -n \
    --arg host_key "$host_key" \
    --arg profile "$profile" \
    --arg host "$host_addr" \
    --arg api_port "$api_port" \
    '{host_key:$host_key,profile:$profile,host:$host,api_port:($api_port|tonumber)}' \
    >"$host_dir/host.json"

  for endpoint in "${endpoints[@]}"; do
    out_name="$(endpoint_to_file "$endpoint")"
    output_file="$host_dir/$out_name"
    error_file="$err_dir/${out_name}.err"

    method="none"
    fetch_ok=1

    if [[ "$MODE" == "auto" || "$MODE" == "direct" ]]; then
      if fetch_direct "$host_addr" "$api_port" "$endpoint" >"$output_file" 2>"$error_file"; then
        method="direct"
        fetch_ok=0
      fi
    fi

    if [[ $fetch_ok -ne 0 && ( "$MODE" == "auto" || "$MODE" == "ssh" ) ]]; then
      if fetch_ssh "$host_key" "$api_port" "$endpoint" >"$output_file" 2>"$error_file"; then
        method="ssh"
        fetch_ok=0
      fi
    fi

    if [[ $fetch_ok -eq 0 ]]; then
      rm -f "$error_file"
      printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$host_key" "$endpoint" "ok" "$method" "$output_file" "" >>"$summary_file"
    else
      failed_fetches=$((failed_fetches + 1))
      printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$host_key" "$endpoint" "failed" "$method" "$output_file" "$error_file" >>"$summary_file"
    fi
  done
done

echo "Evidence collected to: $run_dir"
echo "Summary: $summary_file"

if [[ $failed_fetches -gt 0 ]]; then
  echo "Completed with $failed_fetches failed fetch(es). Check $summary_file for details." >&2
  exit 1
fi

echo "All endpoint snapshots collected successfully."
