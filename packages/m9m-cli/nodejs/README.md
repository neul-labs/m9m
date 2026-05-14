# m9m-cli

> **High-performance workflow automation CLI and SDK for Node.js** — 5–10x faster than n8n, 70% lower memory, sub-second startup. Execute n8n-compatible workflows programmatically from your Node.js or TypeScript applications.

[![npm version](https://img.shields.io/npm/v/m9m-cli.svg)](https://www.npmjs.com/package/m9m-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/neul-labs/m9m/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/neul-labs/m9m)](https://github.com/neul-labs/m9m/releases)

## What is m9m?

**m9m** is a cloud-native workflow automation engine built in Go that delivers **95% feature parity with n8n** while running **5–10x faster** and using **~70% less memory** (~150 MB vs 512 MB). It supports the full n8n workflow JSON format, expression syntax, and node ecosystem — making it a drop-in replacement for teams that need enterprise-grade performance without the overhead.

The `m9m-cli` npm package embeds the engine directly into your Node.js application. On installation, it automatically downloads the correct platform-native binary (macOS, Linux, or Windows; AMD64 and ARM64) from GitHub Releases and caches it locally. No Docker, no Kubernetes, no separate server process required.

## Key Features

- **n8n-Compatible Workflows** — Load and execute workflows exported from n8n without modification
- **TypeScript-First API** — Full type definitions for workflows, nodes, data items, and execution results
- **Zero Configuration** — Binary auto-downloads on `npm install` for your OS/architecture
- **Lightweight** — ~15 MB binary, ~150 MB runtime memory footprint
- **Cross-Platform** — Native support for macOS (Intel & Apple Silicon), Linux (AMD64 & ARM64), and Windows
- **Custom Node Registration** — Extend the engine with JavaScript/TypeScript node implementations
- **Credential Manager** — Securely manage API keys and credentials within your application

## Installation

```bash
npm install m9m-cli
```

Or with your favorite package manager:

```bash
yarn add m9m-cli
pnpm add m9m-cli
bun add m9m-cli
```

### Manual Binary Download

If the automatic download fails (e.g., behind a corporate proxy), you can download the binary manually:

```bash
# macOS Apple Silicon
curl -L -o ~/.m9m/bin/m9m https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-arm64
chmod +x ~/.m9m/bin/m9m

# Linux AMD64
curl -L -o ~/.m9m/bin/m9m https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64
chmod +x ~/.m9m/bin/m9m
```

## Quick Start

### Execute a Workflow

```typescript
import { WorkflowEngine, Workflow } from 'm9m-cli';

const engine = new WorkflowEngine();

// Load a workflow from an n8n-compatible JSON file
const workflow = Workflow.fromFile('./my-workflow.json');

// Execute with input data
const result = await engine.execute(workflow, [
  { json: { email: 'user@example.com', name: 'Alice' } }
]);

console.log('Output:', result.data);
if (result.error) {
  console.error('Execution failed:', result.error);
}
```

### Create a Workflow Programmatically

```typescript
import { createWorkflow, WorkflowEngine } from 'm9m-cli';

const workflow = createWorkflow({
  name: 'Email Notification',
  nodes: [
    {
      name: 'Start',
      type: 'n8n-nodes-base.start',
      parameters: {}
    },
    {
      name: 'Send Email',
      type: 'n8n-nodes-base.emailSend',
      parameters: {
        to: '={{ $json.email }}',
        subject: 'Welcome!',
        text: 'Hello {{ $json.name }}'
      }
    }
  ],
  connections: {
    'Start': {
      main: [[{ node: 'Send Email', type: 'main', index: 0 }]]
    }
  }
});

const engine = new WorkflowEngine();
const result = await engine.execute(workflow);
```

### Register Custom Nodes

```typescript
import { WorkflowEngine } from 'm9m-cli';

const engine = new WorkflowEngine();

// Register a custom transformation node
engine.registerNode('custom.uppercase', (inputData, params) => {
  return inputData.map(item => ({
    json: {
      ...item.json,
      text: String(item.json.text).toUpperCase()
    }
  }));
});

// Or use the decorator-style API
const processData = engine.node('custom.process')((input, params) => {
  return input.map(item => ({
    json: { ...item.json, processed: true }
  }));
});
```

## API Reference

### `WorkflowEngine`

The main entry point for executing workflows.

```typescript
const engine = new WorkflowEngine();

// Execute a workflow
const result = await engine.execute(workflow, inputData);

// Load from file
const workflow = engine.loadWorkflow('./workflow.json');

// Parse from JSON string or object
const workflow = engine.parseWorkflow('{ "name": "test", ... }');

// Register custom nodes
engine.registerNode('custom.myNode', executor);
engine.node('custom.myNode')(executor);
```

### `Workflow`

Represents a workflow definition.

```typescript
// From file
const workflow = Workflow.fromFile('workflow.json');

// From JSON string
const workflow = Workflow.fromJSON('{ "name": "test" ... }');

// From JSON object
const workflow = Workflow.fromJSON({
  name: 'My Workflow',
  nodes: [],
  connections: {}
});

// Access properties
console.log(workflow.name);
console.log(workflow.id);

// Serialize
const json = workflow.toJSON();
```

### `DataItem`

The standard data structure passed between nodes.

```typescript
interface DataItem {
  json: Record<string, unknown>;
  binary?: Record<string, BinaryData>;
  pairedItem?: PairedItemInfo;
  error?: ExecutionError;
}
```

### `ExecutionResult`

Returned by `engine.execute()`.

```typescript
interface ExecutionResult {
  data: DataItem[];
  error?: string;
}
```

### `CredentialManager`

Securely store and retrieve credentials.

```typescript
import { WorkflowEngine, CredentialManager } from 'm9m-cli';

const credManager = new CredentialManager();
credManager.store({
  id: 'api-key-1',
  name: 'My API Key',
  type: 'apiKey',
  data: { apiKey: 'secret123' }
});

const engine = new WorkflowEngine({ credentialManager: credManager });
```

### Utility Functions

```typescript
import { version, downloadBinary, getBinaryPath } from 'm9m-cli';

// Get the m9m binary version
console.log(version()); // "0.2.1"

// Manually download the binary for a specific version
const path = await downloadBinary('v0.2.1');

// Get the path to the cached binary
const binaryPath = getBinaryPath();
```

## Supported Platforms

| OS | Architecture | Status |
|---|---|---|
| macOS | Intel (AMD64) | ✅ Supported |
| macOS | Apple Silicon (ARM64) | ✅ Supported |
| Linux | AMD64 | ✅ Supported |
| Linux | ARM64 | ✅ Supported |
| Windows | AMD64 | ✅ Supported |

The correct binary is automatically selected and downloaded during installation based on your operating system and CPU architecture.

## Environment Variables

| Variable | Description |
|---|---|
| `M9M_DOWNLOAD_BINARY` | Set to `1` to force binary download during `npm install` in CI environments |
| `CI` | When set, binary download is skipped unless `M9M_DOWNLOAD_BINARY=1` is also set |

## Supported Node Types

m9m supports 40+ built-in node types including:

- **Core**: `start`, `noOp`, `wait`, `executeWorkflow`
- **Transform**: `set`, `filter`, `code`, `function`, `merge`, `json`, `if`, `loop`
- **HTTP**: `httpRequest`
- **Trigger**: `webhook`, `errorTrigger`
- **Timer**: `cron`
- **Messaging**: `slack`, `discord`, `twilio`, `teams`
- **Database**: `postgres`, `mysql`, `sqlite`, `elasticsearch`, `redis`, `mongodb`
- **Email**: `emailSend`, `sendGrid`
- **AI**: `openAi`, `anthropic`
- **VCS**: `github`, `gitlab`
- **Cloud**: `aws`, `gcp`, `azure`
- **Productivity**: `notion`, `stripe`, `googleSheets`

## Performance

| Metric | m9m | n8n | Improvement |
|---|---|---|---|
| Startup Time | < 500 ms | ~3 s | **6x faster** |
| Memory Usage | ~150 MB | ~512 MB | **70% lower** |
| Execution Speed | Baseline | 5–10x slower | **5–10x faster** |
| Container Size | ~300 MB | ~1.2 GB | **75% smaller** |

## Why m9m?

- **Go-Powered Performance** — Built in Go for maximum concurrency and minimal resource usage
- **n8n Compatibility** — Import existing n8n workflows without changes
- **Cloud-Native** — Stateless architecture ready for Kubernetes, Docker, and serverless
- **Enterprise Observability** — Built-in Prometheus metrics and OpenTelemetry tracing
- **No Node.js Runtime Required** — The engine is a single static binary; no npm install on the server

## Documentation

- [Full Documentation](https://github.com/neul-labs/m9m/tree/main/docs)
- [API Reference](https://github.com/neul-labs/m9m/tree/main/docs/sdk)
- [Workflow Examples](https://github.com/neul-labs/m9m/tree/main/examples)
- [n8n Compatibility Guide](https://github.com/neul-labs/m9m/tree/main/docs/api/API_COMPATIBILITY.md)

## License

MIT License — see [LICENSE](https://github.com/neul-labs/m9m/blob/main/LICENSE) for details.

## Contributing

Contributions are welcome! Please see our [Contributing Guide](https://github.com/neul-labs/m9m/blob/main/docs/CONTRIBUTING.md) and [Code of Conduct](https://github.com/neul-labs/m9m/blob/main/CODE_OF_CONDUCT.md).

## Community

- [GitHub Discussions](https://github.com/neul-labs/m9m/discussions)
- [GitHub Issues](https://github.com/neul-labs/m9m/issues)
- [Release Notes](https://github.com/neul-labs/m9m/releases)

---

**Built with ❤️ by [Neul Labs](https://github.com/neul-labs)**
