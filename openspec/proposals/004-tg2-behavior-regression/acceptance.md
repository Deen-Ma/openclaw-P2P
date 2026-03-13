# Acceptance

## Automated

- `bash -n observe-mabot-acceptance.sh`
- `bash -n sync-host-workspace.sh`
- `bash -n reset-host-session.sh`

## Manual Regression Matrix

Required steps:

1. preflight
2. task draft
3. task confirm
4. interest draft
5. interest confirm
6. cancel draft
7. `my`
8. `detail`

Each step must record:

- Telegram input
- bot reply
- session log evidence
- pending/outgoing/events evidence
- API evidence where relevant
- pass/fail classification

## Commands

```bash
bash -n observe-mabot-acceptance.sh
bash -n sync-host-workspace.sh
bash -n reset-host-session.sh
./sync-host-workspace.sh mabot tg2
./connect-host.sh mabot 'cat ~/.openclaw/agents/tg2/sessions/sessions.json'
./reset-host-session.sh mabot tg2 '<session-key>'
./observe-mabot-acceptance.sh mabot
```

Use the message sequence from `MABOT_OPENCLAW_TG_ACCEPTANCE.md`:

1. `帮我找人一起做一个数学证明`
2. `确认`
3. `我对设计协作很感兴趣`
4. `确认`
5. `帮我找人一起优化一个软件部署脚本`
6. `取消`
7. `看看我的协作发布`
8. `看看 T2 的详情`

## Evidence

- `MABOT_OPENCLAW_TG_ACCEPTANCE.md`
- `.local/mabot-acceptance-2026-03-11.md`
- `~/openclaw-P2P/adapter/.local/adapter/pending-network-actions.json`
- `~/openclaw-P2P/adapter/.local/adapter/outgoing-publications.json`
- `~/openclaw-P2P/adapter/.local/adapter/adapter-events.jsonl`
- `~/.openclaw/agents/tg2/sessions/*.jsonl`
- `~/.openclaw/logs/gateway.log`
- `~/.openclaw/logs/gateway.err.log`

## Stop Conditions

- stop after the first failed step in the regression matrix
- stop if Telegram has no reply and classify the failure before retrying
- stop if bot replies but pending/outgoing/API evidence disagrees with the claimed behavior

## Success Condition

A second operator can run the regression from the written material alone and
produce a comparable evidence log.
