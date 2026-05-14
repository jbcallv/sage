# Agentic Authorization Proxy

An eBPF-backed authorization proxy for multi-agent AI systems. The system enforces semantic authorization — not just *where* agents can connect, but *what they are permitted to do* — without modifying any agent framework.

## What It Does

Multi-agent AI systems (OpenClaw, LangGraph, AutoGen, CrewAI, etc.) suffer from a structural delegation gap: when an orchestrator agent delegates work to a sub-agent, authorization context is lost. A confused deputy attack exploits this — an injected instruction causes a sub-agent to perform an action neither the user nor the orchestrator authorized.

This system closes that gap by placing an authorization proxy between agents and everything they can reach:

- All outbound agent traffic is intercepted by an iptables redirect rule and routed through the proxy regardless of framework
- The proxy will verify Macaroon delegation tokens and evaluate Cedar policies on every request
- [Tetragon](https://tetragon.io) monitors all process execution inside containers at the eBPF level, providing a tamper-proof audit trail and kernel-level enforcement

The agent framework is never modified. Any HTTP-speaking agent works without configuration changes.

## Repository Layout

```
.
├── proxy/               FastAPI authorization proxy
│   └── src/main.py      HTTP middleware + passthrough (auth logic added next)
├── listener/            Go binary subscribing to Tetragon's gRPC event stream
│   └── main.go          Receives process exec events, will bootstrap agent credentials
├── agents/
│   ├── orchestrator/    Stub orchestrator agent (demonstrates normal + injected calls)
│   └── worker/          Stub worker agent (the confused deputy victim)
├── stub-target/         Simple echo server (stands in for an external API)
├── policies/            Tetragon TracingPolicy YAMLs
│   └── exec-monitor.yaml  Logs all execve syscalls across containers
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
| Go | 1.25+ | For building the listener |
| Python | 3.12+ | Used inside containers via uv |
| Tetragon | 1.6+ | Must be installed on the host |

### Install Go

```bash
curl -fsSL https://go.dev/dl/go1.24.3.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
```

### Install uv (Python package manager)

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
source ~/.local/bin/env
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
cd listener && go build -o listener . && sudo ./listener
```

**Terminal 3 — trigger the attack:**
```bash
docker compose run --rm orchestrator
```

Expected output in terminal 3:
```
GET /hello -> 200: {"ok": true, "path": "/hello"}
injected task -> 200: {"url":"http://stub-target/admin/secret-data", ...}
```

The second line is the confused deputy — the orchestrator injected a task that caused the worker to call `/admin/secret-data`. Both succeed because the proxy is currently a passthrough. Once Macaroon + Cedar are wired in, the second call will return 403.

Terminal 2 shows an `[exec]` line for every process that spawns inside any container, attributed by Docker container ID. This is the Tetragon event stream that will drive automatic credential bootstrap.

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
cd listener && go build -o listener . && sudo ./listener
```

**Tear everything down:**
```bash
bash scripts/stop.sh
```

## Environment Variables

Copy `.env.example` to `.env` and set values as needed.

| Variable | Default | Description |
|---|---|---|
| `TETRAGON_SOCK` | `unix:///var/run/tetragon/tetragon.sock` | Tetragon gRPC socket path |

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

This is the MVP scaffolding. The following components are stubbed and will be implemented next:

- **Macaroon token issuance and verification** — attenuated delegation tokens so sub-agents cannot exceed their originating scope
- **Cedar policy evaluation** — declarative per-role, per-action authorization rules
- **Credential bootstrap** — the Go listener will call `POST /api/bootstrap` when Tetragon detects a new agent container, automatically issuing credentials derived from the parent agent's token
- **Delegation chain logging** — full `parent → child` audit trail exposed via `GET /api/audit`

## Research Context

See `docs/paper.txt` for the full background on the confused deputy problem in multi-agent systems and the authorization frameworks (Macaroons, WAVE, Cedar) this system draws from.
