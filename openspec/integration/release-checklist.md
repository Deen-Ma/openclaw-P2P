# Release Checklist

Before merging a proposal branch:

1. `status.md` reflects the real state of the proposal.
2. The implementation matches the proposal scope and does not quietly absorb
   unrelated work.
3. Tests or reproducible checks for that proposal have been run.
4. Acceptance evidence is linked or recorded in repo-local notes.
5. Any required remote/runtime steps are documented explicitly.
6. Any follow-on work discovered during implementation is split into a new
   proposal instead of being smuggled into the current one.

Before merging multiple proposal branches together:

1. Re-check `openspec/integration/merge-order.md`.
2. Re-run the highest-level regression that covers the merged surface.
3. Update architecture docs if module ownership or boundaries changed.
