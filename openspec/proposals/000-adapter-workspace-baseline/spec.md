# 000 Adapter and Workspace Baseline

## Summary

The repo is missing the shared `adapter/` and `workspace-templates/` baseline
that later proposals were supposed to modify. Because that baseline is absent
from `main`, the current `001` and `002` worktree branches both re-add the same
foundational files and are not clean merge candidates.

## Module Boundary

- Primary: B. Adapter Orchestration
- Secondary: C. OpenClaw Tool Plugin
- Secondary: D. Local State Store
- Secondary: E. Telegram Bridge Runtime
- Supporting: F. Agent Workspace Templates

## Problem Statement

Current repo state:

- `main` does not yet track the shared `adapter/` source tree
- `main` does not yet track the shared `workspace-templates/` tree
- `001` and `002` therefore both carry the same baseline import plus their own
  proposal-specific behavior changes
- this breaks the one-proposal-per-diff model and makes later merge order
  ambiguous

## Target Behavior

After this proposal lands:

- `main` tracks the shared `adapter/` baseline
- `main` tracks the shared `workspace-templates/` baseline
- generated build artifacts and vendored dependencies are not part of the
  tracked baseline
- `001` and `002` can be rebuilt as small deltas on top of that baseline

## Implementation Decisions

- Import only the shared baseline, not the `001` detail-handle behavior change.
- Import only the shared baseline, not the `002` session-isolation behavior
  change.
- Treat `.gitignore` updates as part of the baseline only when they are needed
  to keep generated `adapter/dist/` output or `adapter/node_modules/` out of
  version control.
- Use the existing scratch worktree branches only as extraction sources; do not
  merge them directly.

## Out of Scope

- Fixing `read_detail` handle resolution
- Fixing session-key precedence or pending isolation
- Running the full Telegram regression flow
