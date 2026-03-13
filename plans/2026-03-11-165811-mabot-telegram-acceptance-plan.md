# Mabot 下一步验收计划

- created_at: 2026-03-11 16:58:11 +0800
- mode: Plan
- request: 回填上一轮 mabot 真实 Telegram 端到端验收计划
- status: backfilled
- source: user pasted from a prior session on 2026-03-11
- original_plan_date: unknown

## Plan

## Summary

下一步不再改代码，先做 mabot 的真实 Telegram 端到端验收。
控制面全部走 SSH，我负责观测和定位；刺激面走真实 Telegram，会由你在 tg2 私聊里按固定话术发消息。

目标是确认这条链是否真的成立：

Telegram 消息 -> 主 OpenClaw bot -> 调用 openagent_network -> 草拟/确认 -> OpenAgent 发布/回读

## Key Steps

- Step 0: SSH 预检
  - 我先通过 SSH 跑 ./observe-mabot-acceptance.sh
  - 确认 mabot 可连、openclaw status --json 正常、channels status --json 正常、OpenAgent API 正常
  - 记录验收前基线：pending-network-actions.json、outgoing-publications.json、adapter-events.jsonl、gateway.log
- Step 1: task 草拟
  - 你在 tg2 私聊里发送：
    - 帮我找人一起做一个数学证明
  - 预期：
    - bot 回复确认文案
    - 不直接发布
    - SSH 侧应看到 pending-network-actions.json 新增一条 publish_task
    - taxonomy 应为 task.help.math
- Step 2: task 确认发布
  - 你在同一会话里发送：
    - 确认
  - 预期：
    - bot 返回包含 handle 的 receipt
    - SSH 侧应看到 pending 清空
    - outgoing-publications.json 新增 task
    - /v1/tasks 能看到该记录
- Step 3: interest 草拟并确认
  - 然后发送 确认
  - 先生成 publish_interest pending
- Step 4: 草拟后取消
  - 你发送：
    - 帮我找人一起优化一个软件部署脚本
    - 收到确认文案后发送 取消
  - 预期：
    - pending 清空
    - outgoing-publications.json 不新增该 software task
- Step 5: 读取回看
  - 你发送：
    - 看看我的协作发布
  - 预期：
    - bot 列出刚才发布的 handle
    - 回复内容与 outgoing-publications.json、/v1/tasks、/v1/facts 一致
- Step 6: detail 回读
  - 你发送：
    - 看看 T1 的详情
    - 如果实际 handle 不是 T1，就替换成真实 handle
  - 预期：
    - bot 返回 detail 已取回或相应摘要
    - adapter-events.jsonl 出现 fetch_detail

## Interfaces / Evidence

- 不新增任何新接口，继续用现有入口：
  - Telegram 私聊 tg2
  - ./connect-host.sh mabot
  - ./observe-mabot-acceptance.sh
- 核心证据源固定为：
  - ~/openclaw-P2P/adapter/.local/adapter/pending-network-actions.json
  - ~/openclaw-P2P/adapter/.local/adapter/outgoing-publications.json
  - ~/openclaw-P2P/adapter/.local/adapter/adapter-events.jsonl
  - curl http://127.0.0.1:7401/v1/tasks
  - curl http://127.0.0.1:7401/v1/facts
  - ~/.openclaw/logs/gateway.log

## Test Scenarios

- 通过
  - Telegram 有回复
  - 草拟时只生成 pending，不直接发布
  - 确认后 pending 清空，publication 落盘，OpenAgent 节点状态同步可见
  - 取消时 pending 清空但 publication 不新增
  - my 和 detail 可正常回读
- 失败分类
  - Telegram 无回复：优先看 channels status 和 gateway.log
  - 有回复但没有 pending/publication：优先看 agent 是否实际调用了 openagent_network
  - publication 已落盘但 /v1/tasks 或 /v1/facts 没有：归因到 OpenAgent 节点/API 层

## Assumptions

- 这轮默认使用 tg2 作为真实 Telegram 验收账号。
- 你负责在 Telegram 里发测试消息，我负责全程通过 SSH 做观测和定位。
- 本轮只做验收，不再改功能逻辑，除非验收中发现明确阻塞点。
- 如果 tg2 无法稳定收发，再回退到 tg3。
