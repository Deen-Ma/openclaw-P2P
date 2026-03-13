# Tasks

1. Update the `tg2` workspace instructions to emit the canonical Telegram DM
   `sessionKey`.
2. Harden plugin-side `resolveSessionKey()` so `"default"` is not treated as a
   high-quality explicit key.
3. Add tests proving two different session keys keep independent pending drafts.
4. Re-run Telegram draft/confirm/cancel scenarios and capture session keys from
   the `tg2` session log.
5. Confirm no new stateful `tg2` tool calls use `sessionKey="default"` during
   the acceptance run.
