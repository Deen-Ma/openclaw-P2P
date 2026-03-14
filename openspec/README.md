# OpenSpec

This repo uses a lightweight, repo-local spec system for parallel development.

The goal is simple:

- define stable module boundaries first
- write one proposal per focused change
- map one proposal to one `git worktree`
- merge only after proposal acceptance is satisfied

## Layout

```text
openspec/
  README.md
  architecture/
  proposals/
  integration/
```

## Proposal Lifecycle

Use these states in `status.md`:

- `draft`: proposal exists but is not ready for implementation
- `active`: ready for implementation in its own worktree
- `blocked`: waiting on a dependency or external runtime issue
- `completed`: merged and accepted

## Worktree Model

Rules:

- one proposal per worktree
- one worktree per branch
- do not mix multiple proposal implementations in the same worktree
- update proposal status before and after implementation

Suggested commands:

```bash
git worktree add ../wt-000-adapter-baseline -b spec/000-adapter-workspace-baseline
git worktree add ../wt-003-tasks-api-stability -b spec/003-tasks-api-stability
```

Operational shortcuts:

```bash
./openspec-worktree.sh list
./openspec-worktree.sh create 000
./openspec-worktree.sh create --force 003
./openspec-worktree.sh prompt 000
```

By default, only proposals in `state: active` should be created as worktrees.
Use `--force` only for exceptional investigation work on a non-active proposal.

## Current Baseline

Start with:

- [architecture/system-overview.md](architecture/system-overview.md)
- [architecture/component-map.md](architecture/component-map.md)
- [architecture/interface-map.md](architecture/interface-map.md)
- [proposals/README.md](proposals/README.md)
- [integration/module-worktree-board.md](integration/module-worktree-board.md)
- [integration/worktree-codex-playbook.md](integration/worktree-codex-playbook.md)

For new proposals, start from:

- [proposals/_template/spec.md](proposals/_template/spec.md)
- [proposals/_template/tasks.md](proposals/_template/tasks.md)
- [proposals/_template/acceptance.md](proposals/_template/acceptance.md)
- [proposals/_template/status.md](proposals/_template/status.md)

The current model treats repo-tracked code and docs as formal modules.
Remote `mabot` runtime, Telegram real-world traffic, and peer machines remain
external dependencies unless a later proposal explicitly pulls them into repo
ownership.
