# OpenSpec Hardening Plan

- created_at: 2026-03-14 05:57:30 +0800
- mode: Plan
- request: Optimize openspec before parallel worktree execution
- status: completed

## Plan

<proposed_plan>
# OpenSpec 优化计划

## Summary

本轮不再扩展新的 proposal，而是先把 `openspec` 本身收口，解决刚才发现的 3 个结构性问题：

- `draft` proposal 目前也能直接创建 worktree，生命周期约束不严
- `acceptance.md` 还不够可执行，多个 Codex 并行时容易各自发明验证方法
- `Adapter` 的细分结构只存在于图里，还没有正式进入规范文本

同时补一套 proposal 模板，避免后续新 proposal 质量漂移。目标是把这套 `openspec` 提升到“可以稳定长期并行开发”的状态，而不只是能勉强启动第一轮。

## Key Changes

### 1. 收紧 worktree 生命周期约束
修改 [`openspec-worktree.sh`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec-worktree.sh) 的行为规则：

- `create` 和 `batch-create` 默认只允许 `state=active`
- `state=draft` 时：
  - `list`
  - `board`
  - `show`
  - `prompt`
  仍然允许
  - `create`
  默认拒绝
- 如果确实要对 `draft` proposal 强制开工，新增显式开关：
  - `./openspec-worktree.sh create --force 003`
  - `./openspec-worktree.sh batch-create --force 003`
- 错误提示里要明确说明：
  - 当前 proposal 还是 `draft`
  - 需要先把 spec 定稿并改成 `active`
  - 或显式使用 `--force`

这样可以保证 `status.md` 真正参与流程控制，而不是只做装饰。

### 2. 把 acceptance 从“描述”升级成“执行说明”
对当前 001-005 的每个 `acceptance.md` 统一补成同一结构：

- `Automated`
  - 本地测试命令
  - 若无自动化，也要明确写 `none yet`
- `Manual`
  - 明确按步骤执行
- `Commands`
  - 具体命令行
  - 尽量直接可复制运行
- `Evidence`
  - 证据文件路径
  - 需要看哪些日志/状态文件
- `Stop Conditions`
  - 哪种失败出现后应立即停止并回报

每个 proposal 的命令内容要按它自己的模块边界写，不要混：

- `001`
  - 重点是 adapter/plugin 单测 + `detail` Telegram 回归
- `002`
  - 重点是 sessionKey 行为验证 + 多用户/多会话草稿隔离
- `003`
  - 重点是 `/v1/tasks` 重复读取脚本和服务端日志
- `004`
  - 重点是 runbook 化的 tg2 回归步骤
- `005`
  - 重点是 `mabot` / `cloud1` 的跨节点证据链

目标是让“acceptance 文件”本身就能当执行清单，而不是还需要另一个人脑内补全。

### 3. 把 Adapter 细分正式写入架构文档
在 [`openspec/architecture/component-map.md`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec/architecture/component-map.md) 增加一个 `Adapter Submodules` 段落，但不推翻现有 7 大模块编号。

做法是：

- 保留 `B. Adapter Orchestration` 作为一级模块
- 在文档中明确其 4 个子层：
  - `B1` Adapter Core
    - `adapter/src/adapter.ts`
  - `B2` Classification and Topic Routing
    - `adapter/src/classifier.ts`
    - `adapter/src/decision-provider.ts`
    - `adapter/src/topic-registry.ts`
    - `adapter/src/config.ts`
  - `B3` Transport Client
    - `adapter/src/client.ts`
  - `B4` Entry Bindings
    - `adapter/src/openclaw-plugin.ts`
    - `adapter/src/openclaw-runtime-plugin.ts`
    - `adapter/src/telegram-bridge.ts`
    - `adapter/src/cli.ts`
- 同时明确：
  - `store.ts` 仍属于 `D. Local State Store`
  - `openclaw-plugin.ts` 虽然物理上在 `adapter/src`，但逻辑归属优先算 `C`

这一步的目标是让后续 proposal 写作时，不再把整个 adapter 当一个黑盒。

### 4. 新增 proposal 模板
新增 `openspec/proposals/_template/`，至少包含：

- `spec.md`
- `tasks.md`
- `acceptance.md`
- `status.md`

模板要求固定这些字段：

- `spec.md`
  - Summary
  - Module Boundary
  - Problem Statement
  - Target Behavior
  - Implementation Decisions
  - Out of Scope
- `tasks.md`
  - 实现任务
  - 验证任务
  - 收尾任务
- `acceptance.md`
  - Automated
  - Manual
  - Commands
  - Evidence
  - Stop Conditions
- `status.md`
  - state
  - owner
  - modules
  - suggested_branch
  - suggested_worktree
  - dependencies
  - completion_gate

这样后续新增 proposal 时，粒度和质量更稳定。

### 5. 小幅更新 README 和运行手册
同步更新这几个入口文档：

- [`openspec/README.md`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec/README.md)
- [`openspec/integration/worktree-codex-playbook.md`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec/integration/worktree-codex-playbook.md)
- [`openspec/integration/module-worktree-board.md`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec/integration/module-worktree-board.md)

需要补的内容：

- proposal 只有在 `active` 时才推荐创建 worktree
- `draft` 的正常流程是先完善 spec，再切成 `active`
- `--force` 只用于特例，不是默认工作流
- 当前 `Adapter` 已经有正式的子模块分层，可用于继续拆 proposal

## Test Plan

### 1. Worktree 工具验证
对 [`openspec-worktree.sh`](/Users/med/Desktop/OpenAgent/openclaw-P2P/openspec-worktree.sh) 做这些检查：

- `bash -n` 通过
- `list` 正常
- `board` 正常
- `show 001` 正常
- `create 001` 在 `active` proposal 上行为正常
- `create 003` 在 `draft` proposal 上默认拒绝
- `create --force 003` 可显式放行
- 未提交 `openspec/` 时，仍然保持现有防护

### 2. 文档一致性检查
检查这些文档互相不打架：

- `README`
- `component-map`
- `module-worktree-board`
- `worktree-codex-playbook`
- 001-005 的 `status.md`

特别确认：

- `board` 里推荐启动的 proposal 与 `status.md` 的 `state` 一致
- Adapter 子模块划分和现有 proposal 的模块归属不冲突

### 3. Proposal 可执行性检查
对 001-005 快速人工审查：

- 每个 `acceptance.md` 都有 `Commands`
- 每个 `acceptance.md` 都有 `Evidence`
- 每个 `acceptance.md` 都有 `Stop Conditions`
- 另一个工程师只看 proposal 文件也知道怎么验证

## Assumptions

- 当前第一轮仍以 `001` 和 `002` 为主，不改变 proposal 启动顺序
- `003` 仍保持 `draft`，除非你明确决定把它升级成 `active`
- 不调整现有 proposal 编号
- 不推翻现有 7 大模块，只在 `Adapter` 模块内部补二级拆分
</proposed_plan>
