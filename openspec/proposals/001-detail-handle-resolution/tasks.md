# Tasks

1. Update plugin-side detail input normalization so handle-shaped `detailRef`
   values are routed through handle resolution.
2. Reuse existing record lookup and one-time sync retry before any fetch call.
3. Add unit tests for:
   - `handle="T2"`
   - `detailRef="T2"`
   - `detailRef="openagent://..."`
   - unknown handle does not trigger fetch
4. Re-run the real Telegram `T2` detail read acceptance.
5. Mark the proposal completed only after the failed-then-retry pattern
   disappears from `adapter-events.jsonl`.
