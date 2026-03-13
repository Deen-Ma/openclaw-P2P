# Multi Worktree Codex Execution Plan

- created_at: 2026-03-13 14:41:36 +0800
- mode: Plan
- request: Explain how to run multiple worktrees and Codex sessions in parallel, and whether proposals should be written first
- status: completed

## Plan

<proposed_plan>
# 多 Worktree + 多 Codex 的执行方式

## Summary

对，推荐顺序就是：

1. 先把模块拆清楚
2. 再把要并行做的 proposal 写到“可直接实现”的程度
3. 然后每个 proposal 开一个独立 `git worktree`
4. 每个 worktree 用一个独立终端 + 一个独立 Codex 会话去执行
5. 最后回到主仓库做验收、集成和合并

核心原则是：`一个 proposal = 一个 worktree = 一个 Codex 会话`。不要让一个会话同时负责多个 proposal。

## 执行方式

### 1. 先把 proposal 写到能直接干活
每个 proposal 至少要有这些内容：

- 目标和完成定义
- 涉及模块
- 明确要改的行为
- 不改什么
- 测试和验收标准
- 如果有依赖，要写清楚依赖谁先落地

判断标准很简单：
如果把 proposal 丢给另一个工程师，他不需要再问“到底怎么做”，那就够了。

### 2. 只把适合并行的 proposal 拉出去
不是所有 proposal 都该同时开工。先挑互相不强依赖的。

按你这个仓库，当前推荐：

- 先并行：
  - `001-detail-handle-resolution`
  - `002-pending-session-isolation`
- 暂时别抢跑：
  - `004-tg2-behavior-regression`
  - `005-cross-node-e2e-regression`
- `003-tasks-api-stability` 可以单独并行，但它偏排障，优先级可以排在前两条之后或并行第三条

### 3. 主仓库和 worktree 的职责分开
主仓库只做这些事：

- 维护 `openspec/`
- 看全局状态
- 做 proposal 调度
- 做最终集成和合并
- 跑跨 proposal 的总体验收

每个 worktree 只做一条 proposal：

- 改代码
- 跑该 proposal 自己的测试
- 记录该 proposal 的结果
- 不碰其他 proposal 的实现

### 4. 终端和 Codex 的推荐组织
建议至少这样开：

- Terminal 1：主仓库
  - 用途：看 `openspec`、检查全局、做最终 merge
- Terminal 2：`wt-001-detail-handle`
  - 一个 Codex，会话只做 `001`
- Terminal 3：`wt-002-session-isolation`
  - 一个 Codex，会话只做 `002`
- Terminal 4：如果你要并第三条，再给 `003`

不要在同一个 Codex 会话里来回切 proposal。
也不要让一个 worktree 同时实现两个 proposal。

### 5. 每个新 Codex 会话的启动方式
进入对应 worktree 后，第一条任务就应该非常明确，例如：

- “实现 `openspec/proposals/001-detail-handle-resolution`，只按这个 proposal 做，不要顺手改别的”
- “实现 `openspec/proposals/002-pending-session-isolation`，改完后按 acceptance 自测”

这样 Codex 会先读 proposal，再落代码，边界会很清楚。

### 6. 每条 proposal 完成后的回流顺序
每个 worktree 完成后，先在该 worktree 内完成：

- 代码修改
- 单元测试 / 局部验证
- proposal 对应 acceptance
- 把 proposal `status` 更新为完成候选

然后回到主仓库统一做：

- 拉最新主干
- 按依赖顺序合并
- 跑更高层回归
- 更新 `openspec` 状态
- 记录最终结果

## Important Changes / Interfaces

这套方式本身不要求先改业务接口，但流程上要固定两条规则：

- proposal 是实现输入，不是会后纪要
- merge 顺序服从 proposal 依赖，不服从“谁先写完”

建议采用的工作命名：

- branch：`spec/001-detail-handle`
- worktree：`../wt-001-detail-handle`

同理类推，编号必须稳定，避免后面混乱。

## Test Plan

开始并行前，先检查：

- proposal 是否已经写到可执行
- proposal 之间是否存在强依赖
- 每个 proposal 是否有独立 acceptance

每个 worktree 完成后检查：

- 只包含该 proposal 的改动
- 该 proposal 的测试通过
- 没有顺手修 unrelated 问题
- 能引用自己的 acceptance 证据

集成时再检查：

- merge 顺序符合依赖
- 高层回归重新跑一次
- 没有因为 proposal 并行而把接口打架

## Assumptions

- 你愿意同时开多个终端和多个 Codex 会话
- 当前 `openspec` 已经可以作为 proposal 的主入口
- 第一批并行目标以 `001` 和 `002` 为主
- 主仓库保留为调度与集成环境，不直接承担某个 proposal 的实现
</proposed_plan>
