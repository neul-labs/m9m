---
title: m9m FAQ — frequently asked questions
description: Answers to common questions about m9m — n8n compatibility, performance vs n8n, reliability, MCP/Claude Code integration, deployment, licensing.
keywords: m9m faq, n8n alternative faq, n8n vs m9m, workflow automation faq, m9m performance, m9m reliability, m9m mcp, m9m claude code
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "FAQPage",
  "mainEntity": [
    {
      "@type": "Question",
      "name": "What is m9m?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "m9m is an open-source workflow automation platform written in Go. It runs n8n workflow JSON unchanged but executes 5–10× faster, uses 70% less memory, and ships as a single 30 MB binary with zero runtime dependencies."
      }
    },
    {
      "@type": "Question",
      "name": "Is m9m a drop-in replacement for n8n?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "For workflow execution: yes. m9m runs n8n workflow JSON, expression syntax, and credentials unchanged across 40+ built-in node types. Community nodes published as n8n-nodes-* npm packages and n8n Cloud–specific features are not yet supported."
      }
    },
    {
      "@type": "Question",
      "name": "Is m9m free and open source?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. MIT-licensed. No fair-use clauses, no source-available restrictions, no commercial-use carveouts."
      }
    },
    {
      "@type": "Question",
      "name": "How much faster is m9m than n8n?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "On the same hardware: cold start ~500 ms vs ~3 s (6× faster), idle memory ~150 MB vs ~512 MB (70% lower), container size ~300 MB vs ~1.2 GB (75% smaller), workflow execution 5–10× faster, concurrent workflows 500 vs 50. Reproducible with m9m benchmark."
      }
    },
    {
      "@type": "Question",
      "name": "Why is m9m faster than n8n?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Three reasons: Go is compiled and statically linked (no V8 warm-up, no JIT de-optimisation); goroutines on a real thread pool replace the single Node.js event loop so concurrent workflows scale; and there's no require() overhead, no node_modules walk, no npm install on the server."
      }
    },
    {
      "@type": "Question",
      "name": "Why do you say 'without the bugs'?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Three concrete reliability properties that come from the Go runtime: (1) no JavaScript-runtime memory leaks because Go's garbage collector eliminates long-running heap growth; (2) deterministic execution because the workflow runs as a topologically-sorted DAG instead of on an event loop; (3) a smaller attack surface because m9m is a single statically-linked binary without ~1,000 npm transitive dependencies."
      }
    },
    {
      "@type": "Question",
      "name": "Is m9m production-ready?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. Single static binary, Prometheus metrics, OpenTelemetry tracing, Git-based workflow versioning, audit logs, and multi-workspace support all ship in the open-source build — not gated behind a paid tier."
      }
    },
    {
      "@type": "Question",
      "name": "Can I import existing n8n workflows?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. Export the workflow JSON from n8n and run 'm9m exec workflow.json'. No conversion step. Credentials use the same format."
      }
    },
    {
      "@type": "Question",
      "name": "Which n8n nodes are supported?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "40+ built-in node types covering HTTP, databases (PostgreSQL, MySQL, SQLite, MongoDB, Redis, Elasticsearch), AI (OpenAI, Anthropic), messaging (Slack, Discord, Twilio, Teams), email (SMTP, SendGrid), cloud storage (S3, GCS, Azure Blob), version control (GitHub, GitLab), productivity (Notion, Stripe, Google Sheets), file I/O, webhooks, cron, Code, Function, Set, Filter, Merge, JSON, If, Loop."
      }
    },
    {
      "@type": "Question",
      "name": "What about community n8n nodes?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Community nodes published as n8n-nodes-* npm packages are not supported today. They are Node.js packages with arbitrary npm imports; m9m doesn't load arbitrary npm at runtime by design. If a community node is important, file an issue or contribute a native equivalent."
      }
    },
    {
      "@type": "Question",
      "name": "Does m9m work with Claude Code, Cursor, and other AI coding agents?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes, natively. m9m ships a built-in MCP (Model Context Protocol) server with 37 tools for workflow orchestration. Agents can list, create, execute, and inspect workflows directly."
      }
    },
    {
      "@type": "Question",
      "name": "Can I run AI agents inside a workflow?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. The cliExecute node runs Claude Code, Codex, Aider, or any CLI agent in a sandboxed environment using bubblewrap on Linux, namespace isolation, and resource limits."
      }
    },
    {
      "@type": "Question",
      "name": "Where can m9m run?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "macOS (Intel + Apple Silicon), Linux (AMD64 + ARM64), Windows (AMD64). Prebuilt binaries for all platforms on every GitHub release."
      }
    },
    {
      "@type": "Question",
      "name": "Does m9m need a database?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "For development: no — it uses SQLite by default. For production: PostgreSQL is recommended."
      }
    },
    {
      "@type": "Question",
      "name": "How does m9m compare to Zapier or Make?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Zapier and Make are hosted SaaS platforms; m9m is self-hosted open-source software you run on your own infrastructure. The right comparison is to n8n, Airflow, or Temporal — and m9m is faster than all three for the integration-automation use case."
      }
    },
    {
      "@type": "Question",
      "name": "How does m9m compare to Apache Airflow or Temporal?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Airflow is a Python DAG scheduler aimed at data engineering pipelines. Temporal is a durable workflow engine for long-running code-defined workflows. m9m is aimed at integration automation (API calls, webhooks, business workflows) with an n8n-compatible JSON interface."
      }
    },
    {
      "@type": "Question",
      "name": "Can I use m9m commercially?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. MIT license. Use it in commercial products, host it for clients, embed it in proprietary software — no restrictions."
      }
    },
    {
      "@type": "Question",
      "name": "Is there a paid tier with extra features?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "No. All features ship in the open-source build — Prometheus metrics, audit logs, multi-workspace, Git versioning, all of it. No 'enterprise edition' gating."
      }
    }
  ]
}
</script>

# Frequently asked questions

## General

### What is m9m?
m9m is an open-source workflow automation platform written in Go. It runs [n8n](https://n8n.io) workflow JSON unchanged but executes 5–10× faster, uses 70% less memory, and ships as a single 30 MB binary with zero runtime dependencies.

### Is m9m a drop-in replacement for n8n?
For workflow *execution*: yes. m9m runs n8n workflow JSON, expression syntax, and credentials unchanged across 40+ built-in node types. Community nodes published as `n8n-nodes-*` npm packages and n8n Cloud–specific features are not yet supported. See the [migration guide](migrate-from-n8n.md).

### Is m9m free and open source?
Yes. MIT-licensed. No "fair-use" clauses, no source-available restrictions, no commercial-use carveouts. The full source is at [github.com/neul-labs/m9m](https://github.com/neul-labs/m9m).

### Who builds m9m?
[Neul Labs](https://github.com/neul-labs) and an open-source community of contributors. Bug reports, PRs, and design proposals are all welcome on [GitHub](https://github.com/neul-labs/m9m).

### What does "m9m" stand for?
It's just a name. Pronounced "em-nine-em."

---

## Performance

### How much faster is m9m than n8n?
On the same hardware, in our benchmarks:

| Metric | m9m | n8n |
|---|---|---|
| Cold start | ~500 ms | ~3 s |
| Idle memory | ~150 MB | ~512 MB |
| Container size | ~300 MB | ~1.2 GB |
| Workflow execution | Baseline | 5–10× slower |
| Concurrent workflows | 500 | 50 |

Reproducible with `m9m benchmark`. Full methodology in the [performance report](https://github.com/neul-labs/m9m/blob/main/docs/performance-report.md).

### Why is m9m faster than n8n?
Three reasons: (1) Go is compiled and statically linked — no V8 warm-up, no JIT de-optimisation; (2) goroutines on a real thread pool replace the single Node.js event loop, so concurrent workflows scale; (3) no `require()` overhead, no `node_modules/` walk, no npm install on the server.

### How is "5–10× faster" measured?
End-to-end workflow execution time, same workflow JSON, same hardware, same inputs. The 5× / 10× spread reflects workflow shape: HTTP-bound workflows see less speedup (network dominates), CPU-bound transforms see more.

---

## Reliability

### Why do you say "without the bugs"?
Three concrete properties that come from the Go runtime, not from us being smarter:

1. **No JavaScript-runtime memory leaks.** Go's garbage collector + value semantics eliminate the long-running heap growth that's documented in n8n deployments and often requires nightly process restarts.
2. **Deterministic execution.** Same input → same output. No event-loop ordering surprises across runs.
3. **Smaller attack surface.** Single statically-linked binary, no transitive npm dependencies in the runtime path. CVEs in n8n's ~1,000-package dep tree don't apply.

It's a claim about runtime properties, not about bug count. See [Why m9m?](why-m9m.md#reliability-why-we-say-without-the-bugs) for the long version.

### Is m9m production-ready?
Yes. Single static binary, Prometheus metrics, OpenTelemetry tracing, Git-based workflow versioning, audit logs, and multi-workspace support are all built in — not gated behind a paid tier. Production deployment guide: [Deployment](deployment/index.md).

### Does m9m have automated tests?
Yes — ~46 internal Go packages with tests, all passing on every commit. Plus a smoke-test suite (`make smoke-test`) that runs eight end-to-end workflow scenarios against a real binary.

---

## n8n compatibility

### Can I import existing n8n workflows?
Yes. Export the workflow JSON from n8n and run `m9m exec workflow.json`. No conversion step. Credentials use the same format. Full walkthrough: [Migrate from n8n](migrate-from-n8n.md).

### Which n8n nodes are supported?
40+ built-in node types covering HTTP, databases (PostgreSQL, MySQL, SQLite, MongoDB, Redis, Elasticsearch), AI (OpenAI, Anthropic), messaging (Slack, Discord, Twilio, Teams), email (SMTP, SendGrid), cloud storage (S3, GCS, Azure Blob), version control (GitHub, GitLab), productivity (Notion, Stripe, Google Sheets), file I/O, webhooks, cron, Code, Function, Set, Filter, Merge, JSON, If, Loop. Run `m9m node list` for the full catalog. Full matrix: [N8N_FEATURE_COMPARISON.md](https://github.com/neul-labs/m9m/blob/main/docs/N8N_FEATURE_COMPARISON.md).

### Do n8n expressions work?
Yes. The full expression syntax is supported: `{{ $json.field }}`, `{{ $node["name"].data }}`, `={{ ... }}` for computed values, built-in functions, and helper objects (`$now`, `$workflow`, `$execution`, etc.).

### What about community n8n nodes (`n8n-nodes-*` npm packages)?
Not supported today. They're Node.js packages with arbitrary npm imports; m9m doesn't load arbitrary npm at runtime by design. If a community node is important to you, file an issue or contribute a native equivalent.

### What about n8n Cloud features?
n8n Cloud–specific features (hosted credentials, Cloud webhooks, etc.) aren't supported. m9m is self-hosted.

---

## AI agents and MCP

### Does m9m work with Claude Code, Cursor, and other AI coding agents?
Yes, natively. m9m ships a built-in MCP (Model Context Protocol) server with 37 tools for workflow orchestration. Agents can list, create, execute, and inspect workflows directly. See [the MCP integration guide](https://github.com/neul-labs/m9m/blob/main/docs/mcp/README.md).

### Can I run AI agents *inside* a workflow?
Yes. The `cliExecute` node runs Claude Code, Codex, Aider, or any CLI agent in a sandboxed environment (bubblewrap on Linux, namespace isolation, resource limits). See [Nodes › CLI Execution](nodes/cli.md).

### Which LLM providers are supported?
First-class nodes for OpenAI (GPT-4 and o-series) and Anthropic Claude. LiteLLM, Groq, Ollama, and other providers via HTTP Request. Roadmap includes additional native providers.

---

## Deployment

### Where can m9m run?
macOS (Intel + Apple Silicon), Linux (AMD64 + ARM64), Windows (AMD64). Prebuilt binaries for all platforms on every [GitHub release](https://github.com/neul-labs/m9m/releases).

### Does m9m need a database?
For dev: no — it uses SQLite by default. For production: PostgreSQL is recommended. Configuration: [database](configuration/database.md).

### Can I run m9m in Kubernetes?
Yes. Helm chart and example manifests in [deploy/](https://github.com/neul-labs/m9m/tree/main/deploy). Production guide: [Deployment › Kubernetes](deployment/kubernetes.md).

### What about Docker?
`ghcr.io/neul-labs/m9m:latest` (releases, multi-arch) or `neul-labs/m9m:latest` (Docker Hub, CI builds). See [Deployment › Docker](deployment/docker.md).

### Does m9m run on Windows?
Yes. Prebuilt Windows AMD64 binaries on every release.

### How do I monitor m9m in production?
Prometheus metrics on `/metrics` (port 9090 by default) and OpenTelemetry tracing. Both ship out of the box. See [Configuration › Server](configuration/server.md).

---

## SDKs

### How do I embed m9m in a Node.js app?
`npm install m9m-cli` — see the [npm package](https://www.npmjs.com/package/m9m-cli).

### How do I embed m9m in a Python app?
`pip install m9m-cli` — see the [PyPI package](https://pypi.org/project/m9m-cli/).

### How do I embed m9m in a Go app?
Import `github.com/neul-labs/m9m` directly. See the [Go reference](https://pkg.go.dev/github.com/neul-labs/m9m).

### Do the SDKs bundle a Node.js or Python runtime?
No. The SDKs are thin clients that auto-download the platform-native `m9m` Go binary and call it. Your app keeps its own runtime; m9m runs in a separate process.

---

## Comparisons

### How does m9m compare to Zapier or Make?
Zapier and Make are hosted SaaS platforms; m9m is self-hosted open-source software you run on your own infrastructure. The right comparison is to n8n, Airflow, or Temporal — and m9m is faster than all three for the integration-automation use case.

### How does m9m compare to Apache Airflow?
Different shape. Airflow is a Python-based DAG scheduler aimed at data engineering pipelines (Spark jobs, dbt runs, batch ETL). m9m is aimed at integration automation (API calls, webhooks, business workflows) with an n8n-compatible interface. Both can run workflows on schedules, but their out-of-the-box node libraries don't overlap much.

### How does m9m compare to Temporal?
Different shape. Temporal is a durable workflow engine where you write workflows as code (Go / TypeScript / Python / Java) and the platform handles retries, state, and durability over long time horizons (days, months). m9m runs n8n-style declarative workflow JSON and is optimised for shorter-running automation, not multi-day human-in-the-loop processes.

### How does m9m compare to GitHub Actions?
GitHub Actions runs CI/CD workflows tied to your repository. m9m runs general-purpose automation independent of any VCS. You can use both — m9m has GitHub and GitLab nodes for cross-pollination.

---

## Licensing and commercial use

### Can I use m9m commercially?
Yes. MIT license. Use it in commercial products, host it for clients, embed it in proprietary software — no restrictions.

### Can I modify the source?
Yes. MIT license. Just retain the copyright notice in distributed source.

### Is there a paid tier with extra features?
No. All features ship in the open-source build — Prometheus metrics, audit logs, multi-workspace, Git versioning, all of it. No "enterprise edition" gating.

### Is there commercial support?
Reach out to [Neul Labs](https://github.com/neul-labs) for commercial support arrangements.

---

## Troubleshooting

### Where do I report bugs?
[GitHub Issues](https://github.com/neul-labs/m9m/issues).

### Where do I ask questions?
[GitHub Discussions](https://github.com/neul-labs/m9m/discussions).

### How do I see what m9m is doing?
`m9m exec workflow.json --debug` enables verbose logging. For a running server, set `M9M_LOG_LEVEL=debug`.

### My workflow fails with "node type not registered" — what do I do?
Run `m9m node list` to see what's available. The error usually means the workflow uses a community node (`n8n-nodes-*`) that isn't part of the built-in catalog.

### Where's the data stored?
`~/.m9m/data/` by default. SQLite by default; PostgreSQL configurable. Override with `M9M_DATA_DIR`.
