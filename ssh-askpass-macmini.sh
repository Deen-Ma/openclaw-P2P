#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PASSWORD_FILE="${PASSWORD_FILE:-$SCRIPT_DIR/.local/ssh/macmini.password}"

if [[ ! -f "$PASSWORD_FILE" ]]; then
  exit 1
fi

IFS= read -r password < "$PASSWORD_FILE" || true
printf '%s' "$password"

