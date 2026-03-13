# Acceptance

## Automated

- `cd adapter && npm test`
- adapter tests cover handle-based detail resolution
- no test expects a failed fetch for handle input

## Manual

1. Publish or reuse a task with a stable handle such as `T2`.
2. In Telegram, send `看看 T2 的详情`.
3. Verify the bot returns the detail payload successfully.
4. Verify `adapter-events.jsonl` shows one `fetch_detail` with `ok=true`.
5. Verify there is no preceding `fetch_detail` failure for the same request.

## Commands

```bash
cd adapter && npm test
./connect-host.sh mabot 'tail -n 40 ~/openclaw-P2P/adapter/.local/adapter/adapter-events.jsonl'
./connect-host.sh mabot 'LATEST=$(ls -t ~/.openclaw/agents/tg2/sessions/*.jsonl 2>/dev/null | head -n 1); echo "$LATEST"; [ -n "$LATEST" ] && tail -n 80 "$LATEST"'
```

## Evidence

- original Telegram request and bot reply for `看看 T2 的详情`
- `~/.openclaw/agents/tg2/sessions/*.jsonl`
- `adapter/.local/adapter/adapter-events.jsonl`

## Stop Conditions

- stop if the bot does not reply within 30 seconds
- stop if the first `fetch_detail` for the handle-shaped request fails
- stop if the request only succeeds after a second fallback call
