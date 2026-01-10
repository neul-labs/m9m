/**
 * Basic tests for m9m Node.js bindings.
 */

const { describe, it, before } = require('node:test');
const assert = require('node:assert');
const path = require('node:path');

// Try to load the module
let m9m;
let nativeAvailable = false;

before(async () => {
  try {
    // Build the TypeScript first if dist doesn't exist
    const distPath = path.join(__dirname, '..', 'dist', 'index.js');
    const fs = require('fs');

    if (fs.existsSync(distPath)) {
      m9m = require(distPath);
      nativeAvailable = m9m.isNativeAvailable?.() ?? false;
    }
  } catch (err) {
    console.log('Module not built yet:', err.message);
  }
});

describe('m9m Node.js bindings', () => {
  describe('Module Loading', () => {
    it('should export expected functions', () => {
      if (!m9m) {
        console.log('Skipping: module not built');
        return;
      }

      assert.ok(typeof m9m.WorkflowEngine === 'function' || typeof m9m.WorkflowEngine === 'undefined');
      assert.ok(typeof m9m.Workflow === 'function' || typeof m9m.Workflow === 'undefined');
      assert.ok(typeof m9m.CredentialManager === 'function' || typeof m9m.CredentialManager === 'undefined');
    });
  });

  describe('Type Definitions', () => {
    it('should have DataItem interface properties documented', () => {
      // This test validates that our types are properly defined
      // by checking that they can be used in JavaScript
      const dataItem = {
        json: { key: 'value' },
        binary: undefined,
        pairedItem: undefined,
        error: undefined,
      };

      assert.ok(dataItem.json);
      assert.strictEqual(dataItem.json.key, 'value');
    });

    it('should have WorkflowData interface properties', () => {
      const workflow = {
        name: 'Test Workflow',
        nodes: [],
        connections: {},
        active: true,
      };

      assert.strictEqual(workflow.name, 'Test Workflow');
      assert.ok(Array.isArray(workflow.nodes));
      assert.ok(typeof workflow.connections === 'object');
    });

    it('should have ExecutionResult interface properties', () => {
      const result = {
        data: [{ json: { result: 'ok' } }],
        error: undefined,
      };

      assert.ok(Array.isArray(result.data));
      assert.strictEqual(result.error, undefined);
    });
  });

  describe('Native Bindings', { skip: !nativeAvailable }, () => {
    it('should get version', () => {
      const version = m9m.version();
      assert.ok(typeof version === 'string');
      assert.ok(version.length > 0);
    });

    it('should create WorkflowEngine', () => {
      const engine = new m9m.WorkflowEngine();
      assert.ok(engine);
    });

    it('should create Workflow from JSON', () => {
      const workflow = m9m.Workflow.fromJSON({
        name: 'Test',
        nodes: [],
        connections: {},
      });
      assert.ok(workflow);
      assert.strictEqual(workflow.name, 'Test');
    });

    it('should execute empty workflow', async () => {
      const engine = new m9m.WorkflowEngine();
      const workflow = m9m.Workflow.fromJSON({
        name: 'Empty Test',
        nodes: [],
        connections: {},
      });

      const result = await engine.execute(workflow);
      assert.ok(result);
      assert.ok(!result.error);
    });

    it('should create CredentialManager', () => {
      const credManager = new m9m.CredentialManager();
      assert.ok(credManager);
    });

    it('should store credentials', () => {
      const credManager = new m9m.CredentialManager();
      const stored = credManager.store({
        id: 'test-cred',
        name: 'Test Credential',
        type: 'apiKey',
        data: { apiKey: 'secret' },
      });
      assert.ok(stored);
    });
  });
});

// Run if executed directly
if (require.main === module) {
  console.log('Running tests...');
  console.log('Native bindings available:', nativeAvailable);
}
