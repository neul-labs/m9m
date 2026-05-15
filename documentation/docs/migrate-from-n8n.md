---
title: Migrate from n8n to m9m
description: Step-by-step guide to migrating from n8n to m9m. Run existing n8n workflows unchanged, what's compatible, what's not, side-by-side testing.
keywords: migrate n8n, n8n to m9m, n8n migration, n8n alternative migration, n8n workflow export, n8n compatibility
---

# Migrate from n8n to m9m

m9m runs n8n workflow JSON unchanged. Migration is usually a one-command operation: `m9m exec my-n8n-workflow.json`. This page covers the details — what's compatible, what's not, and how to test confidently.

## TL;DR

```bash
# 1. Install m9m
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash

# 2. Export a workflow from n8n (Settings → Download)
# 3. Run it
m9m exec my-n8n-workflow.json
```

If the workflow uses only built-in n8n nodes, it just works. If it uses community nodes (`n8n-nodes-*`), see [Unsupported features](#unsupported-features) below.

## What runs unchanged

m9m targets ~95% backend feature parity with n8n. The following all work with no modification:

- **Workflow JSON format** — same shape, same node connections, same metadata.
- **Expression syntax** — `{{ $json.field }}`, `{{ $node["name"].data }}`, `={{ ... }}`, all built-in helpers (`$now`, `$workflow`, `$execution`, `$itemIndex`, `$runIndex`).
- **40+ built-in node types** — HTTP, databases, AI, messaging, email, file, cloud, VCS, productivity. Run `m9m node list` for the full catalog.
- **Trigger types** — webhooks, cron schedules, error triggers.
- **Credential formats** — n8n credential JSON works directly.
- **Environment variables** — n8n env var conventions are honored.
- **REST API surface** — n8n-compatible endpoints for workflow / execution / credential management.

## Unsupported features

Honest list. These are *not yet* in m9m:

| Feature | Status | Workaround |
|---|---|---|
| Community nodes (`n8n-nodes-*` from npm) | Not supported | File an issue, contribute a native Go node, or use HTTP Request to call the underlying API |
| n8n Cloud–specific features | Not supported | Self-host m9m instead |
| n8n web UI (visual editor) | m9m ships its own UI | Use m9m's UI or edit JSON directly |
| Live shared database with n8n | Not supported | Run side-by-side, migrate workflow-by-workflow |
| Sub-workflows that recurse community nodes | Not supported | Refactor to use built-in nodes |

Full feature matrix: [N8N_FEATURE_COMPARISON.md](https://github.com/neul-labs/m9m/blob/main/docs/N8N_FEATURE_COMPARISON.md).

## Step-by-step migration

### Step 1 — install m9m alongside n8n

m9m is a single binary, so running it alongside an existing n8n instance is harmless. Use a different port to avoid conflicts:

```bash
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash
M9M_PORT=8081 m9m serve
```

### Step 2 — export workflows from n8n

In the n8n UI: open the workflow, click the menu, choose **Download**. You get a `*.json` file.

Or use the n8n API:

```bash
curl http://localhost:5678/api/v1/workflows/{id} \
  -H "X-N8N-API-KEY: $N8N_KEY" \
  > my-workflow.json
```

### Step 3 — test with `m9m exec`

```bash
m9m exec my-workflow.json
```

`m9m exec` runs the workflow once with empty input and prints the result. If it fails with `node type not registered`, you're hitting a community node — see [Unsupported features](#unsupported-features).

### Step 4 — copy credentials

n8n credential JSON works directly. Either:

```bash
# CLI
m9m credential import n8n-credentials.json
```

…or copy them via the API:

```bash
curl -X POST http://localhost:8081/api/v1/credentials \
  -H "Content-Type: application/json" \
  -d @n8n-credentials.json
```

n8n encrypts credentials at rest with its `N8N_ENCRYPTION_KEY`. To import them as-is, set `M9M_ENCRYPTION_KEY` to the same value before importing. Otherwise, re-enter secret values after import.

### Step 5 — recreate triggers

Webhooks and cron triggers are workflow-internal — they migrate with the JSON. After the workflow is registered:

```bash
# Webhook will be available at http://localhost:8081/webhook/<path>
# Cron triggers start running on schedule once the workflow is enabled
```

If n8n was serving webhooks on `https://example.com/webhook/...`, update your DNS / proxy to point at m9m's port.

### Step 6 — run side-by-side for a week

Lowest-risk migration:

1. Keep n8n running.
2. Mirror webhook traffic to both n8n and m9m (e.g., with a small proxy that POSTs to both).
3. Diff the outputs / side effects daily.
4. When you have a week of clean parallel runs, switch DNS to m9m.
5. Decommission n8n.

### Step 7 — adopt m9m-specific features

Once migrated, you can use what m9m has that n8n doesn't:

- **MCP server** — let Claude Code or Cursor manage workflows directly. ([guide](https://github.com/neul-labs/m9m/blob/main/docs/mcp/README.md))
- **Sandboxed CLI agents** — Claude Code, Codex, Aider as workflow steps with resource limits.
- **Go / Node.js / Python SDKs** — embed the engine in your application.
- **Reproducible benchmarks** — `m9m benchmark`.

## Differences to expect

### Performance
Workflows that took seconds in n8n often take milliseconds in m9m. If you had latency-sensitive timing logic ("sleep 2 s between steps"), it still works — but the surrounding nodes return much faster.

### Memory
m9m's idle footprint is ~150 MB vs ~512 MB for n8n. Useful if you were pinning n8n at high replica counts for headroom.

### Determinism
m9m runs parallel branches as goroutines but joins are explicit. If your n8n workflow relied on accidental event-loop ordering (e.g., "branch A's downstream node runs before branch B's because A's HTTP is faster"), you'll need an explicit `Merge` to enforce ordering. This is usually a bug being fixed, not a regression.

### Logging format
m9m logs structured JSON by default; n8n logs text. Adjust your log pipeline accordingly. Set `M9M_LOG_FORMAT=text` if you need the n8n shape.

## FAQ

### Do I need to convert my n8n workflows?
No. `m9m exec workflow.json` runs them directly.

### Do credentials transfer?
Yes, if you use the same encryption key. Otherwise re-enter secrets.

### Can m9m and n8n share a database?
No — they have different schemas. Run them with separate stores.

### What if a node behaves differently?
Open an issue with both outputs ([GitHub Issues](https://github.com/neul-labs/m9m/issues)). m9m aims for parity, and divergence is a bug.

### What if a community node is missing?
File an issue describing the node and the use case. Many community nodes are thin HTTP-Request wrappers and can be replaced inline. For others, native Go ports are welcome contributions.

## See also

- [Why m9m?](why-m9m.md) — the full comparison
- [FAQ](faq.md) — common questions
- [Installation](getting-started/installation.md) — every install path
- [First Workflow](getting-started/first-workflow.md) — five-minute walkthrough
- [Feature comparison matrix](https://github.com/neul-labs/m9m/blob/main/docs/N8N_FEATURE_COMPARISON.md)
