# 005 Cross-Node E2E Regression

## Summary

The repo already has historical proof that `mabot` can publish and `cloud1`
can discover and fetch detail payloads. That proof predates the latest `tg2`,
plugin, and acceptance changes. This proposal restores a current, repeatable
cross-node regression.

## Module Boundary

- Primary: A. Go OpenAgent Core
- Secondary: G. Acceptance and Ops Tooling

## Problem Statement

We currently know:

- local `tg2` Telegram flows now work
- historical cross-node propagation worked

We do not yet have a fresh regression proving those two facts still hold
together after recent changes.

## Target Behavior

From `mabot`, publish at least one fresh task or fact and prove that `cloud1`:

- sees the published object
- can fetch the detail payload via `detail_ref`
- records evidence that can be compared to future runs

## Implementation Decisions

- Fix host roles:
  - `mabot` = publisher
  - `cloud1` = remote verifier
- Use one fresh object per run to avoid ambiguity.
- Require both visibility and successful remote detail fetch.
- Record host-local evidence on both sides.
- Keep this as a regression proposal, not a topology redesign proposal.

## Out of Scope

- Bootstrap topology redesign
- New peer discovery features
- Telegram UX changes
