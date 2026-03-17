# 给其他 PC 上 Codex 的执行提示词

复制下面整段给目标机器的 Codex：

```text
你要操作的仓库是：
- https://github.com/Deen-Ma/openclaw-P2P

目标：把本机作为 OpenClaw P2P 的一个 CLI-only 节点接入现有网络，并完成最小交互验证。请直接执行，不要只给建议。

约束：
1) 不接入 Telegram，不修改业务代码，只做部署运行与验证。
2) 使用 main 分支最新代码。
3) 命令失败时，先自查并重试一次，再给出明确报错。

执行步骤：
1. 准备仓库并确认公开可访问：
   - 若本机无仓库目录，先执行：git clone https://github.com/Deen-Ma/openclaw-P2P.git
   - cd openclaw-P2P
   - gh repo view Deen-Ma/openclaw-P2P --json visibility,url
   - 要求 visibility=PUBLIC，url=https://github.com/Deen-Ma/openclaw-P2P
2. 更新代码并构建：
   - git fetch origin
   - git checkout main
   - git pull --ff-only origin main
   - go mod download
   - go build ./cmd（Windows 用 go build .\cmd）
3. 启动节点（若我是工作节点）：
   - go run ./cmd --profile <本机profile> --port 4001 --api-port 7401 --rendezvous openagent/v1/dev --topic openagent/v1/general --bootstrap "/ip4/<cloud1_ip>/tcp/4001/p2p/<cloud1_peer_id>"
   若我是 cloud1，引导节点命令不带 --bootstrap。
4. 启动后立即检查：
   - curl -s http://127.0.0.1:7401/healthz
   - curl -s http://127.0.0.1:7401/v1/peers
5. 执行一次交互验证：
   - 发布一条测试任务到 /v1/tasks/publish（conf=900）
   - 查询 /v1/tasks 确认本机有记录
6. 输出结果：
   - 明确给出本机 peer_id
   - 列出 healthz 结果、peers 数量、测试 task_id
   - 若失败，说明失败点和下一步修复建议（最多 3 条）

最后不要改动 git 历史，不要提交，不要 push。
```
