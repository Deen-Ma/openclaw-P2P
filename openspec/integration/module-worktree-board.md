# Module to Worktree Board

This board is the operational mapping from repo modules to proposal worktrees.

## Terminal and Worktree Layout

| Terminal | Role | Proposal | Modules | Suggested worktree | Start timing |
| --- | --- | --- | --- | --- | --- |
| `T1` | control | main repo | all | current repo | now |
| `T2` | execute | `000-adapter-workspace-baseline` | B, C, D, E, F | `../wt-000-adapter-baseline` | now |
| `T3` | execute | `003-tasks-api-stability` | A | `../wt-003-tasks-api-stability` | optional parallel line |
| `T4` | execute | `001-detail-handle-resolution` | C, D | `../wt-001-detail-handle-rework` | after `000` |
| `T5` | execute | `002-pending-session-isolation` | C, D, F | `../wt-002-session-isolation-rework` | after `000` |
| `T6` | execute | `004-tg2-behavior-regression` | F, G | `../wt-004-tg2-regression` | after `000`, `001`, and `002` |
| `T7` | execute | `005-cross-node-e2e-regression` | A, G | `../wt-005-cross-node-e2e` | after `003`, once promoted to `active` |

## Module Legend

| Code | Module |
| --- | --- |
| `A` | Go OpenAgent Core |
| `B` | Adapter Orchestration |
| `C` | OpenClaw Tool Plugin |
| `D` | Local State Store |
| `E` | Telegram Bridge Runtime |
| `F` | Agent Workspace Templates |
| `G` | Acceptance and Ops Tooling |

## Recommended First Wave

Open immediately:

- `000-adapter-workspace-baseline`

Optional third parallel line:

- `003-tasks-api-stability`

Hold until the first wave settles:

- `001-detail-handle-resolution`
- `002-pending-session-isolation`
- `004-tg2-behavior-regression`
- `005-cross-node-e2e-regression`

## Important Guardrail

Do not create fresh worktrees from an uncommitted `openspec/` state.

Reason:

- new worktrees are created from the current `HEAD`
- uncommitted proposal files and helper scripts do not appear in a new worktree
- that leads to Codex sessions implementing against stale specs

Commit or stash `openspec/` and `openspec-worktree.sh` before running:

```bash
./openspec-worktree.sh create 000
./openspec-worktree.sh batch-create 000 003
```

Only proposals in `state: active` should be created without an override.
Use `--force` only for exceptional investigation on a non-active proposal:

```bash
./openspec-worktree.sh create --force 003
```
