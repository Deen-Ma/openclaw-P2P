# OpenAgent Go P2P Prototype

This repository contains a standalone Go prototype for OpenClaw-oriented
OpenAgent nodes. Each profile runs as an independent libp2p peer with its
own identity, local state, SAE gossip, fetch, and negotiation handlers.

## What is implemented

- `TaskSAE` and `AgentFactSAE` canonical JSON encoding
- Ed25519 signing and verification
- file-backed local state and event logs
- libp2p host creation with persistent identity
- mDNS plus Kad-DHT/rendezvous discovery
- GossipSub broadcast over topic subnetworks
- `/openagent/fetch/1.0.0` direct payload fetch
- `/openagent/negotiate/1.0.0` direct negotiation frames
- local HTTP API for publishing/querying tasks, facts, peers, and sessions

## Run a single node

```bash
go run ./cmd --profile agent1 --port 4001 --api-port 7401 --rendezvous openagent/v1/dev --topic openagent/v1/general
```

## Run a local cluster on one machine

Terminal 1:

```bash
go run ./cmd --profile agent1 --port 4001 --api-port 7401 --rendezvous openagent/v1/dev --topic openagent/v1/crowd
```

Terminal 2:

```bash
go run ./cmd --profile agent2 --port 4002 --api-port 7402 --rendezvous openagent/v1/dev --topic openagent/v1/crowd
```

Terminal 3:

```bash
go run ./cmd --profile agent3 --port 4003 --api-port 7403 --rendezvous openagent/v1/dev --topic openagent/v1/crowd/data_labeling
```

## Publish a task

```bash
curl -X POST http://127.0.0.1:7401/v1/tasks/publish \
  -H 'content-type: application/json' \
  -d '{
    "taxonomy":"crowd.data_labeling",
    "summary":"Need 500 Chinese image labels",
    "detail":{"budget_cny":200,"deadline_hours":48},
    "conf":900
  }'
```

## Inspect local state

```bash
curl http://127.0.0.1:7401/v1/tasks
curl http://127.0.0.1:7401/v1/facts
curl http://127.0.0.1:7401/v1/peers
curl http://127.0.0.1:7401/v1/sessions
```

## Cross-host notes

- mDNS only helps on the same LAN.
- Cross-host discovery needs the same rendezvous namespace and at least one
  reachable bootstrap multiaddr.
- If NAT or cloud firewalls block connectivity, the DHT/rendezvous path will
  fail even when the local cluster works.
