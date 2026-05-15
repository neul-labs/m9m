---
title: Why m9m? The n8n alternative without the bugs
description: Detailed comparison of m9m vs n8n. Performance benchmarks, reliability evidence (no JS heap leaks, deterministic execution, smaller attack surface), feature parity.
keywords: n8n alternative, n8n replacement, n8n vs m9m, n8n performance, n8n bugs, n8n memory leak, workflow automation comparison, faster than n8n
---

# Why m9m?

**The n8n alternative without the bugs — faster, more reliable workflow automation.**

m9m is a drop-in replacement for n8n's workflow execution engine, written from scratch in Go. It runs the same workflow JSON, the same expressions, and the same credentials — but eliminates an entire class of production issues that come with the Node.js runtime.

This page lays out the case: what's faster, what's more reliable, and what stays the same.

## Performance: m9m vs n8n

All numbers are from [`docs/performance-report.md`](https://github.com/neul-labs/m9m/blob/main/docs/performance-report.md) and reproducible on your hardware with `m9m benchmark`.

| Metric | m9m | n8n | Improvement |
|---|---|---|---|
| **Cold start** | ~500 ms | ~3 s | **6× faster** |
| **Idle memory** | ~150 MB | ~512 MB | **70% lower** |
| **Container size** | ~300 MB | ~1.2 GB | **75% smaller** |
| **Binary / image footprint** | ~15 MB binary | ~200 MB image base | **93% smaller** |
| **Workflow execution (simple)** | Baseline | 5–10× slower | **5–10× faster** |
| **Concurrent workflows (single node)** | 500 | 50 | **10×** |
| **Simple HTTP request workflow** | 211 ms | ~500 ms | **2.4× faster** |
| **5 concurrent HTTP workflows** | 1.64 s | ~2.5 s | **1.5× faster** |
| **10 concurrent HTTP workflows** | 3.83 s | ~5 s | **1.3× faster** |

### Why is m9m faster?

- **No interpreter, no event loop.** m9m executes workflows as Go goroutines on a real thread pool. n8n executes them on a single Node.js event loop with cooperative async scheduling.
- **Compiled, not JIT'd.** Go's ahead-of-time compilation means no V8 warm-up cost, no inline-cache misses, no de-optimisation pauses.
- **Goroutines are cheap.** A goroutine costs ~2 KB of stack. A Node.js promise chain on a hot workflow holds far more memory in closures and microtask queues.
- **Static linking.** The whole runtime is one binary. No module-resolution overhead, no `require()` cost per invocation.

## Reliability: why we say *"without the bugs"*

Not marketing. Three concrete properties that follow from the runtime choice.

### 1. No JavaScript-runtime memory leaks

Long-running n8n processes grow their heap over time — a known pattern that's documented by the n8n team and one of the most common reasons production deployments add a nightly process restart. The causes are familiar to every Node.js shop: closures capturing references in callback chains, prototype chain modifications, the V8 heap fragmenting under bursty allocation.

m9m runs on Go's garbage collector with concurrent mark-and-sweep. Memory pressure from a workflow burst is released back to the OS once the burst subsides. There is no scheduled-restart pattern in m9m deployments.

### 2. Deterministic execution

Same input, same workflow, same output. Every time.

Node.js workflows are subject to event-loop ordering: a promise that resolves "first" depends on when its `setImmediate` was queued relative to I/O completions and timer callbacks. Two consecutive runs of the same n8n workflow can produce subtly different ordering — usually harmless, occasionally not (e.g., when two parallel branches both write to the same downstream node and the merge depends on arrival order).

m9m's execution model is a topologically-sorted DAG. Parallel branches run as goroutines, but joins are explicit. Output of a given workflow run is a deterministic function of its input. This matters most for testing (golden tests stay green) and for workflows that interact with external systems where ordering implies state.

### 3. Smaller attack surface

The n8n npm package transitively pulls ~1,000 packages. Every one is a potential source of a supply-chain CVE, a malicious-update incident, or a license compliance surprise. The n8n team can't audit them all, and neither can you.

The m9m runtime is a single statically-linked Go binary. Its dependency graph is `go.mod` — about 30 direct dependencies, all visible, all auditable, all compiled into the binary at build time. No npm install runs in production. No `node_modules/`. The whole runtime is an artifact you can SHA-pin.

For workflows that need custom JavaScript or Python, m9m runs that code in a sandboxed interpreter (Goja for JS, embedded for Python) — not by importing arbitrary npm packages at runtime.

### What about bugs in m9m itself?

m9m is open-source, MIT-licensed, ~80K LoC of Go, with 90%+ test coverage on the core engine and ~46 internal packages all passing tests on every commit. Bug reports go to [GitHub Issues](https://github.com/neul-labs/m9m/issues). The trade-off we're making is *fewer surprises in production*, not *zero defects ever* — which would be a claim no software project can honestly make.

## Compatibility: what runs unchanged

m9m targets *95% backend feature parity* with n8n. The practical translation: most production workflows work without modification.

**Runs unchanged:**

- Workflow JSON format and node connections
- Expression syntax (`{{ $json.field }}`, `{{ $node["x"].data }}`, `={{ ... }}`)
- 40+ built-in node types (HTTP, databases, AI, messaging, file, cloud storage, etc.)
- Trigger types (webhooks, cron schedules)
- Credential formats
- Environment variables
- REST API endpoints (n8n-compatible surface)

**Doesn't run (yet):**

- Custom community nodes published as `n8n-nodes-*` npm packages
- n8n Cloud–specific features (n8n Cloud webhooks, hosted credentials)
- Live database sharing with an existing n8n instance
- The n8n web UI (m9m ships its own UI)

Full feature matrix: [N8N_FEATURE_COMPARISON.md](https://github.com/neul-labs/m9m/blob/main/docs/N8N_FEATURE_COMPARISON.md). Migration walkthrough: [Migrate from n8n](migrate-from-n8n.md).

## What m9m has that n8n doesn't

- **Native MCP server** — 37 tools for Claude Code, Cursor, and other Model Context Protocol clients. AI agents can list, create, execute, and inspect workflows directly. ([docs/mcp/README.md](https://github.com/neul-labs/m9m/blob/main/docs/mcp/README.md))
- **CLI agent sandboxing** — bubblewrap-based isolation for running Claude Code, Codex, or Aider as workflow steps.
- **First-class Go SDK** — embed the engine in any Go application.
- **MIT license** — no Sustainable Use License caveats, no commercial-use carveouts.
- **Reproducible benchmarks** — `m9m benchmark` runs the full suite on your hardware.
- **Deterministic execution by construction** — see above.

## When *not* to use m9m

Honest answer:

- **You rely on community n8n nodes.** If your workflows depend on `n8n-nodes-someprovider` from npm, m9m doesn't have them. File an issue or contribute the node.
- **You need the n8n web UI's visual editor.** m9m ships its own UI; it's not pixel-identical to n8n's.
- **You're paying for n8n Cloud and that's working.** m9m is self-hosted. If managed n8n meets your needs, the migration ROI is low.
- **You don't have a Linux/macOS/Windows target.** m9m runs everywhere Go runs, but if you're on an exotic platform, check the [release matrix](https://github.com/neul-labs/m9m/releases) first.

## How to evaluate m9m for your workflows

1. **Export a representative workflow from n8n** — pick something with the nodes you actually use.
2. **`m9m exec workflow.json`** — verify it runs unchanged.
3. **Run `m9m benchmark` on your hardware** — compare against your n8n instance under the same load.
4. **Side-by-side for a week** — point a webhook at both, compare outputs and latencies.
5. **Migrate when comfortable** — [migration guide](migrate-from-n8n.md).

## See also

- [Migrate from n8n](migrate-from-n8n.md) — step-by-step
- [Performance report](https://github.com/neul-labs/m9m/blob/main/docs/performance-report.md) — full methodology
- [Feature comparison](https://github.com/neul-labs/m9m/blob/main/docs/N8N_FEATURE_COMPARISON.md) — node-by-node matrix
- [FAQ](faq.md) — common questions
