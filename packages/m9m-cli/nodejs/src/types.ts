/**
 * Types for the m9m workflow engine Node.js bindings.
 */

/**
 * Represents a data item flowing through the workflow.
 */
export interface DataItem {
  json: Record<string, unknown>;
  binary?: Record<string, BinaryData>;
  pairedItem?: unknown;
  error?: unknown;
}

/**
 * Represents binary data in a workflow.
 */
export interface BinaryData {
  data: string;
  mimeType: string;
  fileSize?: string;
  fileName?: string;
  directory?: string;
  fileExtension?: string;
}

/**
 * Result of a workflow execution.
 */
export interface ExecutionResult {
  data: DataItem[];
  error?: string;
}

/**
 * Represents a workflow node.
 */
export interface WorkflowNode {
  id?: string;
  name: string;
  type: string;
  typeVersion?: number;
  position?: [number, number];
  parameters?: Record<string, unknown>;
  credentials?: Record<string, CredentialReference>;
  webhookId?: string;
  notes?: string;
  disabled?: boolean;
}

/**
 * Reference to a credential used by a node.
 */
export interface CredentialReference {
  id?: string;
  name: string;
  type: string;
}

/**
 * Connection between nodes.
 */
export interface Connection {
  node: string;
  type: string;
  index: number;
}

/**
 * Node connections configuration.
 */
export interface NodeConnections {
  main?: Connection[][];
}

/**
 * Workflow settings.
 */
export interface WorkflowSettings {
  executionOrder?: string;
  timezone?: string;
  saveDataError?: boolean;
  saveDataSuccess?: boolean;
  saveManualExecutions?: boolean;
}

/**
 * Workflow definition.
 */
export interface WorkflowData {
  id?: string;
  name: string;
  description?: string;
  active?: boolean;
  nodes: WorkflowNode[];
  connections: Record<string, NodeConnections>;
  settings?: WorkflowSettings;
  staticData?: Record<string, unknown>;
  pinData?: Record<string, DataItem[]>;
  tags?: string[];
  versionId?: string;
  isArchived?: boolean;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: string;
}

/**
 * Credential data for storage.
 */
export interface CredentialData {
  id: string;
  name: string;
  type: string;
  data: Record<string, unknown>;
}

/**
 * Node description metadata.
 */
export interface NodeDescription {
  name: string;
  description: string;
  category: string;
}

/**
 * Custom node executor function type.
 */
export type NodeExecutorFn = (
  inputData: DataItem[],
  params: Record<string, unknown>
) => DataItem[] | Promise<DataItem[]>;
