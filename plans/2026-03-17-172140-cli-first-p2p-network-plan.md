# CLI-First OpenClaw P2P Network Plan

- created_at: 2026-03-17 17:21:40 +08:00
- mode: Default (formal proposal)
- request: Build a usable CLI-first OpenClaw P2P runbook for new nodes to join, interact, and verify network behavior without Telegram.
- status: completed

## Plan

<proposed_plan>
目标：先落地“纯命令行可用”的 OpenClaw P2P 网络，再把 agent 接入交互；Telegram 集成放到后续阶段。

范围与约束：
1. 采用单引导节点拓扑（`cloud1` 作为 bootstrap），其它节点（如 `mabot`、`windows1`）通过统一 rendezvous + bootstrap multiaddr 加入网络。
2. 第一阶段不依赖 Telegram，不依赖 OpenClaw Runtime；只要求本仓库 OpenAgent 节点可运行并可通过 HTTP API 交互。
3. 验证标准以“跨节点可观测证据”为准，而不是仅本地单机成功。

实施项：
1. 产出 CLI 操作手册（新节点视角）
   - 说明 bootstrap 启动、worker 入网、常用 API 交互（publish/list/fetch/negotiate/complete/withdraw）。
   - 明确可接受最小拓扑、端口、rendezvous、topic 一致性要求。
   - 给出排障路径（空 peers、消息不传播、fetch 失败、协商无会话记录）。
2. 增加证据采集脚本
   - 从操作机对指定 host key 收集 `/healthz`、`/v1/tasks`、`/v1/facts`、`/v1/peers`、`/v1/sessions`。
   - 统一输出到 `.local/evidence/p2p-cli/<timestamp>/`，并记录成功/失败汇总，便于验收与回归对比。
3. README 入口补齐
   - 把 CLI runbook 和证据脚本作为主入口，降低新节点接入成本。
4. 计划归档
   - 按仓库规范在 `plans/` 保存本次执行计划和状态，确保后续可追溯。

验收标准：
1. 三节点（`cloud1`、`mabot`、`windows1`）均 `healthz` 正常。
2. worker 节点 `v1/peers` 非空并能看到远端 peer。
3. 至少一条任务跨节点可见。
4. 至少一次 `fetch` 成功返回远端 detail。
5. 至少一次 `negotiate` 往返在 `v1/sessions` 可见。
</proposed_plan>
