# Module Worktree Assignment Plan

- created_at: 2026-03-13 16:21:18 +0800
- mode: Plan
- request: Split implementation by module across multiple Codex sessions and git worktrees
- status: completed

## Plan

<proposed_plan>
# 多 Codex / 多 Worktree 的模块分工方案

## Summary

先不围绕评审改论文，先按实现模块把工作拆给多个 `Codex + git worktree`。目标是让每个会话只负责一个清晰边界，不互相污染上下文，也不在同一个 worktree 里混多个 proposal。

当前最合理的执行单位不是“整个 adapter”这种大块，而是下面这 5 条实现线。建议先并行 2 到 3 条，不要一次全开。

## Worktree Assignment

### Worktree A: `001-detail-handle-resolution`
绑定模块：
- 3. OpenClaw Tool Plugin
- 4. Local State Store

职责：
- 修 `read_detail` 的 handle 解析
- 保证 `看看 T2 的详情` 一次成功
- 消除先失败再 fallback 到完整 `detail_ref` 的路径

对应 proposal：
- `openspec/proposals/001-detail-handle-resolution`

推荐分支 / worktree：
- `spec/001-detail-handle`
- `../wt-001-detail-handle`

适合第一个 Codex 立即开工。

### Worktree B: `002-pending-session-isolation`
绑定模块：
- 3. OpenClaw Tool Plugin
- 4. Local State Store
- 6. Agent Workspace Templates

职责：
- 消灭 `sessionKey=default` 的常态化使用
- 让 pending draft 按 Telegram 用户/会话隔离
- 把 `tg2` 的工具调用参数和插件 session 解析对齐

对应 proposal：
- `openspec/proposals/002-pending-session-isolation`

推荐分支 / worktree：
- `spec/002-pending-session-isolation`
- `../wt-002-session-isolation`

适合第二个 Codex 立即开工。

### Worktree C: `003-tasks-api-stability`
绑定模块：
- 1. Go OpenAgent Core

职责：
- 定位 `/v1/tasks` 偶发 `502`
- 加最小可复现检查
- 如果根因在 repo 内，就直接修服务端路径

对应 proposal：
- `openspec/proposals/003-tasks-api-stability`

推荐分支 / worktree：
- `spec/003-tasks-api-stability`
- `../wt-003-tasks-api-stability`

这是可选第三条线。它偏排障，不一定要和前两条同时启动，但如果你能多开一个 Codex，可以并行。

### Worktree D: `004-tg2-behavior-regression`
绑定模块：
- 6. Agent Workspace Templates
- 7. Acceptance and Ops Tooling

职责：
- 把本轮成功的 Telegram 验收整理成可重复回归
- 固化 preflight、reset、evidence、failure classification

对应 proposal：
- `openspec/proposals/004-tg2-behavior-regression`

推荐分支 / worktree：
- `spec/004-tg2-regression`
- `../wt-004-tg2-regression`

这条不要先开。它应该等 `001` 和 `002` 合并后再做，不然回归会固化旧缺陷。

### Worktree E: `005-cross-node-e2e-regression`
绑定模块：
- 1. Go OpenAgent Core
- 7. Acceptance and Ops Tooling

职责：
- 重新验证 `mabot -> cloud1` 的传播和 detail fetch
- 形成 fresh evidence pack

对应 proposal：
- `openspec/proposals/005-cross-node-e2e-regression`

推荐分支 / worktree：
- `spec/005-cross-node-e2e-regression`
- `../wt-005-cross-node-e2e`

这条放最后。它依赖本地主链和 API 读面先稳定。

## Recommended Start Order

### 第一轮并行
直接开：
- Worktree A = `001`
- Worktree B = `002`

可选再加：
- Worktree C = `003`

### 第二轮
等第一轮结果稳定后再开：
- Worktree D = `004`

### 第三轮
最后再开：
- Worktree E = `005`

## Terminal and Codex Layout

推荐布局：

- Terminal 1: 主仓库
  - 用于 `openspec` 管理、review、merge、总体验收
- Terminal 2: `../wt-001-detail-handle`
  - Codex 只做 `001`
- Terminal 3: `../wt-002-session-isolation`
  - Codex 只做 `002`
- Terminal 4: `../wt-003-tasks-api-stability`
  - Codex 只做 `003`，如果你决定并第三条
- 后续 Terminal 5/6 再给 `004/005`

约束：
- 一个终端只进一个 worktree
- 一个 Codex 会话只负责一个 proposal
- 主仓库不用来实现 proposal，只做调度和集成

## Codex Prompt Contract

每个新 Codex 会话的第一条指令固定成：

- 实现 `openspec/proposals/<id>-<name>`
- 先读 `spec.md`、`tasks.md`、`acceptance.md`、`status.md`
- 只做这个 proposal，不顺手改别的
- 完成后按 `acceptance.md` 自测
- 如发现新问题，拆成新 proposal，不要吸进当前 worktree

这样可以保证多个会话边界稳定。

## Test Plan

每条 proposal 完成时，先在各自 worktree 内完成：

- 代码改动只覆盖该 proposal
- 相关测试通过
- `acceptance.md` 里的检查做完
- `status.md` 更新到完成候选

主仓库集成时按这个顺序验：

1. 合并 `001`
2. 合并 `002`
3. 若已完成，再合并 `003`
4. 重新跑一次本地高层验收
5. 再启动 `004`
6. 最后跑 `005`

## Assumptions

- 当前 `openspec` 已经足够作为并行实现输入
- 第一轮最值得投入的是 `001` 和 `002`
- `004` 和 `005` 属于“在前面修复完成后再收口”的后置工作
- 主仓库继续充当调度中心，不承担任何单个 proposal 的实现
</proposed_plan>
