# @m9m/workflow-engine

Node.js bindings for the m9m high-performance workflow automation engine.

## Installation

```bash
npm install @m9m/workflow-engine
```

### Prerequisites

Before using, ensure the m9m shared library is built:

```bash
cd cgo
make shared
```

Then build the native addon:

```bash
cd bindings/nodejs
npm install
npm run build
```

## Quick Start

```typescript
import { WorkflowEngine, Workflow, createWorkflow } from '@m9m/workflow-engine';

// Create an engine
const engine = new WorkflowEngine();

// Load and execute a workflow
const workflow = Workflow.fromFile('workflow.json');
const result = await engine.execute(workflow);

console.log('Success:', !result.error);
for (const item of result.data) {
  console.log(item.json);
}
```

## Custom Nodes

Register custom nodes to extend workflow functionality:

```typescript
// Register with a function
engine.registerNode('custom.uppercase', (inputData, params) => {
  return inputData.map(item => ({
    json: { text: String(item.json.text).toUpperCase() }
  }));
});

// Or use the decorator-style API
const processData = engine.node('custom.process')((input, params) => {
  return input.map(item => ({
    json: { ...item.json, processed: true }
  }));
});
```

## Creating Workflows

```typescript
import { createWorkflow } from '@m9m/workflow-engine';

const workflow = createWorkflow({
  name: 'My Workflow',
  nodes: [
    {
      name: 'Start',
      type: 'n8n-nodes-base.start',
      parameters: {}
    },
    {
      name: 'Process',
      type: 'custom.process',
      parameters: { option: 'value' }
    }
  ],
  connections: {
    'Start': {
      main: [[{ node: 'Process', type: 'main', index: 0 }]]
    }
  }
});
```

## Credentials

Use the credential manager for secure credential storage:

```typescript
import { WorkflowEngine, CredentialManager } from '@m9m/workflow-engine';

const credManager = new CredentialManager();
credManager.store({
  id: 'api-key-1',
  name: 'My API Key',
  type: 'apiKey',
  data: { apiKey: 'secret123' }
});

const engine = new WorkflowEngine({ credentialManager: credManager });
```

## API Reference

### WorkflowEngine

- `execute(workflow, inputData?)` - Execute a workflow
- `loadWorkflow(path)` - Load workflow from file
- `parseWorkflow(json)` - Parse workflow from JSON
- `registerNode(nodeType, executor)` - Register a custom node
- `node(nodeType)` - Decorator to register nodes

### Workflow

- `Workflow.fromFile(path)` - Load from JSON file
- `Workflow.fromJSON(json)` - Parse from JSON string/object
- `toJSON()` - Serialize to JSON object
- `id`, `name`, `active`, `nodes` - Properties

### ExecutionResult

- `data` - Array of output DataItems
- `error` - Error message if failed

### CredentialManager

- `store(credential)` - Store a credential
- `storeMany(credentials)` - Store multiple credentials

## TypeScript Support

Full TypeScript definitions are included:

```typescript
import type {
  DataItem,
  ExecutionResult,
  WorkflowData,
  WorkflowNode,
  CredentialData,
  NodeExecutorFn
} from '@m9m/workflow-engine';
```

## License

MIT License
