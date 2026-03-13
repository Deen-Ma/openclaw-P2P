# 004 tg2 Behavior Regression

## Summary

The current real Telegram acceptance succeeded, but the knowledge is split
across ad hoc notes, shell commands, and session artifacts. This proposal turns
the successful `tg2` flow into a repeatable regression package.

## Module Boundary

- Primary: F. Agent Workspace Templates
- Secondary: G. Acceptance and Ops Tooling

## Problem Statement

Today, re-running the successful `tg2` flow depends on operator memory:

- which workspace files to sync
- which session files to reset
- which Telegram messages to send
- which evidence sources to inspect
- how to classify failures

## Target Behavior

One operator should be able to re-run the full `tg2` regression with a single
runbook and minimal tribal knowledge.

## Implementation Decisions

- Keep the current acceptance scenario order:
  - task draft
  - task confirm
  - interest draft
  - interest confirm
  - cancel draft
  - `my`
  - `detail`
- Package the reset, sync, and evidence steps into a single documented workflow.
- Preserve the distinction between:
  - runtime failures
  - tool-routing failures
  - adapter/state failures
  - node/API failures
- Keep `tg2` as the primary account; `tg3` remains a fallback, not a default.

## Out of Scope

- Fixing product bugs discovered by the regression itself
- Redesigning Telegram runtime deployment
