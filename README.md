# m9m

Automate repetitive work with one command. No servers, no dependencies, no Node.js runtime. Just a 30MB binary that runs your workflows.

Connect apps, process data, schedule tasks, and run AI agents -- all from your terminal.

[![Build Status](https://img.shields.io/github/actions/workflow/status/neul-labs/m9m/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/neul-labs/m9m/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/neul-labs/m9m?style=flat-square)](https://goreportcard.com/report/github.com/neul-labs/m9m)
[![Coverage](https://img.shields.io/codecov/c/github/neul-labs/m9m?style=flat-square&logo=codecov)](https://codecov.io/gh/neul-labs/m9m)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go)](https://pkg.go.dev/github.com/neul-labs/m9m)
[![Release](https://img.shields.io/github/v/release/neul-labs/m9m?style=flat-square&logo=github)](https://github.com/neul-labs/m9m/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)

---

## What You Can Automate

**Business Operations**
- Process new signups and route them to the right onboarding flow
- Score leads based on activity across multiple platforms
- Route invoices to approvers based on amount and department
- Segment customers by behavior and sync to your CRM

**Data Processing**
- Sync databases on a schedule with conflict resolution
- Merge reports from 5 different services into one dashboard
- Clean and validate incoming data before it hits your database
- Transform CSVs, JSON, and API responses into the format you need

**AI & Developer Workflows**
- Run Claude Code in a sandboxed environment to analyze codebases
- Process customer feedback with GPT-4 and route by sentiment
- Auto-create GitHub issues from support tickets
- Chain multiple AI models together with human review steps

**Scheduled Tasks**
- Back up critical data to S3 every night
- Generate and email weekly reports automatically
- Clean up expired records and notify stakeholders
- Monitor APIs and alert your team on Slack when something breaks

Browse ready-to-use examples in the [`examples/`](examples/) directory.

---

## See It In Action

### 1. Install

```bash
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash
```

Or download from [GitHub Releases](https://github.com/neul-labs/m9m/releases). Docker also available: `docker run -p 8080:8080 neul-labs/m9m:latest`

### 2. Run the demo

```bash
m9m demo
```

You'll see six workflows execute in milliseconds:

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

Demo 3: Multi-Branch Routing
  Pipeline:  Start -> Set -> Switch -> Set(x3)
  Status:    PASS
  Duration:  8ms

...

Results: 6/6 demos passed
```

### 3. Run your own workflow

```bash
m9m exec workflow.json
m9m exec workflow.json --input '{"customer_id": 42}'
```

List all 32 built-in nodes:

```bash
m9m node list
```

---

## 30+ Integrations

| What you need | What's available |
|---------------|------------------|
| **Store data** | PostgreSQL, MySQL, SQLite |
| **Cloud storage** | AWS S3, GCP Cloud Storage, Azure Blob |
| **Get AI help** | OpenAI (GPT-4), Anthropic Claude |
| **Notify people** | Slack, Discord, Email (SMTP) |
| **Manage code** | GitHub, GitLab |
| **Spreadsheets** | Google Sheets |
| **Read/write files** | Binary file I/O with encoding support |
| **Any API** | HTTP Request node + Webhooks |

Write custom logic in JavaScript or Python when you need it. See full node details with `m9m node list`.

---

## Why m9m?

**Zero infrastructure.** One binary, no Node.js, no database required. Download it. Run it. You're done. Under 500ms from cold start to executing workflows.

**Fast.** Workflows run 5-10x faster than Node.js-based alternatives. Process 500 items in 6 seconds, not 60. Run `m9m benchmark` to see the numbers on your own hardware.

**Free enterprise features.** Git-based workflow versioning, audit logs, multi-workspace support, and Prometheus metrics are included -- not locked behind a paid tier.

**Works with your existing n8n workflows.** Import them and run them unchanged (see below).

| | m9m | Typical Node.js alternative |
|---|---|---|
| **Startup time** | 500ms | 3s |
| **Memory usage** | ~150MB | ~512MB |
| **Container size** | 300MB | 1.2GB |
| **Concurrent workflows** | 500 | 50 |

---

## Works with n8n

Already using n8n? Import your workflows. They run unchanged, just faster.

```bash
m9m exec my-n8n-workflow.json
```

**What's compatible:**
- Workflow JSON format and node connections
- Expression syntax (`{{ $json.field }}`, `{{ $node["name"].data }}`)
- 32 node types covering the most-used n8n functionality
- Trigger types (webhooks, cron schedules)

See the [migration guide](docs/migration/) for details.

---

## Go Further

These features are available when you need them. Start simple, add complexity later.

- **Run AI agents in sandboxed environments** -- Claude Code, Codex, and Aider with resource limits and namespace isolation. [Learn more](docs/nodes/cli.md)
- **Embed in your app** -- Go, Python, and Node.js SDKs for programmatic workflow execution. [Learn more](docs/sdk/)
- **Enterprise monitoring** -- Prometheus metrics and OpenTelemetry tracing out of the box. [Learn more](docs/monitoring/)
- **Build custom nodes** -- Extend m9m with your own node types in Go. [Learn more](docs/nodes/README.md)
- **MCP integration** -- 37 tools for AI-powered workflow management via Claude Code. [Learn more](docs/mcp/README.md)

---

## Documentation

| Resource | Description |
|----------|-------------|
| [Getting Started](docs/README.md) | Quick start guide |
| [Architecture](docs/architecture/README.md) | System design |
| [API Reference](docs/api/API_COMPATIBILITY.md) | REST API documentation |
| [Node Development](docs/nodes/README.md) | Building custom nodes |
| [CLI Nodes](docs/nodes/cli.md) | CLI agent orchestration |
| [Deployment](docs/deployment/DEPLOYMENT_GUIDE.md) | Production deployment |
| [MCP Integration](docs/mcp/README.md) | Claude Code integration |

## Contributing

We welcome contributions! See our [Contributing Guide](docs/CONTRIBUTING.md) for details.

```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
make deps && make test && make build
```

## License

MIT License. See [LICENSE](LICENSE) for details.

---

[Documentation](https://docs.neullabs.com/m9m) | [GitHub](https://github.com/neul-labs/m9m) | [Report an Issue](https://github.com/neul-labs/m9m/issues)
