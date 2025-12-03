export interface Workflow {
  id: string
  name: string
  description?: string
  active: boolean
  nodes: WorkflowNode[]
  connections: Record<string, NodeConnections>
  settings?: WorkflowSettings
  staticData?: Record<string, unknown>
  pinData?: Record<string, DataItem[]>
  tags?: string[]
  versionId?: string
  isArchived?: boolean
  createdAt: string
  updatedAt: string
  createdBy?: string
}

export interface WorkflowNode {
  id: string
  name: string
  type: string
  typeVersion: number
  position: [number, number]
  parameters: Record<string, unknown>
  credentials?: Record<string, NodeCredential>
  webhookId?: string
  notes?: string
  disabled?: boolean
}

export interface NodeCredential {
  id?: string
  name: string
  type: string
  data?: Record<string, unknown>
}

export interface WorkflowSettings {
  executionOrder?: string
  timezone?: string
  saveDataError?: boolean | string
  saveDataSuccess?: boolean | string
  saveManualExecutions?: boolean | string
}

export interface NodeConnections {
  main?: Connection[][]
}

export interface Connection {
  node: string
  type: string
  index: number
}

export interface DataItem {
  json: Record<string, unknown>
  binary?: Record<string, BinaryData>
  pairedItem?: unknown
  error?: unknown
}

export interface BinaryData {
  data: string
  mimeType: string
  fileSize?: string
  fileName?: string
  directory?: string
  fileExtension?: string
}

export interface WorkflowExecution {
  id: string
  workflowId: string
  status: ExecutionStatus
  mode: ExecutionMode
  startedAt: string
  finishedAt?: string
  data?: DataItem[]
  error?: string
  nodeData?: Record<string, DataItem[]>
  metadata?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export type ExecutionStatus = 'running' | 'completed' | 'failed' | 'cancelled' | 'waiting'
export type ExecutionMode = 'manual' | 'trigger' | 'test' | 'retry'

export interface WorkflowFilters {
  search?: string
  active?: boolean
  tags?: string[]
  offset?: number
  limit?: number
}

export interface ExecutionFilters {
  workflowId?: string
  status?: ExecutionStatus
  offset?: number
  limit?: number
}
