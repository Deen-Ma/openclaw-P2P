# Worktree and Codex Playbook

This runbook turns the `openspec/` proposal queue into an operational workflow.

## Goal

Run multiple focused proposals in parallel without mixing scope, context, or
branches.

The operating model is:

- one proposal per branch
- one branch per worktree
- one worktree per terminal
- one terminal per Codex session

## Recommended Terminal Layout

Use this layout for the current repo:

| Terminal | Directory | Purpose |
| --- | --- | --- |
| 1 | main repo | planning, `openspec/`, review, integration, merge |
| 2 | `../wt-000-adapter-baseline` | implement proposal `000` |
| 3 | optional third worktree | proposal `003` only if you want a parallel investigation line |
| 4 | `../wt-001-detail-handle-rework` | implement proposal `001` after `000` |
| 5 | `../wt-002-session-isolation-rework` | implement proposal `002` after `000` |

Do not use one Codex session to hop across multiple worktrees.

## Daily Flow

### 1. In the main repo, inspect the queue

```bash
./openspec-worktree.sh list
```

Read:

- `openspec/proposals/README.md`
- `openspec/integration/merge-order.md`

### 2. Create one worktree per active proposal

Examples:

```bash
./openspec-worktree.sh create 000
```

Only create `state: active` proposals by default. If you need to investigate a
`draft` proposal before promoting it, use an explicit override:

```bash
./openspec-worktree.sh create --force 003
```

This uses each proposal's `status.md` to pick:

- the branch name
- the target worktree path

### 3. Open a dedicated terminal per worktree

Example:

```bash
cd ../wt-000-adapter-baseline
```

Start Codex in that terminal, then paste:

```bash
./openspec-worktree.sh prompt 000
```

If you are already inside the worktree, the same script path still works
because the worktree contains the whole repo checkout.

### 4. Keep each Codex session scoped

Each implementation session must:

- read only its own proposal files first
- avoid unrelated fixes
- run only its own proposal tests and acceptance checks
- treat `acceptance.md` as an executable checklist, not a vague success note
- leave follow-on work as a new proposal instead of expanding scope

### 5. Merge back through the main repo

When a proposal branch is ready:

1. return to the main repo terminal
2. review the proposal branch
3. verify acceptance evidence
4. merge in the order from `openspec/integration/merge-order.md`

Do not merge in "who finished first" order if dependency order disagrees.

## Current Recommended Start

The current first-wave proposal is:

- `000-adapter-workspace-baseline`

Use `003-tasks-api-stability` as a third line only if you want one engineer or
one Codex session dedicated to investigation work.

Hold these until the earlier fixes settle:

- `001-detail-handle-resolution`
- `002-pending-session-isolation`
- `004-tg2-behavior-regression`
- `005-cross-node-e2e-regression`

## Completion Checklist

Before declaring a proposal done:

- code changes stay inside proposal scope
- relevant tests pass
- manual checks from `acceptance.md` are complete
- `status.md` is updated
- evidence is recorded in the repo's usual notes/logs
