# Status

- state: blocked
- owner: unassigned
- modules: C, D, F
- suggested_branch: `spec/002-pending-session-isolation-rework`
- suggested_worktree: `../wt-002-session-isolation-rework`
- dependencies: proposal 000-adapter-workspace-baseline must land first
- blocked_reason: current scratch branch `spec/002-pending-session-isolation` also imports the shared adapter/workspace baseline and must not be merged directly
- completion_gate: adapter-side stateful session resolution and isolation are covered by tests and validated against the live acceptance expectations after rework on top of the merged baseline
