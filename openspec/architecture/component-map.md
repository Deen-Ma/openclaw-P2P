# Component Map

This file maps code areas to formal modules.

## Repo Modules

| Module | Primary paths | Owns | Depends on |
| --- | --- | --- | --- |
| A. Go OpenAgent Core | `cmd/`, `internal/` | Node startup, P2P, fetch protocol, task/fact/session APIs | Network peers, local persistence, bootstrap topology |
| B. Adapter Orchestration | `adapter/src/adapter.ts`, `adapter/src/client.ts`, `adapter/src/classifier.ts`, `adapter/src/topic-registry.ts` | Intent classification, publish/read orchestration, API client behavior | A, D |
| C. OpenClaw Tool Plugin | `adapter/src/openclaw-plugin.ts` | Tool schema, tool action routing, session resolution, user-facing replies | B, D |
| D. Local State Store | `adapter/src/store.ts` | Pending actions, outgoing publications, handle lookup, event log | Local filesystem |
| E. Telegram Bridge Runtime | `adapter/src/telegram-bridge.ts`, `adapter/src/cli.ts` | Direct Telegram polling bridge and command parsing | B, D, Telegram Bot API |
| F. Agent Workspace Templates | `workspace-templates/tg2/`, `sync-host-workspace.sh`, `reset-host-session.sh` | `tg2` behavior rules and deployment/sync for that workspace | External OpenClaw runtime |
| G. Acceptance and Ops Tooling | `observe-mabot-acceptance.sh`, `connect-host.sh`, `MABOT_OPENCLAW_TG_ACCEPTANCE.md`, `.local/mabot-acceptance-2026-03-11.md` | Acceptance procedure, SSH observation, evidence collection | External hosts and live runtime |

## Adapter Submodules

Module B is still one formal module, but proposal scoping should use these
internal boundaries so "adapter" does not become a catch-all bucket.

| Submodule | Primary paths | Owns |
| --- | --- | --- |
| B1. Adapter Core | `adapter/src/adapter.ts` | Publish/read orchestration, local record projection, high-level adapter flows |
| B2. Classification and Topic Routing | `adapter/src/classifier.ts`, `adapter/src/decision-provider.ts`, `adapter/src/topic-registry.ts`, `adapter/src/config.ts` | Text classification, topic selection, taxonomy mapping, runtime config defaults |
| B3. Transport Client | `adapter/src/client.ts` | HTTP calls into the local OpenAgent API |
| B4. Entry Bindings | `adapter/src/telegram-bridge.ts`, `adapter/src/cli.ts`, `adapter/src/openclaw-runtime-plugin.ts` | Process entrypoints and binding layers that invoke the adapter |

Boundary notes:

- `adapter/src/store.ts` remains module D, not module B.
- `adapter/src/openclaw-plugin.ts` remains module C even though it lives under
  `adapter/src/`; it is the OpenClaw-facing tool surface, not generic adapter
  orchestration.
- When a proposal touches both `openclaw-plugin.ts` and `adapter.ts`, record it
  as a cross-module change between C and B instead of calling the whole thing
  "adapter work".

## Ownership Rules

Use these rules when creating a new proposal:

- If the change is about how a tool call is interpreted, it belongs to module C.
- If the change is about pending/outgoing persistence or handle lookup, it belongs to module D.
- If the change is about HTTP/API correctness or fetch protocol behavior, it belongs to module A.
- If the change is about `tg2` choosing the right tool or shaping tool arguments, it belongs to module F unless plugin code also changes.
- If the change is about manual acceptance or repeatable evidence capture, it belongs to module G.
- If the change is confined to `classifier.ts`, `decision-provider.ts`, `topic-registry.ts`, or `client.ts`, cite the relevant B-submodule in the proposal text even though the formal module code is still B.

## Current Proposal Targets

| Proposal | Primary modules | Why |
| --- | --- | --- |
| `001-detail-handle-resolution` | C + D | Fix handle-based detail reads without agent fallback |
| `002-pending-session-isolation` | C + D + F | Stop stateful Telegram flows from sharing `sessionKey=default` |
| `003-tasks-api-stability` | A | Isolate and fix intermittent `/v1/tasks` failures |
| `004-tg2-behavior-regression` | F + G | Turn the successful Telegram flow into a repeatable regression package |
| `005-cross-node-e2e-regression` | A + G | Re-validate publish/gossip/detail fetch across hosts after recent changes |
