# Acceptance

## Automated

- `go test ./...`

## Manual

1. Publish a fresh task or fact from `mabot`.
2. Confirm the object is visible on `mabot`.
3. Confirm the object is visible on `cloud1`.
4. Fetch the detail payload from `cloud1` using the published `detail_ref`.
5. Record timestamps, ids, handles, and logs on both hosts.

## Commands

```bash
go test ./...
./connect-host.sh mabot 'ts=$(date +%s); curl -s -X POST http://127.0.0.1:7401/v1/tasks/publish -H "Content-Type: application/json" -d "{\"task_id\":\"cross-node-$ts\",\"taxonomy\":\"task.help.math\",\"topics\":[\"openagent/v1/general\",\"openagent/v1/math\"],\"summary\":\"cross-node regression $ts\",\"detail\":{\"source\":\"openspec-005\",\"ts\":\"$ts\"},\"conf\":800}"'
./connect-host.sh mabot 'curl -s http://127.0.0.1:7401/v1/tasks'
./connect-host.sh cloud1 'curl -s http://127.0.0.1:7401/v1/tasks'
./connect-host.sh cloud1 'curl -s -X POST http://127.0.0.1:7401/v1/fetch -H "Content-Type: application/json" -d "{\"detail_ref\":\"<paste detail_ref here>\"}"'
```

## Evidence

- publish response from `mabot`
- `/v1/tasks` output from `mabot`
- `/v1/tasks` output from `cloud1`
- `/v1/fetch` output from `cloud1`
- acceptance notes with exact publish and fetch timestamps

## Stop Conditions

- stop if the fresh object never appears on `mabot` after publish
- stop if it appears on `mabot` but not on `cloud1`
- stop if `cloud1` sees the object but cannot fetch the referenced `detail_ref`

## Success Condition

The same published object is proven at all three stages:

- local publish on `mabot`
- remote visibility on `cloud1`
- remote payload fetch on `cloud1`
