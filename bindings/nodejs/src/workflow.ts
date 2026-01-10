/**
 * Workflow class for representing workflow definitions.
 */

import { getNativeBinding, NativeWorkflow } from './native';
import type { WorkflowData, WorkflowNode, NodeConnections } from './types';

/**
 * Represents a workflow definition.
 *
 * @example
 * ```typescript
 * // Load from file
 * const workflow = Workflow.fromFile('workflow.json');
 *
 * // Create from JSON
 * const workflow = Workflow.fromJSON({
 *   name: 'My Workflow',
 *   nodes: [],
 *   connections: {}
 * });
 *
 * console.log(workflow.name);
 * console.log(workflow.toJSON());
 * ```
 */
export class Workflow {
  private native: NativeWorkflow;

  /**
   * Private constructor. Use static methods to create instances.
   */
  private constructor(native: NativeWorkflow) {
    this.native = native;
  }

  /**
   * Load a workflow from a JSON file.
   *
   * @param path - Path to the workflow JSON file.
   * @returns The loaded Workflow object.
   * @throws Error if the file cannot be read or parsed.
   */
  static fromFile(path: string): Workflow {
    const binding = getNativeBinding();
    const native = binding.Workflow.fromFile(path);
    return new Workflow(native);
  }

  /**
   * Parse a workflow from a JSON string or object.
   *
   * @param json - JSON string or object representing the workflow.
   * @returns The parsed Workflow object.
   * @throws Error if the JSON is invalid.
   */
  static fromJSON(json: string | object): Workflow {
    const binding = getNativeBinding();
    const native = binding.Workflow.fromJSON(json);
    return new Workflow(native);
  }

  /**
   * Create a workflow from workflow data.
   *
   * @param data - Workflow data object.
   * @returns The created Workflow object.
   */
  static create(data: WorkflowData): Workflow {
    return Workflow.fromJSON(data);
  }

  /**
   * Get the workflow ID.
   */
  get id(): string | null {
    return this.native.id;
  }

  /**
   * Get the workflow name.
   */
  get name(): string | null {
    return this.native.name;
  }

  /**
   * Convert the workflow to a JSON object.
   */
  toJSON(): WorkflowData {
    return this.native.toJSON() as WorkflowData;
  }

  /**
   * Convert the workflow to a JSON string.
   */
  toString(): string {
    return JSON.stringify(this.toJSON());
  }

  /**
   * Get the workflow nodes.
   */
  get nodes(): WorkflowNode[] {
    return this.toJSON().nodes || [];
  }

  /**
   * Get the workflow connections.
   */
  get connections(): Record<string, NodeConnections> {
    return this.toJSON().connections || {};
  }

  /**
   * Check if the workflow is active.
   */
  get active(): boolean {
    return this.toJSON().active || false;
  }
}

/**
 * Create a new workflow with the given configuration.
 *
 * @param options - Workflow configuration options.
 * @returns A new Workflow instance.
 *
 * @example
 * ```typescript
 * const workflow = createWorkflow({
 *   name: 'My Workflow',
 *   nodes: [
 *     {
 *       name: 'Start',
 *       type: 'n8n-nodes-base.start',
 *       parameters: {}
 *     }
 *   ]
 * });
 * ```
 */
export function createWorkflow(options: {
  name: string;
  nodes?: WorkflowNode[];
  connections?: Record<string, NodeConnections>;
  active?: boolean;
  id?: string;
  description?: string;
}): Workflow {
  const data: WorkflowData = {
    name: options.name,
    nodes: options.nodes || [],
    connections: options.connections || {},
    active: options.active ?? true,
    id: options.id,
    description: options.description,
  };
  return Workflow.fromJSON(data);
}
