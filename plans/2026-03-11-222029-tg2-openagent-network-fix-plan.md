# Repair `tg2` OpenAgent Network Routing

- created_at: 2026-03-11 22:20:29 +08:00
- mode: Plan
- request: Repair `tg2` so collaboration intents draft through `openagent_network` instead of falling back to generic Q&A
- status: proposed

## Plan

当前首个阻塞点已经明确：`tg2` 的消息确实到达了主 OpenClaw bot，也完成了普通回复，但主 agent 没有把“帮我找人一起做一个数学证明”收敛成协作网络草拟动作，而是按通用助理路径继续追问细节。`openagent_network` 工具本身已注册，`adapter` 和 OpenAgent API 没有成为当前瓶颈。

本轮按“强协作优先”修复，并且采用“仓库固化 + 远端应用”的方式落地。目标不是把 `tg2` 变成纯命令 bot，而是让它在识别到协作网络意图时，优先生成待确认草案；只有当连最小草拟条件都不满足时，才补问必要字段。

### 1. 固化 `tg2` 的协作网络角色约束

在仓库中新增一套可同步到 `workspace-tg2` 的角色模板，覆盖当前通用 assistant 模板的缺口，明确这些规则：

- 当用户表达“找人一起做”“想找协作者”“我对某方向感兴趣”“帮我发布协作需求”等意图时，优先考虑 `openagent_network`。
- 默认采用“先草拟、后确认”的流程，不直接发布。
- 只要能从原句中提取出最小草拟信息，就先调用：
  - `draft_publish_task`，用于协作任务
  - `draft_publish_interest`，用于兴趣声明
- 对于像“帮我找人一起做一个数学证明”这样的输入，允许先用粗粒度 topic/taxonomy 草拟：
  - `topicKey = math`
  - `taxonomy = task.help.math`
  - `summary` 直接从用户原句压缩得出
- 不允许在草拟前先进入通用咨询式长表单追问，除非连任务/兴趣类型都无法判断。
- 一旦草拟完成，回复必须是待确认文案，而不是继续讨论问题细节。
- 用户回复“确认”时提交 pending；回复“取消”时清空 pending。
- `my` / `feed` / `detail` 类型请求优先映射为工具读操作，而不是通用对话。

这部分应落在 repo 内的 workspace 模板或远端同步素材中，而不是只依赖当前远端手工状态。

### 2. 保持现有工具接口不变，只修主 agent 选择策略

不修改 `openagent_network` 的公开接口，不改 `adapter` 的 action 名或 pending 数据结构。当前工具接口已经满足需要：

- `draft_publish_task`
- `draft_publish_interest`
- `commit_pending`
- `cancel_pending`
- `read_feed`
- `read_my`
- `read_detail`

实现重点是让 `tg2` 的角色提示和行为规则更强地偏向这些动作，而不是再增加新的工具或 API。

### 3. 补一条可重复部署的远端应用流程

在仓库中补一份针对 `mabot` / `tg2` 的应用说明或同步流程，保证后续可以稳定把角色模板推到远端 workspace，而不是靠人工临时改：

- 约定 `workspace-tg2` 需要落哪些文件。
- 明确哪些文件属于协作网络角色模板，哪些仍保留通用人格信息。
- 约定更新后必须重跑的验证：
  - 查看远端 workspace 文件内容
  - 重新触发 `tg2` 会话
  - 确认会话日志里出现 `openagent_network` 工具调用

如果现有脚本不足以同步 workspace 文件，本轮计划里补一个简单、非交互、可重复使用的同步入口。

### 4. 把运行态问题与 agent 选择问题分开处理

Telegram polling stall 仍然存在间歇性风险，但这轮真实样本已经证明：

- pairing 已可完成
- 至少一次真实 Telegram 入站已到达 `tg2`
- 至少一次真实 Telegram 出站已成功回复

因此本轮修复主线聚焦 agent 选择策略，不再把 adapter / OpenAgent API 当成优先排查对象。运行态问题继续记录，但只在“消息完全收不到”时重新上升为主阻塞。

## Test Plan

修复后按下面顺序复验，仍采用失败即停：

1. 预检
- `channels status --json` 可读，`tg2` 仍为 `running=true`
- pairing 已批准，不再出现 access not configured
- 不要求 `lastInboundAt` / `lastOutboundAt` 先变非空才开始，但要记录当前值

2. Step 1 重新验证 task 草拟
- Telegram 发送：`帮我找人一起做一个数学证明`
- 通过条件：
  - bot 返回待确认草拟文案，不再返回长表单式澄清
  - 远端生成 `pending-network-actions.json`
  - pending 的 `kind = publish_task`
  - `topicKey = math`
  - `taxonomy = task.help.math`

3. Step 2 确认发布
- Telegram 发送：`确认`
- 通过条件：
  - pending 清空
  - `outgoing-publications.json` 新增 task
  - `/v1/tasks` 出现对应记录

4. 行为回归
- Telegram 发送一个明显不是协作网络发布的普通问题
- 通过条件：
  - `tg2` 仍可以正常走通用助理回复
  - 不会误创建 pending draft

5. 证据检查
- `~/.openclaw/agents/tg2/sessions/*.jsonl` 中能看到 `openagent_network` 工具调用
- `adapter-events.jsonl` 与 pending/outgoing 文件变化一致

## Assumptions

- 当前 `openagent_network` 工具已经正确注册，不需要新增工具或修改 action schema。
- 当前核心问题是 `workspace-tg2` 的角色提示未对协作网络意图建立足够强的优先级。
- 本轮默认优先修 `tg2`；`tg3` 不同步修改，除非 `tg2` 方案验证通过后再复制。
- 运行态 Telegram stall 继续存在风险，但不阻止本轮针对 agent 选择策略的修复。
- 若角色模板修复后仍不调用工具，再进入下一轮计划，排查 OpenClaw 主 agent 的工具选择权重或系统提示拼装逻辑。
