# Proposal Queue

Each proposal is scoped so it can be implemented in its own `git worktree`.

| ID | Title | Modules | Status | Suggested worktree |
| --- | --- | --- | --- | --- |
| 001 | Detail Handle Resolution | C + D | active | `../wt-001-detail-handle` |
| 002 | Pending Session Isolation | C + D + F | active | `../wt-002-session-isolation` |
| 003 | Tasks API Stability | A | draft | `../wt-003-tasks-api-stability` |
| 004 | tg2 Behavior Regression | F + G | active | `../wt-004-tg2-regression` |
| 005 | Cross-Node E2E Regression | A + G | draft | `../wt-005-cross-node-e2e` |

Read `status.md` in each proposal directory before implementation.

Worktree creation should normally target proposals in `state: active`.
If a `draft` proposal must be explored early, use an explicit override:

```bash
./openspec-worktree.sh create --force 003
```

Use [`_template/`](./_template) when adding a new proposal.
