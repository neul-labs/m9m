/**
 * Workflow class for representing workflow definitions.
 */

import * as fs from 'fs';
import type { WorkflowData, WorkflowNode, NodeConnections } from './types';

/**
 * Represents a workflow definition.
 */
export class Workflow {
  private data: WorkflowData;

  private constructor(data: WorkflowData) {
    this.data = data;
  }

  /**
   * Load a workflow from a JSON file.
   */
  static fromFile(path: string): Workflow {
    const content = fs.readFileSync(path, 'utf8');
    return Workflow.fromJSON(content);
  }

  /**
   * Parse a workflow from a JSON string or object.
   */
  static fromJSON(json: string | object): Workflow {
    const data = typeof json === 'string' ? JSON.parse(json) : json;
    return new Workflow(data as WorkflowData);
  }

  /**
   * Create a workflow from workflow data.
   */
  static create(data: WorkflowData): Workflow {
    return Workflow.fromJSON(data);
  }

  get id(): string | null {
    return this.data.id || null;
  }

  get name(): string | null {
    return this.data.name || null;
  }

  toJSON(): WorkflowData {
    return this.data;
  }

  toString(): string {
    return JSON.stringify(this.data);
  }

  get nodes(): WorkflowNode[] {
    return this.data.nodes || [];
  }

  get connections(): Record<string, NodeConnections> {
    return this.data.connections || {};
  }

  get active(): boolean {
    return this.data.active ?? true;
  }
}

/**
 * Create a new workflow with the given configuration.
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
