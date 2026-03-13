# Acceptance

## Automated

- `go test ./...`
- a repeat-read check can hit `/v1/tasks` many times without a spurious `502`
- any added server test or harness is committed with the fix

## Manual

1. Seed the node with at least one active task and one withdrawn task.
2. Read `/v1/tasks` repeatedly in a loop.
3. Confirm the endpoint stays `200`.
4. If a failure occurs, capture the exact server log entry and response body.

## Commands

```bash
go test ./...
./connect-host.sh mabot 'for i in $(seq 1 100); do code=$(curl -s -o /tmp/openagent-v1-tasks.out -w "%{http_code}" http://127.0.0.1:7401/v1/tasks); echo "$i $code"; [ "$code" = "200" ] || { echo "--- body ---"; cat /tmp/openagent-v1-tasks.out; break; }; done'
./connect-host.sh mabot 'curl -s http://127.0.0.1:7401/v1/tasks | head -c 400 && printf "\n"'
```

## Evidence

- server logs around the `/v1/tasks` handler
- reproduction harness output
- acceptance notes referencing exact timestamps

## Stop Conditions

- stop on the first reproducible `502`
- stop if `/healthz` stays healthy while `/v1/tasks` alone fails and capture that mismatch
- stop if the proposal cannot distinguish a repo-owned server failure from an external runtime failure
