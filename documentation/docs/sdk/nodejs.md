# Node.js Bindings

Node.js bindings use N-API for native addon support with full TypeScript types.

## Prerequisites

- Node.js 16+
- node-gyp and build tools
- CGO shared library

## Installation

```bash
# Build the shared library first
cd /path/to/m9m
make cgo-lib

# Install and build Node.js bindings
cd bindings/nodejs
npm install
npm run build
```

## Quick Start

```typescript
import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';

const engine = new WorkflowEngine();

const workflow = Workflow.fromJSON({
  name: 'My Workflow',
  nodes: [],
  connections: {}
});

const result = await engine.execute(workflow);
console.log(result.data);
```

## API Reference

### WorkflowEngine

#### Creating an Engine

```typescript
import { WorkflowEngine, CredentialManager } from '@m9m/workflow-engine';

// Basic engine
const engine = new WorkflowEngine();

// Engine with credential manager
const credManager = new CredentialManager();
const engine = new WorkflowEngine(credManager);
```

#### Executing Workflows

```typescript
// Basic execution
const result = await engine.execute(workflow);

// With input data
const result = await engine.execute(workflow, [
  { json: { key: 'value' } },
  { json: { key: 'value2' } }
]);

// Check results
if (!result.error) {
  for (const item of result.data) {
    console.log(item.json);
  }
}
```

#### Registering Custom Nodes

```typescript
engine.registerNode('custom.myNode', {
  name: 'My Node',
  category: 'transform'
}, async (input, params) => {
  return [{ json: { processed: true } }];
});
```

### Workflow

#### Loading Workflows

```typescript
import { Workflow } from '@m9m/workflow-engine';

// From JSON object
const workflow = Workflow.fromJSON({
  name: 'Test',
  nodes: [],
  connections: {}
});

// From file
const workflow = Workflow.fromFile('workflow.json');

// From JSON string
const workflow = Workflow.fromJSON(JSON.parse(jsonString));
```

#### Workflow Properties

```typescript
workflow.name   // Workflow name
workflow.id     // Unique identifier

// Convert to JSON
const json = workflow.toJSON();
```

### CredentialManager

```typescript
import { CredentialManager } from '@m9m/workflow-engine';

const credManager = new CredentialManager();

// Store credentials
credManager.store({
  id: 'my-api-key',
  name: 'API Key',
  type: 'apiKey',
  data: { apiKey: 'secret-key' }
});
```

### Type Definitions

```typescript
interface DataItem {
  json: Record<string, unknown>;
  binary?: Record<string, BinaryData>;
  pairedItem?: PairedItemInfo;
  error?: ExecutionError;
}

interface ExecutionResult {
  data: DataItem[];
  error?: string;
}

interface WorkflowData {
  name: string;
  nodes: NodeData[];
  connections: Record<string, NodeConnections>;
  active?: boolean;
  settings?: WorkflowSettings;
}

interface NodeData {
  id: string;
  name: string;
  type: string;
  position?: [number, number];
  parameters?: Record<string, unknown>;
  credentials?: Record<string, CredentialReference>;
}
```

## Custom Nodes

### Async Node Handler

```typescript
engine.registerNode('custom.fetch', {
  name: 'Fetch Data',
  category: 'http'
}, async (input, params) => {
  const url = params.url as string;
  const response = await fetch(url);
  const data = await response.json();

  return [{ json: data }];
});
```

### Processing Input Items

```typescript
engine.registerNode('custom.transform', {
  name: 'Transform',
  category: 'transform'
}, async (input, params) => {
  const multiplier = (params.multiplier as number) || 1;

  return input.map(item => ({
    json: {
      original: item.json,
      value: (item.json.value as number) * multiplier
    }
  }));
});
```

## Error Handling

```typescript
try {
  const workflow = Workflow.fromFile('missing.json');
} catch (error) {
  console.error('Failed to load workflow:', error);
}

// Check execution errors
const result = await engine.execute(workflow);
if (result.error) {
  console.error('Execution failed:', result.error);
}
```

## Environment Setup

Set the library path if the native addon can't find the shared library:

```bash
# Linux
export LD_LIBRARY_PATH=/path/to/m9m/cgo:$LD_LIBRARY_PATH

# macOS
export DYLD_LIBRARY_PATH=/path/to/m9m/cgo:$DYLD_LIBRARY_PATH
```

## Testing

```bash
cd bindings/nodejs
npm test
```

## CommonJS vs ESM

The package supports both CommonJS and ES modules:

```javascript
// CommonJS
const { WorkflowEngine, Workflow } = require('@m9m/workflow-engine');

// ES Modules
import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';
```
