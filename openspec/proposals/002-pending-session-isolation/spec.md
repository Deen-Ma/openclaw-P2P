# 002 Pending Session Isolation

## Summary

Pending collaboration drafts must be isolated per Telegram conversation.
The current acceptance run proved that the main flow works, but many tool calls
still use `sessionKey="default"`, which risks cross-user draft collisions.

## Module Boundary

- Primary: C. OpenClaw Tool Plugin
- Secondary: D. Local State Store
- Supporting: F. Agent Workspace Templates

## Problem Statement

Current behavior:

- stateful tool calls from `tg2` often send `sessionKey="default"`
- pending drafts are keyed by `sessionKey`
- different Telegram users or conversations could share the same pending slot

## Target Behavior

Every Telegram DM flow must use a stable, user-scoped session key.

For `tg2` Telegram DMs, the canonical format is:

```text
telegram:tg2:dm:<sender_id>
```

## Implementation Decisions

- Update `workspace-templates/tg2/AGENTS.md` so all stateful
  `openagent_network` calls explicitly pass the canonical Telegram session key
  when sender metadata is present in the conversation.
- Use the Telegram sender id from the incoming metadata block as `<sender_id>`.
- Treat these actions as stateful and therefore session-key sensitive:
  - `draft_publish_task`
  - `draft_publish_interest`
  - `commit_pending`
  - `cancel_pending`
  - `read_my`
  - `read_feed`
  - `read_detail`
- Update plugin-side session resolution so literal `"default"` is only accepted
  as a last-resort fallback.
- If a better identifier is present in `context`, prefer it over explicit
  `"default"`.
- Keep store file format unchanged; the change is only in key quality and usage.

## Out of Scope

- Redesigning pending storage format
- Supporting non-Telegram session-key conventions in this proposal
- Solving group chat semantics beyond the canonical DM key above
