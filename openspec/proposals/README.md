# Proposal Queue

Each proposal is scoped so it can be implemented in its own `git worktree`.

| ID | Title | Modules | Status | Suggested worktree |
| --- | --- | --- | --- | --- |
| 000 | Adapter and Workspace Baseline | B + C + D + E + F | active | `../wt-000-adapter-baseline` |
| 001 | Detail Handle Resolution | C + D | blocked | `../wt-001-detail-handle-rework` |
| 002 | Pending Session Isolation | C + D + F | blocked | `../wt-002-session-isolation-rework` |
| 003 | Tasks API Stability | A | draft | `../wt-003-tasks-api-stability` |
| 004 | tg2 Behavior Regression | F + G | blocked | `../wt-004-tg2-regression` |
| 005 | Cross-Node E2E Regression | A + G | draft | `../wt-005-cross-node-e2e` |

Read `status.md` in each proposal directory before implementation.

Worktree creation should normally target proposals in `state: active`.
If a `draft` proposal must be explored early, use an explicit override:

```bash
./openspec-worktree.sh create --force 003
```

Blocked proposals are not implementation-ready. Unblock them only after their
listed dependencies have landed.

Use [`_template/`](./_template) when adding a new proposal.
