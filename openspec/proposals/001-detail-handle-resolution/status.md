# Status

- state: blocked
- owner: unassigned
- modules: C, D
- suggested_branch: `spec/001-detail-handle-rework`
- suggested_worktree: `../wt-001-detail-handle-rework`
- dependencies: proposal 000-adapter-workspace-baseline must land first
- blocked_reason: current scratch branch `spec/001-detail-handle` also imports the shared adapter baseline and must not be merged directly
- completion_gate: one-shot handle detail read works in tests and real Telegram acceptance after rework on top of the merged baseline
