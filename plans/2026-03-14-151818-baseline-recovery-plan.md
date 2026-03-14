# 001 and 002 Baseline Recovery Plan

- created_at: 2026-03-14 15:18:18 +0800
- mode: Plan
- request: Correct 001 and 002 so they become mergeable proposals again
- status: completed

## Plan

<proposed_plan>
# 纠正 `001/002` 并进入可合并状态的执行计划

## Summary

当前 `001` 和 `002` 都已经有实现提交，但它们不是可直接合并的 proposal 分支。问题有两个：

- 两个分支都把未入库的 `adapter` / `workspace-templates` 基线整体带进来了，导致 proposal diff 严重重叠，破坏了并行 worktree 的边界。
- 两个 proposal 都提前标成了 `completed`，但真实验收证据还没有完全满足各自 `acceptance.md`。

下一步按“两阶段收口”处理：

1. 先抽出一个单独的 baseline 提案，把共享基线一次性入库。
2. 再把 `001`、`002` 重做成只保留各自行为差异的干净分支，并按各自 acceptance 收口。

## Key Changes

### 1. 新建 baseline proposal，吸收共享基线
新建一个 proposal，建议命名为 `000-adapter-workspace-baseline`。

它只负责把这次两个 worktree 里重复出现的共享基线正式入库，包括：
- `adapter/` 当前正式代码与测试基线
- `workspace-templates/` 的当前正式模板基线
- 必要的 `.gitignore` 基线调整
- 不包含 `001` 的 detail-handle 修复
- 不包含 `002` 的 session-isolation 修复

这个 proposal 的完成定义是：
- 共享目录正式进入主干
- 不引入 proposal 专属行为修复
- `cd adapter && npm test` 作为 baseline 测试通过

实现方式：
- 从 `wt-001` / `wt-002` 中抽取公共文件集合
- 明确排除：
  - `001` 专属逻辑和测试
  - `002` 专属逻辑和测试
  - proposal `status.md` 的 completed 结论

### 2. 重置 `001` 和 `002` 的 proposal 边界
在 baseline 落地后，重新整理两个 proposal：

- `001-detail-handle-resolution`
  - 只保留 `read_detail` 的 handle 解析修复
  - 只保留对应单测和 proposal 状态更新
  - 不再携带整套 adapter 基线导入

- `002-pending-session-isolation`
  - 只保留 sessionKey 解析优先级修复
  - 只保留 store/session 隔离测试
  - 只保留 `tg2` workspace 中与 canonical session key 相关的规则变化
  - 不再携带整套 adapter 基线导入

建议做法不是在现有脏 diff 上继续修，而是：
- baseline 合并后
- 让 `001`、`002` 各自从最新 `main` 重新建立或重写分支
- 用挑拣式方式只带回 proposal 专属改动

### 3. 修正 proposal 状态和完成门槛
在真实验收完成前，不允许把 `001`、`002` 标成 `completed`。

状态规则改成：
- baseline 完成后，`001` / `002` 回到 `active`
- 只有在各自 `acceptance.md` 满足后，才改成 `completed`

具体要求：
- `001`
  - 自动化通过
  - 真实 Telegram `看看 T2 的详情` 一次成功
  - 没有 failed-then-retry 的 `fetch_detail`
- `002`
  - 自动化通过
  - 至少完成一次真实双会话或等效 host-side 可证明会话隔离验证
  - 没有新的 stateful call 优先落到 `sessionKey=default`

### 4. 合并顺序固定
后续集成顺序固定为：

1. `000-adapter-workspace-baseline`
2. `001-detail-handle-resolution`
3. `002-pending-session-isolation`
4. `004-tg2-behavior-regression`

`003` 和 `005` 继续保持独立，不插入这条链路里。

## Public Interfaces / Types

这轮规划里不新增外部 API，但要固定 proposal 边界上的接口约束：

- baseline proposal 不得改变 `openagent_network` 工具行为语义
- `001` 只改变 `read_detail` 的输入归一化与 handle 解析路径
- `002` 只改变 session key 解析优先级和 workspace 传参规则
- 不在这轮引入新的工具动作、pending 文件格式或 OpenAgent HTTP schema 变化

## Test Plan

### Baseline
- `cd adapter && npm test`
- 校验 baseline 分支不包含 `001/002` 专属行为断言
- 校验 diff 只表现为基线导入，而不是功能修复混入

### `001`
- `cd adapter && npm test`
- 真实 Telegram:
  - `看看 T2 的详情`
- 验证：
  - 一次成功返回 detail
  - `adapter-events.jsonl` 只有成功 `fetch_detail`
  - 无同请求前置失败

### `002`
- `cd adapter && npm test`
- host/Telegram 会话隔离验证：
  - A 起草，B 起草
  - A 只能确认 A
  - B 只能取消 B
- 验证：
  - `pending-network-actions.json` 按会话隔离
  - session log 优先使用 `telegram:tg2:dm:<sender_id>`
  - 没有新的高优先级 `default`

### Merge readiness
- 每个 proposal 相对 `main` 的 diff 必须只包含本 proposal 变化
- 不允许再出现 `001` / `002` 都大面积新增同一批 baseline 文件

## Assumptions

- 当前最合理的修法是先补一个 baseline proposal，而不是强行在现有两个分支上做大范围手工拆 diff。
- `adapter/` 和 `workspace-templates/` 现在应被视为共享基础资产，不属于 `001` 或 `002` 任一单独 proposal。
- `004-tg2-behavior-regression` 应等 baseline + `001` + `002` 都落稳后再启动，否则会把当前边界混乱状态固化成回归基线。
</proposed_plan>
