/**
 * WorkflowEngine class for executing workflows.
 */

import { getNativeBinding, NativeEngine } from './native';
import { Workflow } from './workflow';
import { CredentialManager } from './credentials';
import type { DataItem, ExecutionResult, NodeExecutorFn } from './types';

/**
 * High-level interface to the m9m workflow execution engine.
 *
 * @example
 * ```typescript
 * const engine = new WorkflowEngine();
 *
 * // Register a custom node
 * engine.registerNode('custom.myNode', (input, params) => {
 *   return input.map(item => ({
 *     json: { ...item.json, processed: true }
 *   }));
 * });
 *
 * // Load and execute a workflow
 * const workflow = Workflow.fromFile('workflow.json');
 * const result = await engine.execute(workflow);
 * ```
 */
export class WorkflowEngine {
  private native: NativeEngine;
  private customNodes: Map<string, NodeExecutorFn> = new Map();

  /**
   * Create a new workflow engine.
   *
   * @param options - Optional configuration options.
   * @param options.credentialManager - Optional credential manager for secure credential storage.
   */
  constructor(options?: { credentialManager?: CredentialManager }) {
    const binding = getNativeBinding();
    this.native = new binding.Engine(options?.credentialManager?.['native']);
  }

  /**
   * Execute a workflow with optional input data.
   *
   * @param workflow - The workflow to execute.
   * @param inputData - Optional array of input data items.
   * @returns Promise that resolves to the execution result.
   *
   * @example
   * ```typescript
   * const result = await engine.execute(workflow, [
   *   { json: { message: 'Hello' } }
   * ]);
   * ```
   */
  async execute(
    workflow: Workflow,
    inputData?: DataItem[]
  ): Promise<ExecutionResult> {
    const result = this.native.execute(workflow['native'], inputData);
    return this.parseResult(result);
  }

  /**
   * Load a workflow from a JSON file.
   *
   * @param path - Path to the workflow JSON file.
   * @returns The loaded Workflow object.
   */
  loadWorkflow(path: string): Workflow {
    return Workflow.fromFile(path);
  }

  /**
   * Parse a workflow from a JSON string or object.
   *
   * @param json - JSON string or object representing the workflow.
   * @returns The parsed Workflow object.
   */
  parseWorkflow(json: string | object): Workflow {
    return Workflow.fromJSON(json);
  }

  /**
   * Register a custom node type.
   *
   * @param nodeType - The node type identifier (e.g., "custom.myNode").
   * @param executor - The function to execute when the node runs.
   *
   * @example
   * ```typescript
   * engine.registerNode('custom.uppercase', (input, params) => {
   *   return input.map(item => ({
   *     json: { text: String(item.json.text).toUpperCase() }
   *   }));
   * });
   * ```
   */
  registerNode(nodeType: string, executor: NodeExecutorFn): void {
    this.customNodes.set(nodeType, executor);
    this.native.registerNode(nodeType, executor);
  }

  /**
   * Decorator-style method to register a node.
   *
   * @param nodeType - The node type identifier.
   * @returns A decorator function.
   *
   * @example
   * ```typescript
   * const uppercase = engine.node('custom.uppercase')((input, params) => {
   *   return input.map(item => ({
   *     json: { text: String(item.json.text).toUpperCase() }
   *   }));
   * });
   * ```
   */
  node(nodeType: string): (executor: NodeExecutorFn) => NodeExecutorFn {
    return (executor: NodeExecutorFn): NodeExecutorFn => {
      this.registerNode(nodeType, executor);
      return executor;
    };
  }

  /**
   * Parse the raw result from the native binding.
   */
  private parseResult(result: unknown): ExecutionResult {
    if (!result || typeof result !== 'object') {
      return { data: [], error: undefined };
    }

    const r = result as Record<string, unknown>;
    const data = Array.isArray(r.data)
      ? (r.data as DataItem[])
      : [];
    const error = typeof r.error === 'string' ? r.error : undefined;

    return { data, error };
  }
}
