#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PASSWORD_FILE="${PASSWORD_FILE:-$SCRIPT_DIR/.local/ssh/macmini.password}"
PLACEHOLDER="<FILL_PASSWORD_HERE>"
ASKPASS_SCRIPT="$SCRIPT_DIR/ssh-askpass-macmini.sh"

if [[ ! -f "$PASSWORD_FILE" ]]; then
  echo "Password file not found: $PASSWORD_FILE" >&2
  exit 1
fi

PASSWORD_CONTENT="$(<"$PASSWORD_FILE")"
if [[ -z "${PASSWORD_CONTENT//[[:space:]]/}" || "$PASSWORD_CONTENT" == "$PLACEHOLDER" ]]; then
  echo "Fill in the Mac mini password at $PASSWORD_FILE before connecting." >&2
  exit 1
fi

if [[ ! -x "$ASKPASS_SCRIPT" ]]; then
  echo "Askpass helper is missing or not executable: $ASKPASS_SCRIPT" >&2
  exit 1
fi

export SSH_ASKPASS="$ASKPASS_SCRIPT"
export SSH_ASKPASS_REQUIRE=force
export DISPLAY="${DISPLAY:-codex-local}"

exec ssh -o StrictHostKeyChecking=accept-new \
  -o PreferredAuthentications=keyboard-interactive,password \
  -o PubkeyAuthentication=no \
  mabot@192.168.1.102 "$@"
