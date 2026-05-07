/**
 * WorkflowEngine class for executing workflows via the m9m binary.
 */

import { spawn } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { getBinaryPath, downloadBinary } from './binary';
import { Workflow } from './workflow';
import type { DataItem, ExecutionResult, NodeExecutorFn } from './types';

function runBinary(args: string[], stdin?: string): Promise<{ stdout: string; stderr: string; code: number }> {
  return new Promise((resolve, reject) => {
    const binary = getBinaryPath();
    const child = spawn(binary, args, { stdio: ['pipe', 'pipe', 'pipe'] });

    let stdout = '';
    let stderr = '';

    child.stdout.on('data', (data) => (stdout += data));
    child.stderr.on('data', (data) => (stderr += data));

    child.on('close', (code) => {
      resolve({ stdout, stderr, code: code ?? 1 });
    });

    child.on('error', reject);

    if (stdin) {
      child.stdin.write(stdin);
    }
    child.stdin.end();
  });
}

/**
 * High-level interface to the m9m workflow execution engine.
 *
 * @example
 * ```typescript
 * const engine = new WorkflowEngine();
 *
 * // Load and execute a workflow
 * const workflow = Workflow.fromFile('workflow.json');
 * const result = await engine.execute(workflow);
 * ```
 */
export class WorkflowEngine {
  private customNodes: Map<string, NodeExecutorFn> = new Map();

  /**
   * Create a new workflow engine.
   */
  constructor(_options?: { credentialManager?: unknown }) {
    // Binary is auto-downloaded on first use if needed
  }

  /**
   * Ensure the binary is available (downloads if needed).
   */
  async ensureBinary(): Promise<void> {
    try {
      getBinaryPath();
    } catch {
      await downloadBinary();
    }
  }

  /**
   * Execute a workflow with optional input data.
   */
  async execute(workflow: Workflow, inputData?: DataItem[]): Promise<ExecutionResult> {
    await this.ensureBinary();

    // Write workflow to temp file
    const tmpDir = os.tmpdir();
    const workflowPath = path.join(tmpDir, `m9m-workflow-${Date.now()}.json`);
    fs.writeFileSync(workflowPath, JSON.stringify(workflow.toJSON()));

    try {
      const inputJson = inputData ? JSON.stringify(inputData) : '[]';
      const { stdout, stderr, code } = await runBinary(
        ['exec', workflowPath, '--input', inputJson]
      );

      if (code !== 0) {
        return { data: [], error: stderr || 'execution failed' };
      }

      try {
        const parsed = JSON.parse(stdout);
        return { data: parsed.data || [], error: parsed.error };
      } catch {
        return { data: [{ json: { output: stdout } }], error: undefined };
      }
    } finally {
      try {
        fs.unlinkSync(workflowPath);
      } catch {
        // ignore
      }
    }
  }

  /**
   * Load a workflow from a JSON file.
   */
  loadWorkflow(path: string): Workflow {
    return Workflow.fromFile(path);
  }

  /**
   * Parse a workflow from a JSON string or object.
   */
  parseWorkflow(json: string | object): Workflow {
    return Workflow.fromJSON(json);
  }

  /**
   * Register a custom node type.
   */
  registerNode(nodeType: string, executor: NodeExecutorFn): void {
    this.customNodes.set(nodeType, executor);
  }

  /**
   * Decorator-style method to register a node.
   */
  node(nodeType: string): (executor: NodeExecutorFn) => NodeExecutorFn {
    return (executor: NodeExecutorFn): NodeExecutorFn => {
      this.registerNode(nodeType, executor);
      return executor;
    };
  }
}
