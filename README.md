# m9m

> **The n8n alternative without the bugs — faster, more reliable workflow automation.**

m9m is an open-source workflow automation platform written in Go. It runs n8n workflow JSON unchanged, executes 5–10× faster, uses 70% less memory, and ships as a single 30 MB binary with zero runtime dependencies. No Node.js, no npm tree, no event-loop stalls.

[![Build Status](https://img.shields.io/github/actions/workflow/status/neul-labs/m9m/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/neul-labs/m9m/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/neul-labs/m9m?style=flat-square)](https://goreportcard.com/report/github.com/neul-labs/m9m)
[![Coverage](https://img.shields.io/codecov/c/github/neul-labs/m9m?style=flat-square&logo=codecov)](https://codecov.io/gh/neul-labs/m9m)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go)](https://pkg.go.dev/github.com/neul-labs/m9m)
[![Release](https://img.shields.io/github/v/release/neul-labs/m9m?style=flat-square&logo=github)](https://github.com/neul-labs/m9m/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)

```bash
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash
m9m demo
```

---

## Table of contents

- [What is m9m?](#what-is-m9m)
- [Why m9m? (vs n8n)](#why-m9m-vs-n8n)
- [30-second quickstart](#30-second-quickstart)
- [Install](#install)
- [Use cases](#use-cases)
- [Drop-in n8n compatibility](#drop-in-n8n-compatibility)
- [Ecosystem](#ecosystem)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

---

## What is m9m?

m9m is a cloud-native workflow automation engine — a faster, more reliable drop-in alternative to [n8n](https://n8n.io). You give it a workflow (a directed graph of nodes that fetch data, transform it, call APIs, run AI agents, or wait on schedules), and it executes that workflow with the same JSON format, expression syntax, and credentials that n8n uses.

The difference is the runtime. n8n is a Node.js application; m9m is a single statically-linked Go binary. That swap eliminates an entire class of production issues — JavaScript heap leaks, npm supply-chain risk, event-loop pauses under load, slow cold starts — while delivering 5–10× faster workflow execution on the same hardware.

m9m is open source under the MIT license, ships for macOS, Linux, and Windows on both AMD64 and ARM64, and integrates natively with Claude Code and other AI coding agents via the Model Context Protocol (MCP).

---

## Why m9m? (vs n8n)

| | m9m | n8n |
|---|---|---|
| **Cold start** | ~500 ms | ~3 s |
| **Memory (idle)** | ~150 MB | ~512 MB |
| **Container size** | ~300 MB | ~1.2 GB |
| **Workflow execution** | Baseline | 5–10× slower |
| **Concurrent workflows** | 500 | 50 |
| **Runtime** | Single static Go binary | Node.js + ~1,000 npm packages |
| **Deterministic execution** | Yes | No (event-loop ordering) |
| **n8n workflow JSON** | Runs unchanged | Native |
| **MCP server for Claude Code** | Built in (37 tools) | Not available |
| **License** | MIT | Sustainable Use License |

### Why we say *"without the bugs"*

Not a marketing line — three specific reliability properties that fall out of the runtime choice:

1. **No JavaScript-runtime memory leaks.** Go's garbage collector and value semantics eliminate the long-tail heap growth that requires regular n8n process restarts in production.
2. **Deterministic execution.** Same input, same output, every time — no event-loop ordering surprises or async timing edge cases. Workflows behave the same in test, staging, and prod.
3. **Smaller attack surface.** Single static binary with no transitive npm dependencies in the runtime path. CVEs in n8n's ~1,000-package dependency tree don't apply.

Performance numbers are reproducible on your hardware with `m9m benchmark`. The full methodology is in [docs/performance-report.md](docs/performance-report.md).

---

## 30-second quickstart

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash

# 2. Run the bundled demo (six workflows execute end-to-end)
m9m demo

# 3. Run your own workflow
m9m exec workflow.json
m9m exec workflow.json --input '{"customer_id": 42}'

# 4. List all built-in nodes
m9m node list
```

Output from `m9m demo`:

```
m9m Capability Demo
===================

Demo 1: E-Commerce Order Processing
  Pipeline:  Start -> Set -> Set -> Filter
  Status:    PASS
  Duration:  8ms

Demo 2: Customer Segmentation
  Pipeline:  Start -> Set -> Filter -> Set
  Status:    PASS
  Duration:  12ms

...

Results: 6/6 demos passed
```

---

## Install

| Surface | Command |
|---|---|
| **Homebrew (macOS / Linux)** | `brew tap neul-labs/tap && brew install m9m` |
| **Installer script (any platform)** | `curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh \| bash` |
| **Go** | `go install github.com/neul-labs/m9m/cmd/m9m@latest` |
| **Docker (GHCR)** | `docker run -p 8080:8080 ghcr.io/neul-labs/m9m:latest` |
| **Docker (Docker Hub)** | `docker run -p 8080:8080 neul-labs/m9m:latest` |
| **Python SDK** | `pip install m9m-cli` |
| **Node.js SDK** | `npm install m9m-cli` |
| **GitHub Releases** | [Download prebuilt binaries](https://github.com/neul-labs/m9m/releases) |

Supported platforms: macOS (Intel + Apple Silicon), Linux (AMD64 + ARM64), Windows (AMD64).

---

## Use cases

### AI agent orchestration
Run Claude Code, Codex, or Aider in sandboxed environments with resource limits and namespace isolation. Chain multiple AI models, add human-review steps, route by sentiment. See [docs/nodes/cli.md](docs/nodes/cli.md).

### Data operations
Sync databases on a schedule with conflict resolution. Merge reports from multiple SaaS sources into a single dashboard. Clean and validate incoming data before it hits production tables.

### Business automation
Process new signups and route to the right onboarding flow. Score leads from activity across platforms. Route invoices to approvers by amount and department.

### Scheduled jobs
Back up critical data to S3 nightly. Generate and email weekly reports. Monitor APIs and alert on Slack when something breaks.

Ready-to-run examples live in [`examples/`](examples/).

---

## Drop-in n8n compatibility

Already running n8n? Point m9m at your existing workflow JSON. No conversion step.

```bash
m9m exec my-n8n-workflow.json
```

**What runs unchanged:**

- Workflow JSON format and node connections
- Expression syntax (`{{ $json.field }}`, `{{ $node["name"].data }}`)
- 40+ node types covering the most-used n8n functionality
- Trigger types (webhooks, cron schedules)
- Credential formats and environment variables
- REST API endpoints (n8n-compatible surface)

**What doesn't (yet):**

- Custom community nodes from `n8n-nodes-*` npm packages
- n8n Cloud–specific features
- Live database sharing with an existing n8n instance
- The n8n web UI (m9m ships its own)

Full migration guide: [docs/migration/from-n8n.md](docs/migration/from-n8n.md).

---

## Built-in integrations

| Category | Nodes |
|---|---|
| **Databases** | PostgreSQL, MySQL, SQLite, MongoDB, Redis, Elasticsearch |
| **Cloud storage** | AWS S3, GCP Cloud Storage, Azure Blob |
| **AI / LLM** | OpenAI (GPT-4), Anthropic Claude |
| **Messaging** | Slack, Discord, Twilio, Microsoft Teams |
| **Email** | SMTP, SendGrid |
| **Version control** | GitHub, GitLab |
| **Productivity** | Notion, Stripe, Google Sheets |
| **CLI agents** | Claude Code, Codex, Aider (sandboxed) |
| **HTTP / Webhooks** | HTTP Request, Webhook trigger |
| **Logic** | Set, Filter, Code, Function, Merge, JSON, If, Loop, Cron |

Run `m9m node list` for the full catalog. Custom logic in JavaScript or Python when you need it.

---

## Ecosystem

- **CLI** — `m9m` binary, the primary surface. Serve, exec, schedule, MCP.
- **Go SDK** — embed the engine directly in a Go application. [Reference](https://pkg.go.dev/github.com/neul-labs/m9m).
- **Node.js SDK** — [`m9m-cli`](https://www.npmjs.com/package/m9m-cli) on npm. TypeScript-first.
- **Python SDK** — [`m9m-cli`](https://pypi.org/project/m9m-cli/) on PyPI. Full type hints, context-manager API.
- **MCP server** — 37 tools for Claude Code / Cursor / other MCP clients. [docs/mcp/README.md](docs/mcp/README.md).
- **Docker** — `ghcr.io/neul-labs/m9m` (releases) and `neul-labs/m9m` (Docker Hub).
- **Kubernetes** — Helm chart and example manifests in [deploy/](deploy/).

---

## FAQ

### Is m9m a drop-in replacement for n8n?
For workflow execution: yes. m9m runs n8n workflow JSON, expression syntax, and credentials unchanged across 40+ built-in node types. Community nodes published as `n8n-nodes-*` npm packages and n8n Cloud–specific features are not yet supported. See the [migration guide](docs/migration/from-n8n.md).

### How much faster is m9m than n8n?
On the same hardware: 5–10× faster workflow execution, ~6× faster cold start (~500 ms vs ~3 s), 70% lower memory (~150 MB vs ~512 MB), 75% smaller container (~300 MB vs ~1.2 GB). Reproducible with `m9m benchmark`. Full numbers and methodology in [docs/performance-report.md](docs/performance-report.md).

### Why is m9m more reliable than n8n?
m9m is a single statically-linked Go binary. Three concrete consequences: no long-running Node.js heap growth (no scheduled restarts), deterministic execution (no event-loop ordering surprises), and no npm transitive-dependency CVEs in the runtime path. The trade-off is that custom logic is sandboxed JavaScript (via Goja) or Python rather than arbitrary npm imports.

### Can I import existing n8n workflows?
Yes. Export the workflow JSON from n8n and run `m9m exec workflow.json`. No conversion step. Credentials use the same format.

### Does m9m work with Claude Code, Cursor, and other AI coding agents?
Yes, natively. m9m ships a built-in MCP (Model Context Protocol) server with 37 tools for workflow orchestration. Agents can list, create, execute, and inspect workflows directly. See [docs/mcp/README.md](docs/mcp/README.md).

### Does m9m run on Windows?
Yes. Prebuilt Windows AMD64 binaries are on every GitHub release. macOS (Intel + Apple Silicon) and Linux (AMD64 + ARM64) are also first-class.

### Is m9m production-ready?
Yes. Single static binary, Prometheus metrics, OpenTelemetry tracing, Git-based workflow versioning, audit logs, and multi-workspace support are all built in — not gated behind a paid tier. Production deployment guide: [docs/deployment/](docs/deployment/).

### Is m9m free and open source?
Yes. MIT-licensed. No "fair-use" clauses, no source-available restrictions, no commercial-use carveouts.

### How does m9m compare to Zapier or Make?
Zapier and Make are hosted SaaS platforms; m9m is self-hosted open-source software you run on your own infrastructure. The right comparison is to n8n, Airflow, or Temporal — and m9m is faster than all three for the integration-automation use case.

---

## Documentation

| Resource | Description |
|---|---|
| [Documentation site](https://docs.neullabs.com/m9m) | Full docs — installation, nodes, API, deployment |
| [Getting Started](docs/README.md) | Quick-start guide |
| [Why m9m?](docs/N8N_FEATURE_COMPARISON.md) | Feature comparison and gap analysis |
| [Performance report](docs/performance-report.md) | Benchmarks and methodology |
| [Migrate from n8n](docs/migration/from-n8n.md) | Step-by-step migration |
| [Architecture](docs/architecture/README.md) | System design |
| [API reference](docs/api/API_COMPATIBILITY.md) | REST API |
| [Node development](docs/nodes/README.md) | Build custom nodes |
| [Deployment](docs/deployment/DEPLOYMENT_GUIDE.md) | Production deployment |
| [MCP integration](docs/mcp/README.md) | Claude Code & MCP clients |

---

## Contributing

```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
make deps && make test && make build
```

See the [Contributing Guide](docs/CONTRIBUTING.md). Issues and discussions live on [GitHub](https://github.com/neul-labs/m9m/issues).

## Community

- [GitHub Discussions](https://github.com/neul-labs/m9m/discussions) — questions and design proposals
- [GitHub Issues](https://github.com/neul-labs/m9m/issues) — bugs and feature requests
- [Release Notes](https://github.com/neul-labs/m9m/releases) — changelog

## License

MIT License. See [LICENSE](LICENSE).

## Part of the Neul Labs toolchain

m9m is part of the Neul Labs orchestration toolchain:

| Project | Description |
|---------|-------------|
| [brat](https://github.com/neul-labs/brat) | Multi-agent harness for AI coding tools — crash-safe state, parallel execution. |
| [ringlet](https://github.com/neul-labs/ringlet) | One CLI to rule all your coding agents. |
| [fastworker](https://github.com/neul-labs/fastworker) | Background tasks in Python with zero infrastructure — no Redis, no RabbitMQ. |
| [conductor](https://github.com/neul-labs/conductor) | Multi-agent CLI orchestrator for AI coding agents. |

Learn more at [neullabs.com](https://www.neullabs.com).

---

[Documentation](https://docs.neullabs.com/m9m) · [GitHub](https://github.com/neul-labs/m9m) · [npm](https://www.npmjs.com/package/m9m-cli) · [PyPI](https://pypi.org/project/m9m-cli/) · [Docker](https://github.com/neul-labs/m9m/pkgs/container/m9m)
