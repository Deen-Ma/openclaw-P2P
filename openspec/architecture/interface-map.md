# Interface Map

This file describes the important interfaces between modules and records which
flows are already verified.

## Primary Interfaces

| From | To | Interface | Current status |
| --- | --- | --- | --- |
| Telegram user | OpenClaw runtime | Telegram DM / OpenClaw conversation | verified in real use |
| `tg2` workspace | `openagent_network` plugin | Tool call selection and argument shaping | verified, but session scoping needs work |
| Plugin | Store | Pending/outgoing/event persistence | verified |
| Plugin | Adapter | Publish/read action API | verified |
| Adapter | OpenAgent HTTP API | `healthz`, `/v1/tasks`, `/v1/facts`, `/v1/sessions`, fetch detail | mostly verified; `/v1/tasks` needs stability work |
| OpenAgent node | Remote peers | Gossip and fetch by `detail_ref` | historically verified; current regression is pending |

## Verified User Flows

| Flow | Path | Status |
| --- | --- | --- |
| Draft task | Telegram -> workspace -> plugin -> store | verified |
| Confirm task publish | Telegram -> workspace -> plugin -> adapter -> API -> node | verified |
| Draft interest | Telegram -> workspace -> plugin -> store | verified |
| Confirm interest publish | Telegram -> workspace -> plugin -> adapter -> API -> node | verified |
| Cancel draft | Telegram -> workspace -> plugin -> store | verified |
| Read my publications | Telegram -> workspace -> plugin -> adapter/store | verified |
| Read detail | Telegram -> workspace -> plugin -> adapter -> fetch | partially verified |

## Detail Flow Gap

Current observed behavior for `看看 T2 的详情`:

1. agent first sends `read_detail(detailRef="T2")`
2. plugin treats `"T2"` like a real `detail_ref`
3. fetch fails with `unsupported detail_ref scheme ""`
4. agent retries using the full `openagent://...` URI
5. second fetch succeeds

The user-visible flow works, but the interface contract is wrong. Handle input
must be resolved before any fetch call is attempted.

## Session Flow Gap

Current observed behavior for stateful tool calls:

- many `tg2` calls still use `sessionKey="default"`
- the store keys pending state by `sessionKey`
- multi-user or multi-thread use could collide

The interface contract that needs to be enforced is:

- every Telegram DM flow must provide a stable, user-scoped `sessionKey`
- `default` is only an emergency fallback, not the normal path

## Acceptance Evidence Sources

These interfaces are currently validated using:

- `~/.openclaw/agents/tg2/sessions/*.jsonl`
- `adapter/.local/adapter/pending-network-actions.json`
- `adapter/.local/adapter/outgoing-publications.json`
- `adapter/.local/adapter/adapter-events.jsonl`
- `http://127.0.0.1:7401/v1/tasks`
- `http://127.0.0.1:7401/v1/facts`
