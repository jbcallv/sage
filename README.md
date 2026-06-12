# Agentic Authorization Proxy

An eBPF-backed authorization proxy for multi-agent AI systems. The system enforces semantic authorization — not just *where* agents can connect, but *what they are permitted to do* — without modifying any agent framework.

## What It Does

Multi-agent AI systems (OpenClaw, LangGraph, AutoGen, CrewAI, etc.) suffer from a structural delegation gap: when an orchestrator agent delegates work to a sub-agent, authorization context is lost. A confused deputy attack exploits this — an injected instruction causes a sub-agent to perform an action neither the user nor the orchestrator authorized.

This system closes that gap by placing an authorization proxy between agents and everything they can reach:

- All outbound agent traffic is intercepted by an iptables redirect rule and routed through the proxy regardless of framework
- The proxy evaluates a Cedar policy on every request and injects a Macaroon delegation token on allow, or returns 403 on deny
- [Tetragon](https://tetragon.io) monitors all process execution inside containers at the eBPF level, supplying each agent's identity to the proxy and providing a kernel-level audit trail

The agent framework is never modified. Any HTTP-speaking agent works without configuration changes.

## Repository Layout

```
.
├── cmd/
│   ├── proxy/           authorization proxy (Go): intercept, enforce, forward
│   │   ├── main.go      server wiring and routes
│   │   ├── proxy.go     identify + permit (the macaroon ∩ policy intersection)
│   │   ├── forward.go   relays the request, attaches the credential
│   │   ├── http.go      health + /api/bootstrap handlers
│   │   ├── store.go     source IP → agent identity map
│   │   ├── config.go    key + policy-file configuration
│   │   └── Dockerfile   multi-stage build of the proxy image
│   └── listener/        Tetragon gRPC subscriber
│       ├── main.go      receives exec events, bootstraps each container
│       ├── identity.go  identity struct + container store
│       ├── docker.go    resolves a container's agent-net IP and role
│       ├── bootstrap.go POSTs identity to the proxy
│       └── token.go     mints the container's root macaroon
├── internal/
│   ├── macaroon/        mint · attenuate · serialize · verify
│   └── policy/          Cedar evaluation (cedar-go)
├── agents/
│   ├── orchestrator/    Stub orchestrator agent (demonstrates normal + injected calls)
│   └── worker/          Stub worker agent (the confused deputy victim)
├── stub-target/         Simple echo server (echoes path + received headers)
├── policies/
│   ├── agents.cedar     authorization policy the proxy enforces
│   └── exec-monitor.yaml  Tetragon TracingPolicy: logs all execve syscalls
├── scripts/
│   ├── start.sh         Start Tetragon, load policies, bring up containers
│   └── stop.sh          Tear everything down
├── docker-compose.yml   Network topology and service definitions
├── .env.example         Environment variable template
└── docs/                Research paper and Tetragon reference docs
```

## Prerequisites

| Dependency | Version | Notes |
|---|---|---|
| Docker | 29+ | `docker compose` v2 plugin required |
| Go | 1.25+ | For building the proxy and listener |
| Tetragon | 1.6+ | Must be installed on the host |

The proxy is built in Docker, so Go is only needed on the host to build the listener (which talks to the Tetragon socket).

### Install Go

```bash
curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
```

### Install Tetragon

Follow the [official docs](https://tetragon.io/docs/installation/) for your platform. On Ubuntu/Debian, the package installs to `/usr/bin/tetragon` and registers a systemd service.

## Quick Start

```bash
git clone <repo-url>
cd tetragon

# start Tetragon, load policies, bring up all containers
bash scripts/start.sh
```

That's it. The proxy starts on port 8080, the orchestrator runs its stub calls, and the worker listens for task injections.

## Running the Confused Deputy Test

This runs the full attack scenario end to end.

**Terminal 1 — watch proxy traffic:**
```bash
docker compose logs -f proxy
```

**Terminal 2 — watch Tetragon process events:**
```bash
go build -o listener ./cmd/listener && sudo ./listener
```

**Terminal 3 — trigger the attack:**
```bash
docker compose run --rm orchestrator
```

Expected output in terminal 3:
```
GET /hello -> 200: {"ok": true, "path": "/hello", "headers": {... "x-macaroon": "<base64 macaroon>", "x-agent-principal": "orchestrator"}}
injected task -> 200: {"url":"http://stub-target/admin/secret-data", "status": 403, "body": "{\"error\":\"forbidden\",\"principal\":\"worker\",...}"}
```

The first line is a legitimate call: Cedar permits `orchestrator → GET /hello`, so the proxy injects a Macaroon header and forwards it. You can see the injected `x-macaroon` and `x-agent-principal` headers echoed back by the stub target — that is Go-supplied identity reaching the request.

The second line is the confused deputy. The orchestrator's `POST /task` to the worker is permitted (the delegation itself is legitimate), so the worker receives it and tries to fetch `/admin/secret-data`. That downstream call has principal `worker`, which Cedar forbids — so the proxy returns **403 and never forwards it to the stub target**. The worker reports that inner `status: 403` back to the orchestrator. The attack is blocked by policy, at the exact hop where authorization is exceeded — not by network reachability.

Terminal 2 shows an `[exec]` line for every process that spawns inside any container. On the first exec per container the listener resolves that container's IP and role and POSTs it to the proxy's `/api/bootstrap`, which is how the proxy knows who each source IP is.

## Verifying Enforcement

The iptables DNAT rule is the enforcement boundary. These steps confirm it's working and show what changes once the proxy is down.

**With proxy running:**
```bash
docker compose run --rm orchestrator
```
Both calls succeed — the normal GET and the confused deputy injection.

**Stop just the proxy:**
```bash
docker compose stop proxy
```

**With proxy down:**
```bash
docker compose run --rm orchestrator
```
Both calls fail with a connection error — the DNAT rule has nowhere to forward traffic, so the kernel drops it.

This confirms the iptables rule is the choke point, not framework-level configuration. Restart the proxy with `docker compose start proxy` when done.

## Running Components Individually

**Containers only (Tetragon already running):**
```bash
docker compose up -d --build
```

**Check what the proxy is seeing:**
```bash
docker compose logs -f proxy
```

**Watch Tetragon events in real time:**
```bash
sudo tetra getevents -o compact
```

**Build and run the Go listener:**
```bash
go build -o listener ./cmd/listener && sudo ./listener
```

**Tear everything down:**
```bash
bash scripts/stop.sh
```

## Environment Variables

Copy `.env.example` to `.env` and set values as needed.

| Variable | Default | Description |
|---|---|---|
| `TETRAGON_SOCK` | `unix:///var/run/tetragon/tetragon.sock` | Tetragon gRPC socket path (listener) |
| `MACAROON_KEY` | `dev-insecure-key` | HMAC root key for minting/verifying macaroons; must match between listener and proxy |
| `POLICY_FILE` | `policies/agents.cedar` | Cedar policy the proxy enforces |
| `PROXY_URL` | `http://localhost:8080` | where the listener posts bootstrap identities |

The defaults work out of the box: the listener and proxy share the same default `MACAROON_KEY`, so minted tokens verify without any setup.

## How the Network Isolation Works

```
┌─ agent-net ───────────────────────────────────────────┐
│  orchestrator    worker    stub-target                 │
│       │             │           │                      │
│       └──────┬──────┘           │ (iptables intercepts │
└──────────────┼──────────────────┼─────────────────────┘
               │ all TCP          │ before it arrives
               ▼                  │
          [ proxy ] ◀─────────────┘
               │
           proxy-net ──▶ real external APIs
```

An iptables DNAT rule on the `agent-net` bridge interface redirects all TCP traffic (except the proxy's own outbound connections) to `proxy:8080`. Agents make normal HTTP calls with no proxy configuration — the kernel intercepts them transparently. Real external hostnames resolve via normal public DNS; only Docker container names require the target to be on `agent-net`. A Tetragon policy (`policies/exec-monitor.yaml`) provides a kernel-level audit trail of all process execution inside containers.

## Current Status

Working end to end (all Go):

- **Credential bootstrap** — the listener calls `POST /api/bootstrap` when Tetragon detects a new container, supplying its IP, role, and a freshly minted root macaroon
- **Real Macaroons** — `internal/macaroon` mints, attenuates, serializes, and verifies tokens; attenuation provably only narrows authority
- **Cedar policy** — `internal/policy` (cedar-go) evaluates `policies/agents.cedar` per request
- **Intersection enforcement** — the proxy permits a request only if the macaroon verifies *and* the local Cedar policy allows; deny returns 403, unknown sources fail closed
- **Header injection** — on allow, the proxy attaches `x-macaroon` and `x-agent-principal` before forwarding

Next:

- **Caveat-carried authority** — encode allowed operations as macaroon caveats so authority travels in the token itself (currently the token binds identity and Cedar carries the operation policy)
- **Parent → child attenuation** — Tetragon reports each process's parent and ancestors; mint child tokens as a strict subset of the parent's and expose a `parent → child` audit trail via `GET /api/audit`
- **Per-principal service instances** — have the ingress side spawn a per-caller container bound to the attenuated macaroon, so concurrent callers of the same service are isolated

## Research Context

See `docs/paper.txt` for the full background on the confused deputy problem in multi-agent systems and the authorization frameworks (Macaroons, WAVE, Cedar) this system draws from.
