# 001 Detail Handle Resolution

## Summary

Handle-based detail reads must succeed in one pass.
Today, the real Telegram flow `看看 T2 的详情` first triggers a failed fetch
because the plugin tries to fetch `"T2"` as if it were a real `detail_ref`.

## Module Boundary

- Primary: C. OpenClaw Tool Plugin
- Secondary: D. Local State Store

## Problem Statement

Observed runtime behavior:

- agent calls `read_detail(detailRef="T2", sessionKey="default")`
- plugin does not recognize `"T2"` as a handle
- adapter emits a failed `fetch_detail`
- agent retries with a full `openagent://...` URI
- second call succeeds

This is acceptable as a recovery path, but it is not the intended interface.

## Target Behavior

Any of the following must work in one tool call:

- `handle="T2"`
- `detailRef="T2"`
- `detailRef="openagent://..."`

For handle-shaped input, the plugin must resolve the handle to a stored
record before calling fetch.

## Implementation Decisions

- Keep the public tool action name as `read_detail`.
- Keep both `handle` and `detailRef` input fields.
- Treat a non-empty `handle` as the highest-priority identifier.
- If `handle` is empty and `detailRef` matches `^[A-Z][0-9]+$`, treat it as a handle alias.
- Only pass a value into `fetchDetail()` if it is a resolved `detail_ref` URI.
- Handle resolution order:
  1. check local record store
  2. if not found, run one `syncIncoming()` and retry local lookup once
  3. if still not found, return a not-found reply without calling fetch
- A successful handle-based request must emit exactly one successful
  `fetch_detail` event and zero failed `fetch_detail` events.

## Out of Scope

- Changing Telegram prompt wording
- Changing OpenAgent fetch protocol format
- Changing the user-facing `detail` response format beyond current payload rendering
