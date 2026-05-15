# m9m-cli

> **The n8n alternative without the bugs — faster, more reliable workflow automation.**

Node.js / TypeScript bindings for [m9m](https://github.com/neul-labs/m9m), a drop-in n8n alternative written in Go. 5–10× faster execution, 70% lower memory, deterministic runs, single static binary. No Node.js runtime overhead on the server — the engine is a Go binary that this package downloads on install and calls natively.

[![npm version](https://img.shields.io/npm/v/m9m-cli.svg?style=flat-square)](https://www.npmjs.com/package/m9m-cli)
[![npm downloads](https://img.shields.io/npm/dm/m9m-cli.svg?style=flat-square)](https://www.npmjs.com/package/m9m-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://github.com/neul-labs/m9m/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/neul-labs/m9m?style=flat-square)](https://github.com/neul-labs/m9m/releases)

```bash
npm install m9m-cli
```

---

## What you get

- **n8n-compatible workflow execution** — runs n8n workflow JSON, expressions, and credentials unchanged.
- **Single binary, auto-downloaded** — `m9m-cli` fetches the platform-native `m9m` binary on install (macOS Intel + Apple Silicon, Linux AMD64 + ARM64, Windows AMD64). No Docker, no separate server.
- **TypeScript-first API** — full type definitions for workflows, nodes, data items, results.
- **In-process or remote** — embed the engine directly in your Node.js app, or point the client at a `m9m serve` instance.
- **Custom nodes in JS / TS** — register your own node types with a single function.

---

## Install

```bash
npm install m9m-cli
# or
yarn add m9m-cli
pnpm add m9m-cli
bun add m9m-cli
```

The `m9m` binary is downloaded automatically on `npm install` to `~/.m9m/bin/m9m`. If you're behind a corporate proxy or in an air-gapped CI environment, see [Pinning and offline use](#pinning-and-offline-use) below.

---

## 30-second quickstart

```typescript
import { WorkflowEngine, Workflow } from 'm9m-cli';

const engine = new WorkflowEngine();

// Load an n8n-compatible workflow file
const workflow = Workflow.fromFile('./my-workflow.json');

// Execute with input
const result = await engine.execute(workflow, [
  { json: { email: 'user@example.com', name: 'Alice' } }
]);

console.log(result.data);
if (result.error) console.error(result.error);
```

Build a workflow programmatically:

```typescript
import { createWorkflow, WorkflowEngine } from 'm9m-cli';

const workflow = createWorkflow({
  name: 'Email Notification',
  nodes: [
    { name: 'Start', type: 'n8n-nodes-base.start', parameters: {} },
    {
      name: 'Send Email',
      type: 'n8n-nodes-base.emailSend',
      parameters: {
        to: '={{ $json.email }}',
        subject: 'Welcome!',
        text: 'Hello {{ $json.name }}',
      },
    },
  ],
  connections: {
    Start: { main: [[{ node: 'Send Email', type: 'main', index: 0 }]] },
  },
});

const result = await new WorkflowEngine().execute(workflow);
```

Register a custom node:

```typescript
const engine = new WorkflowEngine();

engine.registerNode('custom.uppercase', (input) =>
  input.map((item) => ({
    json: { ...item.json, text: String(item.json.text).toUpperCase() },
  }))
);
```

---

## API at a glance

```typescript
import {
  WorkflowEngine,   // load + execute workflows
  Workflow,         // workflow definition
  createWorkflow,   // builder
  DataItem,         // node-to-node data
  ExecutionResult,  // execute() return
  CredentialManager,// credential storage
  version,          // binary version
  downloadBinary,   // pin a version
  getBinaryPath,    // ~/.m9m/bin/m9m
} from 'm9m-cli';
```

Full reference: [docs.neullabs.com/m9m/sdk](https://docs.neullabs.com/m9m).

---

## Supported platforms

| OS | Architecture |
|---|---|
| macOS | Intel (AMD64), Apple Silicon (ARM64) |
| Linux | AMD64, ARM64 |
| Windows | AMD64 |

The correct binary is selected automatically by `process.platform` and `process.arch` on install.

---

## Why m9m? (the short version)

- **Faster** — sub-second cold start, 5–10× workflow execution, 75% smaller container than n8n.
- **More reliable** — single static binary, no Node.js heap leaks, deterministic execution, no npm transitive-dep CVEs in the runtime path.
- **Drop-in compatible** — n8n workflow JSON, expressions, and credentials run unchanged.

Full comparison + benchmark methodology: [github.com/neul-labs/m9m](https://github.com/neul-labs/m9m#why-m9m-vs-n8n).

---

## Pinning and offline use

### Pin the binary version

```bash
M9M_VERSION=v0.2.1 npm install m9m-cli
```

Or programmatically:

```typescript
import { downloadBinary } from 'm9m-cli';
await downloadBinary('v0.2.1');
```

### Skip the postinstall download (CI / air-gapped)

```bash
M9M_DOWNLOAD_BINARY=0 npm install m9m-cli
```

Then pre-stage the `m9m` binary at `~/.m9m/bin/m9m`, or download manually:

```bash
# macOS Apple Silicon
curl -L -o ~/.m9m/bin/m9m \
  https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-arm64
chmod +x ~/.m9m/bin/m9m
```

| Variable | Meaning |
|---|---|
| `M9M_VERSION` | Pin a specific binary release tag |
| `M9M_DOWNLOAD_BINARY` | Set to `0` to skip auto-download; `1` to force in CI |
| `CI` | Detected automatically; skips download unless `M9M_DOWNLOAD_BINARY=1` |

---

## FAQ

### Does this package bundle a Node.js runtime?
No — m9m is a Go binary. This package is a thin Node.js client that calls it. There is no embedded Node.js runtime, no V8, no event loop on the server side.

### Where does the binary come from?
GitHub Releases: `https://github.com/neul-labs/m9m/releases`. Each release ships signed SHA-256 checksums; the postinstall script verifies them.

### Can I use this offline / air-gapped?
Yes. Set `M9M_DOWNLOAD_BINARY=0` to skip the postinstall download, then stage the binary at `~/.m9m/bin/m9m` from your internal mirror. See [Pinning and offline use](#pinning-and-offline-use).

### How do I pin a specific binary version?
`M9M_VERSION=v0.2.1 npm install m9m-cli`, or call `downloadBinary('v0.2.1')` at runtime.

### Does it run n8n workflows unchanged?
For 40+ built-in node types: yes. Community n8n nodes (`n8n-nodes-*`) and n8n Cloud–specific features are not supported. See the [migration guide](https://github.com/neul-labs/m9m/blob/main/docs/migration/from-n8n.md).

---

## Links

- **Documentation:** [docs.neullabs.com/m9m](https://docs.neullabs.com/m9m)
- **Repository:** [github.com/neul-labs/m9m](https://github.com/neul-labs/m9m)
- **Changelog:** [Releases](https://github.com/neul-labs/m9m/releases)
- **Issues:** [GitHub Issues](https://github.com/neul-labs/m9m/issues)
- **Python SDK:** [`m9m-cli` on PyPI](https://pypi.org/project/m9m-cli/)

## License

MIT — see [LICENSE](https://github.com/neul-labs/m9m/blob/main/LICENSE).
