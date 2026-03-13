# Acceptance

## Automated

- `cd adapter && npm test`
- plugin tests cover session resolution precedence
- store tests cover independent pending actions for different session keys

## Manual

1. Start one draft from Telegram user A.
2. Start a different draft from Telegram user B.
3. Confirm user A only sees and commits user A's draft.
4. Cancel user B only clears user B's draft.
5. Inspect `~/.openclaw/agents/tg2/sessions/*.jsonl` and confirm stateful tool
   calls use `telegram:tg2:dm:<sender_id>` instead of `default`.

## Commands

```bash
cd adapter && npm test
./observe-mabot-acceptance.sh mabot
./connect-host.sh mabot 'cat ~/openclaw-P2P/adapter/.local/adapter/pending-network-actions.json'
./connect-host.sh mabot 'LATEST=$(ls -t ~/.openclaw/agents/tg2/sessions/*.jsonl 2>/dev/null | head -n 1); echo "$LATEST"; [ -n "$LATEST" ] && tail -n 120 "$LATEST"'
```

## Evidence

- `adapter/.local/adapter/pending-network-actions.json`
- `~/.openclaw/agents/tg2/sessions/*.jsonl`
- raw Telegram messages from both user A and user B

## Stop Conditions

- stop if any stateful tool call still uses `sessionKey="default"` while sender metadata exists
- stop if one user can confirm or cancel the other user's draft
- stop after the first cross-user collision and preserve the pending file plus session logs
