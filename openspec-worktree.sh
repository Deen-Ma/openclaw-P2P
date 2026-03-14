#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$SCRIPT_DIR"
PROPOSALS_DIR="$REPO_DIR/openspec/proposals"

usage() {
  cat <<'EOF'
Usage:
  ./openspec-worktree.sh list
  ./openspec-worktree.sh board
  ./openspec-worktree.sh show <proposal-id>
  ./openspec-worktree.sh create [--force] <proposal-id>
  ./openspec-worktree.sh batch-create [--force] <proposal-id> [proposal-id...]
  ./openspec-worktree.sh prompt <proposal-id>

Examples:
  ./openspec-worktree.sh list
  ./openspec-worktree.sh board
  ./openspec-worktree.sh show 000
  ./openspec-worktree.sh create 000
  ./openspec-worktree.sh create --force 003
  ./openspec-worktree.sh batch-create 000 003
  ./openspec-worktree.sh batch-create --force 003 005
  ./openspec-worktree.sh prompt 000
EOF
}

die() {
  echo "Error: $*" >&2
  exit 1
}

require_git_repo() {
  git -C "$REPO_DIR" rev-parse --show-toplevel >/dev/null 2>&1 || die "not inside a git repository: $REPO_DIR"
}

strip_backticks() {
  local value="$1"
  value="${value#\`}"
  value="${value%\`}"
  printf '%s\n' "$value"
}

status_value() {
  local status_file="$1"
  local key="$2"
  sed -n "s/^- ${key}: \\(.*\\)$/\\1/p" "$status_file" | head -n 1
}

title_from_spec() {
  local spec_file="$1"
  sed -n '1s/^# //p' "$spec_file"
}

proposal_dir_stream() {
  find "$PROPOSALS_DIR" -mindepth 1 -maxdepth 1 -type d ! -name '_*' -print0 | sort -z
}

resolve_proposal_dir() {
  local selector="$1"
  local matches=()

  if [[ -d "$PROPOSALS_DIR/$selector" ]]; then
    printf '%s\n' "$PROPOSALS_DIR/$selector"
    return
  fi

  while IFS= read -r -d '' dir; do
    matches+=("$dir")
  done < <(find "$PROPOSALS_DIR" -mindepth 1 -maxdepth 1 -type d ! -name '_*' -name "${selector}*" -print0 | sort -z)

  case "${#matches[@]}" in
    0)
      die "proposal not found: $selector"
      ;;
    1)
      printf '%s\n' "${matches[0]}"
      ;;
    *)
      echo "Error: multiple proposals match '$selector':" >&2
      for dir in "${matches[@]}"; do
        echo "  - $(basename "$dir")" >&2
      done
      exit 1
      ;;
  esac
}

resolve_path_from_repo() {
  local path_value="$1"
  if [[ "$path_value" == /* ]]; then
    printf '%s\n' "$path_value"
    return
  fi

  local path_dir path_base
  path_dir="$(dirname "$path_value")"
  path_base="$(basename "$path_value")"
  (
    cd "$REPO_DIR"
    cd "$path_dir"
    printf '%s/%s\n' "$PWD" "$path_base"
  )
}

branch_exists() {
  local branch="$1"
  git -C "$REPO_DIR" show-ref --verify --quiet "refs/heads/$branch"
}

worktree_registered() {
  local target_path="$1"
  git -C "$REPO_DIR" worktree list --porcelain | awk '/^worktree / { print substr($0, 10) }' | grep -Fx -- "$target_path" >/dev/null
}

ensure_spec_paths_committed() {
  local dirty
  dirty="$(git -C "$REPO_DIR" status --porcelain -- openspec openspec-worktree.sh 2>/dev/null || true)"
  if [[ -n "${dirty//[[:space:]]/}" ]]; then
    cat <<'EOF' >&2
Error: openspec/worktree tooling has uncommitted changes.

Creating a new git worktree from the current HEAD right now would produce a
worktree that does not contain the latest proposal files and helper scripts.

Commit or stash the current openspec changes first, then run create again.
Relevant paths:
- openspec/
- openspec-worktree.sh
EOF
    exit 1
  fi
}

print_proposal_table() {
  printf '%-4s %-10s %-34s %s\n' "ID" "State" "Proposal" "Suggested Worktree"
  local dir status_file state worktree_rel title
  while IFS= read -r -d '' dir; do
    status_file="$dir/status.md"
    state="$(status_value "$status_file" "state")"
    worktree_rel="$(strip_backticks "$(status_value "$status_file" "suggested_worktree")")"
    title="$(title_from_spec "$dir/spec.md")"
    printf '%-4s %-10s %-34s %s\n' "$(basename "$dir" | cut -d- -f1)" "$state" "$title" "$worktree_rel"
  done < <(proposal_dir_stream)
}

print_board() {
  cat <<'EOF'
Terminal  Mode       Proposal  Modules   Suggested Worktree               Start
T1        control    main      all       current repo                      now
T2        execute    000       B,C,D,E,F ../wt-000-adapter-baseline        now
T3        execute    003       A         ../wt-003-tasks-api-stability     optional parallel line
T4        execute    001       C,D       ../wt-001-detail-handle-rework    after 000
T5        execute    002       C,D,F     ../wt-002-session-isolation-rework after 000
T6        execute    004       F,G       ../wt-004-tg2-regression          after 000,001,002
T7        execute    005       A,G       ../wt-005-cross-node-e2e          after 003 and active

Module legend:
  A = Go OpenAgent Core
  B = Adapter Orchestration
  C = OpenClaw Tool Plugin
  D = Local State Store
  E = Telegram Bridge Runtime
  F = Agent Workspace Templates
  G = Acceptance and Ops Tooling

Recommended first wave:
  000-adapter-workspace-baseline

Optional third parallel line:
  003-tasks-api-stability (promote to active first, or create with --force)
EOF
}

show_proposal() {
  local proposal_dir="$1"
  local proposal_name status_file state branch worktree_rel worktree_abs title dependencies modules

  proposal_name="$(basename "$proposal_dir")"
  status_file="$proposal_dir/status.md"
  state="$(status_value "$status_file" "state")"
  branch="$(strip_backticks "$(status_value "$status_file" "suggested_branch")")"
  worktree_rel="$(strip_backticks "$(status_value "$status_file" "suggested_worktree")")"
  worktree_abs="$(resolve_path_from_repo "$worktree_rel")"
  dependencies="$(status_value "$status_file" "dependencies")"
  modules="$(status_value "$status_file" "modules")"
  title="$(title_from_spec "$proposal_dir/spec.md")"

  cat <<EOF
proposal: $proposal_name
title: $title
state: $state
modules: $modules
dependencies: $dependencies
branch: $branch
worktree: $worktree_abs
spec: $proposal_dir/spec.md
tasks: $proposal_dir/tasks.md
acceptance: $proposal_dir/acceptance.md
status: $proposal_dir/status.md
EOF
}

print_prompt() {
  local proposal_dir="$1"
  local proposal_name spec_rel

  proposal_name="$(basename "$proposal_dir")"
  spec_rel="openspec/proposals/$proposal_name"

  cat <<EOF
Implement \`$spec_rel\`.

Read these files first:
- \`$spec_rel/spec.md\`
- \`$spec_rel/tasks.md\`
- \`$spec_rel/acceptance.md\`
- \`$spec_rel/status.md\`

Execution rules:
- Only implement this proposal.
- Do not mix unrelated fixes into this worktree.
- Keep changes inside the proposal scope unless a strictly required interface fix is unavoidable.
- Run the relevant tests and manual checks from \`acceptance.md\`.
- Update \`status.md\` when the proposal is complete.
- If you discover follow-on work, split it into a new proposal instead of absorbing it here.
EOF
}

create_worktree() {
  local proposal_dir="$1"
  local force_create="${2:-0}"
  local proposal_name status_file state branch worktree_rel worktree_abs

  proposal_name="$(basename "$proposal_dir")"
  status_file="$proposal_dir/status.md"
  state="$(status_value "$status_file" "state")"
  branch="$(strip_backticks "$(status_value "$status_file" "suggested_branch")")"
  worktree_rel="$(strip_backticks "$(status_value "$status_file" "suggested_worktree")")"
  worktree_abs="$(resolve_path_from_repo "$worktree_rel")"

  if [[ -z "${branch//[[:space:]]/}" || -z "${worktree_rel//[[:space:]]/}" ]]; then
    die "proposal $proposal_name is missing suggested_branch or suggested_worktree"
  fi

  if [[ "$state" == "completed" ]]; then
    die "proposal $proposal_name is already marked completed"
  fi

  if [[ "$state" != "active" && "$force_create" != "1" ]]; then
    cat <<EOF >&2
Error: proposal $proposal_name is in state '$state', not 'active'.

Create worktrees only for active proposals by default.
Next steps:
- finish the proposal spec and change state to active, or
- use an explicit override: ./openspec-worktree.sh create --force ${proposal_name%%-*}
EOF
    exit 1
  fi

  if worktree_registered "$worktree_abs"; then
    echo "Worktree already exists:"
    show_proposal "$proposal_dir"
  else
    mkdir -p "$(dirname "$worktree_abs")"
    if branch_exists "$branch"; then
      git -C "$REPO_DIR" worktree add "$worktree_abs" "$branch"
    else
      git -C "$REPO_DIR" worktree add -b "$branch" "$worktree_abs"
    fi
    echo "Created worktree:"
    show_proposal "$proposal_dir"
  fi

  cat <<EOF

Next steps:
1. Open a new terminal and run:
   cd $worktree_abs
2. Start a dedicated Codex session in that terminal.
3. Paste this prompt into that Codex session:

EOF
  print_prompt "$proposal_dir"
}

batch_create_worktrees() {
  local force_create="${1:-0}"
  shift
  [[ $# -ge 1 ]] || die "batch-create requires at least one <proposal-id>"
  ensure_spec_paths_committed
  local selector
  for selector in "$@"; do
    create_worktree "$(resolve_proposal_dir "$selector")" "$force_create"
    echo
  done
}

main() {
  require_git_repo

  local cmd="${1:-}"
  local force_create=0
  case "$cmd" in
    list)
      print_proposal_table
      ;;
    board)
      print_board
      ;;
    show)
      [[ $# -ge 2 ]] || die "show requires <proposal-id>"
      show_proposal "$(resolve_proposal_dir "$2")"
      ;;
    create)
      shift
      if [[ "${1:-}" == "--force" ]]; then
        force_create=1
        shift
      fi
      [[ $# -ge 1 ]] || die "create requires <proposal-id>"
      ensure_spec_paths_committed
      create_worktree "$(resolve_proposal_dir "$1")" "$force_create"
      ;;
    batch-create)
      shift
      if [[ "${1:-}" == "--force" ]]; then
        force_create=1
        shift
      fi
      batch_create_worktrees "$force_create" "$@"
      ;;
    prompt)
      [[ $# -ge 2 ]] || die "prompt requires <proposal-id>"
      print_prompt "$(resolve_proposal_dir "$2")"
      ;;
    ""|-h|--help|help)
      usage
      ;;
    *)
      die "unknown command: $cmd"
      ;;
  esac
}

main "$@"
