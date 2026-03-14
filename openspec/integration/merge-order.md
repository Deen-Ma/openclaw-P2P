# Merge Order

This file defines the recommended merge order for the first proposal batch.

## Order

1. `000-adapter-workspace-baseline`
2. `001-detail-handle-resolution`
3. `002-pending-session-isolation`
4. `003-tasks-api-stability`
5. `004-tg2-behavior-regression`
6. `005-cross-node-e2e-regression`

## Why This Order

- `000` establishes the shared adapter/workspace baseline so follow-on proposal
  diffs stay small and non-overlapping.
- `001` removes a known user-visible defect in the `detail` path.
- `002` removes the main correctness risk for concurrent Telegram usage.
- `003` stabilizes the API surface that later regressions rely on.
- `004` should capture the cleaned-up `tg2` behavior, not the pre-baseline or
  pre-fix state.
- `005` should run after the local flow and task-read surface are considered stable.

## Parallelism Rules

Safe to implement in parallel:

- `000` and `003`
- `001` and `003` after `000`
- `002` and `003` after `000`

Prefer not to merge before dependencies settle:

- `001` after `000`
- `002` after `000`
- `004` after `000`, `001`, and `002`
- `005` after `003`
