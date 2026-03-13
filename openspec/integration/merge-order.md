# Merge Order

This file defines the recommended merge order for the first proposal batch.

## Order

1. `001-detail-handle-resolution`
2. `002-pending-session-isolation`
3. `003-tasks-api-stability`
4. `004-tg2-behavior-regression`
5. `005-cross-node-e2e-regression`

## Why This Order

- `001` removes a known user-visible defect in the `detail` path.
- `002` removes the main correctness risk for concurrent Telegram usage.
- `003` stabilizes the API surface that later regressions rely on.
- `004` should capture the cleaned-up `tg2` behavior, not the buggy baseline.
- `005` should run after the local flow and task-read surface are considered stable.

## Parallelism Rules

Safe to implement in parallel:

- `001` and `003`
- `002` and `003`

Prefer not to merge before dependencies settle:

- `004` after `001` and `002`
- `005` after `003`
