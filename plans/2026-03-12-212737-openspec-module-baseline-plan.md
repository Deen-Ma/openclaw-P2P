# OpenSpec Module Baseline Plan

- created_at: 2026-03-12 21:29:01 +0800
- mode: Plan
- request: Define project modules first, then split future work into module-based proposals
- status: completed

## Plan

<proposed_plan>
# OpenClaw-P2P 模块总览与 Proposal 拆分基线

## Summary

先不要直接按 bug 或任务拆 proposal，先把项目实现分成稳定模块。这个仓库当前最合理的边界是：

- `repo` 内代码与文档是正式模块
- `mabot` 远端运行态、OpenClaw gateway、Telegram 真机交互属于外部依赖/部署环境
- proposal 只围绕“单个模块内的明确变更”或“两个相邻模块之间的接口变更”来建

按这个原则，当前项目可先分成 7 个正式模块和 3 个外部环境模块。后续 proposal 应从这些模块边界上切，而不是从零散现象上切。

## 模块划分

### A. Go OpenAgent Core
职责：
- 提供节点、gossip、detail fetch、`/v1/tasks`、`/v1/facts`、`/v1/sessions`
- 定义 Task/Fact envelope 与 fetch 协议

当前状态：
- 已存在
- 基础 publish / fetch / peer discover 已验证可行
- 仍有一个待查问题：`/v1/tasks` 偶发 `502`

适合出的 proposal 类型：
- API 稳定性
- detail fetch 协议问题
- 节点同步/查询行为问题

### B. Adapter Orchestration
核心位置：
- `adapter/src/adapter.ts`
- `adapter/src/client.ts`
- `adapter/src/classifier.ts`
- `adapter/src/topic-registry.ts`

职责：
- 把自然语言协作意图转换成 OpenAgent 发布/读取动作
- 管理 task / interest / withdraw / complete / read 的业务流程
- 对接 OpenAgent HTTP API

当前状态：
- 主流程已打通并验证
- 已修复默认状态目录问题
- 仍有读路径和稳定性类问题待收口

适合出的 proposal 类型：
- 发布/读取流程行为修正
- handle/detailRef 解析
- 主题分类和 taxonomy 映射

### C. OpenClaw Tool Plugin
核心位置：
- `adapter/src/openclaw-plugin.ts`

职责：
- 向 OpenClaw 暴露 `openagent_network`
- 处理 `draft_publish_task`
- `draft_publish_interest`
- `commit_pending`
- `cancel_pending`
- `read_feed`
- `read_my`
- `read_detail`

当前状态：
- 已被真实 Telegram 验收证明有效
- 仍有两个明确遗留：
  - `read_detail` 对 handle 解析不干净
  - `sessionKey` 回退到 `default`，多会话隔离不足

适合出的 proposal 类型：
- 工具行为修复
- session 作用域修复
- 输入归一化与容错改进

### D. Local State Store
核心位置：
- `adapter/src/store.ts`

职责：
- 管理 `pending-network-actions.json`
- 管理 `outgoing-publications.json`
- 管理 `adapter-events.jsonl`
- 维护 handle 与本地记录索引

当前状态：
- 已工作，验收链已依赖它通过
- 但 session 作用域目前仍受上层 `default` fallback 影响

适合出的 proposal 类型：
- pending 隔离
- 本地状态一致性
- 事件与 publication 持久化策略

### E. Telegram Bridge Runtime
核心位置：
- `adapter/src/telegram-bridge.ts`

职责：
- 直接桥接 Telegram Bot API
- 处理 `/task` `/my` `/feed` `/withdraw` `/complete` 等桥接命令
- 把 Telegram chat context 映射为 adapter 输入

当前状态：
- 代码存在，历史上单独桥接链路做过
- 当前主验收不是依赖这个 bridge，而是依赖主 OpenClaw bot + plugin
- 仍应视为独立模块，不应与主 bot workflow 混淆

适合出的 proposal 类型：
- 直连桥接能力补强
- Telegram context 透传一致性
- 命令与自然语言桥接回归

### F. Agent Workspace Templates
核心位置：
- `workspace-templates/tg2/`
- `sync-host-workspace.sh`
- `reset-host-session.sh`

职责：
- 定义 `tg2` 的角色约束、协作优先行为、自然语言到工具选择规则
- 提供远端同步和会话清理流程

当前状态：
- 已修复主 agent 不调用 `openagent_network` 的问题
- 已被真实验收证明有效

适合出的 proposal 类型：
- agent 行为策略优化
- 多 workspace 复制与版本化
- 行为规范文档化

### G. Acceptance / Ops Tooling
核心位置：
- `observe-mabot-acceptance.sh`
- `connect-host.sh`
- `MABOT_OPENCLAW_TG_ACCEPTANCE.md`
- `.local/mabot-acceptance-2026-03-11.md`

职责：
- SSH 观测
- 验收脚本
- 运行态排障
- 证据归档

当前状态：
- 已被本轮验收大量使用
- 已形成事实记录，但还不是统一 spec 体系

适合出的 proposal 类型：
- 验收标准化
- 证据采集模板化
- 运行态观测一致性

## 外部依赖边界

以下先不作为 repo 内正式模块，而是作为外部环境记录在架构图里：

### X1. OpenClaw Runtime on mabot
- gateway
- pairing
- Telegram accounts
- workspace 加载
- plugin 加载

### X2. Telegram Real World
- 真实用户
- pairing 流程
- 入站/出站稳定性
- polling 风险

### X3. Remote Peer Environment
- `cloud1`
- bootstrap / peer discover
- 跨节点 fetch/gossip

这些内容可以进入 proposal 的“依赖与验收环境”部分，但不应成为 proposal 主体，除非后面决定把部署层也纳入正式代码管理。

## Proposal 拆分原则

后续每个 proposal 必须遵守：

- 一个 proposal 只改一个正式模块，或一个模块与其直接相邻模块之间的接口
- 不允许把“代码修复 + 远端运维 + 手工验收”混成一个 proposal
- 每个 proposal 必须明确：
  - 影响模块
  - 不影响模块
  - 输入/输出或接口变化
  - 完成后的验收方法

按当前状态，第一批 proposal 应该这样切：

1. `detail-handle-resolution`
- 模块：C `OpenClaw Tool Plugin` + D `Local State Store`
- 目标：`看看 T2 的详情` 一次成功，不先失败再回退

2. `pending-session-isolation`
- 模块：C `OpenClaw Tool Plugin` + D `Local State Store`
- 目标：pending 不再使用全局 `default`，按 Telegram 会话隔离

3. `tasks-api-stability`
- 模块：A `Go OpenAgent Core`
- 目标：定位并修复 `/v1/tasks` 偶发 `502`

4. `tg2-behavior-regression`
- 模块：F `Agent Workspace Templates` + G `Acceptance / Ops Tooling`
- 目标：把当前真实 Telegram 验收收敛成可重复回归流程

5. `cross-node-e2e-regression`
- 模块：A `Go OpenAgent Core` + G `Acceptance / Ops Tooling`
- 目标：补回 `mabot -> cloud1` 的跨节点传播与 detail fetch 回归

## Test Plan

在进入 proposal 编写前，先把模块图作为统一基线。模块划分是否合格，以这几个标准判断：

- 每个现有代码目录都能归入某个正式模块
- 每个已发现问题都能落到单一模块或相邻模块接口
- 第一批 proposal 都能明确指向一个模块边界
- 不再出现“按现象切 proposal，最后跨 3 个模块一起改”的情况

## Assumptions

- 当前 `repo` 内正式模块优先，不把 `mabot` 远端运行态当作主代码模块管理
- `tg2` 主链验收已经足够证明当前架构拆分可用
- proposal 的目标是支持并行 `git worktree` 开发，所以模块边界必须优先服务并行性，而不是服务文档美观
- 下一步不是立刻实现，而是先基于这份模块图建立 `openspec` 目录，并把第一批 proposal 落进去
</proposed_plan>
