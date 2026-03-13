# Mabot 真实 Telegram 验收推进计划

- created_at: 2026-03-11 17:45:00 +0800
- mode: Plan
- request: 为 mabot 主 OpenClaw Telegram bot 制定真实 Telegram 验收推进计划并准备执行
- status: proposed
- source: codex proposed_plan on 2026-03-11

## Plan

## Summary

本轮以“验收优先”为唯一主线，不改代码，不做功能修复，先验证 `tg2` 私聊到 `openagent_network` 再到 OpenAgent 发布/回读的真实链路是否成立。执行模型固定为双人协作：你只负责在 Telegram 中按既定话术发消息，我负责通过 SSH 做预检、逐步观测、证据记录、失败分类和停机判断。默认策略为“失败即停”，任一步未达到通过条件时立即停止后续步骤，先保全现场并完成定位分类。

本轮结束产出应包含两类结果之一：
1. 完整通过结论：task draft/confirm、interest draft/confirm、cancel、my、detail 全部通过。
2. 首个阻塞点的定性结论：明确归因到 Telegram 运行态、主 agent 工具选择、adapter 状态落盘、或 OpenAgent API/节点层之一，并附对应证据。

## Key Changes / Execution Design

### 1. 固定执行顺序

按下列顺序执行，不允许跳步，不允许并行刺激多个会话：

1. 运行 `./observe-mabot-acceptance.sh mabot` 作为统一预检和基线抓取。
2. 如果预检通过，再进入同一个 `tg2` 私聊会话，依次执行：
   - task 草拟
   - task 确认发布
   - interest 草拟
   - interest 确认发布
   - software task 草拟后取消
   - `my` 回看
   - `detail` 回读
3. 每一步 Telegram 消息发出后，等待 bot 回复，再立即执行该步对应的 SSH 观测，不把两个步骤混在一起。
4. 任一步失败后立即停机，不继续发下一条 Telegram 消息。

### 2. 预检通过门槛

预检必须同时满足以下条件，否则本轮不进入真实验收：

- `openclaw status --json` 正常返回。
- `openclaw channels status --json` 中 Telegram 通道为 `running=true`。
- `tg2` 账号必须满足 `enabled=true`、`configured=true`、`running=true`，且没有当前有效 `lastError`。
- `curl http://127.0.0.1:7401/healthz`、`/v1/tasks`、`/v1/facts`、`/v1/sessions` 都能正常返回 JSON。
- `pending-network-actions.json` 在开始时为空，或至少不存在本轮会话的残留草案。
- `gateway.log` / `gateway.err.log` 在预检时没有持续刷新明显的 polling stall 恢复循环；如果仍在重复出现 `Polling stall detected` 且近时段无有效 inbound/outbound 时间戳，则直接判为 Telegram 运行态阻塞，不进入 Step 1。

若 `tg2` 不满足预检条件，再切换 `tg3`；除这一条外，不做其他临场改策。

### 3. 每一步的固定刺激与通过条件

- Step 1 `task` 草拟
  - Telegram 话术：`帮我找人一起做一个数学证明`
  - 通过条件：bot 返回确认文案；`pending-network-actions.json` 新增一条 `publish_task`；`taxonomy=task.help.math`；`outgoing-publications.json` 不新增对应发布。
- Step 2 `task` 确认发布
  - Telegram 话术：`确认`
  - 通过条件：bot 返回带 handle 的 receipt；pending 清空；`outgoing-publications.json` 新增 task；`adapter-events.jsonl` 出现 `publish_outgoing`；`/v1/tasks` 可见该记录。
- Step 3 `interest` 草拟
  - Telegram 话术：`我对设计协作很感兴趣`
  - 通过条件：bot 返回确认文案；pending 新增 `publish_interest`；`taxonomy=interest.design`；不直接发布。
- Step 4 `interest` 确认发布
  - Telegram 话术：`确认`
  - 通过条件：`outgoing-publications.json` 新增 fact；`/v1/facts` 可见该记录。
- Step 5 草拟后取消
  - Telegram 话术：先发 `帮我找人一起优化一个软件部署脚本`，收到确认文案后再发 `取消`
  - 通过条件：pending 清空；`outgoing-publications.json` 不新增该 software task。
- Step 6 `my` 回看
  - Telegram 话术：`看看我的协作发布`
  - 通过条件：bot 列出刚才成功发布的 task/fact handle；回复内容与 `outgoing-publications.json`、`/v1/tasks`、`/v1/facts` 基本一致。
- Step 7 `detail` 回读
  - Telegram 话术：`看看 <真实 task handle> 的详情`
  - 通过条件：bot 返回 detail 或摘要；`adapter-events.jsonl` 出现 `fetch_detail`。

### 4. 失败即停的分类规则

发生失败后，只做分类和证据收集，不做修复动作：

- Telegram 无回复或明显卡死
  - 归类为 Telegram 运行态问题。
  - 立即抓 `channels status --json`、`gateway.log`、`gateway.err.log` 和最近 inbound/outbound 状态。
- Telegram 有回复，但没有协作网络确认文案，或回复与协作无关
  - 归类为主 agent 工具选择问题。
  - 重点保存原始提问、原始回复、当时 agent 相关日志线索。
- bot 回复声称已草拟/发布，但 pending 或 outgoing 状态文件无变化
  - 归类为 adapter 层问题。
  - 重点保存 `pending-network-actions.json`、`outgoing-publications.json`、`adapter-events.jsonl`。
- `outgoing-publications.json` 已新增，但 `/v1/tasks` 或 `/v1/facts` 不可见
  - 归类为 OpenAgent API/节点层问题。
  - 重点保存 `/healthz`、`/v1/tasks`、`/v1/facts`、`/v1/sessions` 返回值。

## Public Interfaces / Evidence

不新增接口，不改现有入口，统一只使用以下接口和证据源：

- Telegram 私聊：`tg2`，预检失败时回退 `tg3`
- SSH 入口：`./connect-host.sh mabot`
- 统一观测脚本：`./observe-mabot-acceptance.sh mabot`
- 状态证据：
  - `~/openclaw-P2P/adapter/.local/adapter/pending-network-actions.json`
  - `~/openclaw-P2P/adapter/.local/adapter/outgoing-publications.json`
  - `~/openclaw-P2P/adapter/.local/adapter/adapter-events.jsonl`
  - `curl http://127.0.0.1:7401/v1/tasks`
  - `curl http://127.0.0.1:7401/v1/facts`
  - `curl http://127.0.0.1:7401/v1/sessions`
  - `~/.openclaw/logs/gateway.log`
  - `~/.openclaw/logs/gateway.err.log`

每一步都按统一记录模板落证据，至少记录：时间、Telegram 账号、原始话术、原始回复、预期行为、状态文件证据、API/日志证据、结论。

## Test Plan

成功判定必须满足：

- Telegram 可稳定收发，且至少完成一轮 task draft -> confirm -> OpenAgent 可见。
- interest draft -> confirm 成功，且在 `/v1/facts` 可见。
- cancel 不产生误发布。
- `my` 能列出刚刚发布的 handle。
- `detail` 能按真实 handle 触发读取，且事件日志可见 `fetch_detail`。

失败判定采用首个阻塞点原则：

- 任何一步未达到其通过条件，即结束本轮验收。
- 结束时必须有明确的失败分类和对应证据，不允许只记录“失败但原因未知”。

## Assumptions

- 本轮不改代码，不重启组件，不做临场修复；若出现阻塞，只做观察和归因。
- 默认 Telegram 验收账号为 `tg2`，仅在预检不通过时回退到 `tg3`。
- 所有 Telegram 消息都在同一个 DM 会话中完成，避免 pending session 漂移。
- 你负责发送 Telegram 话术；我负责 SSH 观测、证据记录和停机判断。
- 本轮计划完成后，如需进入修复阶段，再基于“首个阻塞点分类结果”单独出下一份修复计划。
