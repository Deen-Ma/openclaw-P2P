# 003 Tasks API Stability

## Summary

`GET /v1/tasks` intermittently returned `502 Bad Gateway` during the current
acceptance run even while `healthz` and `/v1/facts` remained available.
This proposal exists to make the failure reproducible, diagnosable, and then
fixable inside the Go core if the root cause is repo-owned.

## Module Boundary

- Primary: A. Go OpenAgent Core

## Problem Statement

Observed behavior during acceptance:

- `/v1/tasks` sometimes returned `502`
- immediate or later retry sometimes returned `200`
- other API endpoints stayed healthy

This creates uncertainty for `my`, `detail`, and any acceptance that depends on
task visibility.

## Target Behavior

- `GET /v1/tasks` is stable under repeated reads against a healthy local node
- if an internal failure does occur, the server logs a concrete root cause
- the proposal ends with either:
  - a merged code fix and regression coverage, or
  - a documented external blocker with captured evidence

## Implementation Decisions

- Start with reproducibility, not guesswork.
- Add a focused repeat-read harness for `/v1/tasks`.
- Audit the HTTP handler, task list building path, and serialization path.
- Add server-side logging at failure boundaries before returning `502`.
- Do not solve this with client-side retries in this proposal.
- Treat intermittent `502` as a server correctness issue, not an acceptance-only issue.

## Out of Scope

- General API redesign
- Unrelated `/v1/facts` or `/v1/sessions` refactors
- Telegram behavior changes
