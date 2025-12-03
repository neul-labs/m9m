export interface ApiResponse<T> {
  data: T
  total?: number
  offset?: number
  limit?: number
}

export interface ApiError {
  error: boolean
  message: string
  code: number
  details?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  offset: number
  limit: number
}

export interface HealthResponse {
  status: string
  service: string
  version: string
  time: string
}

export interface VersionResponse {
  n8nVersion: string
  serverVersion: string
  implementation: string
  compatibility: {
    workflows: boolean
    nodes: boolean
    expressions: boolean
    credentials: boolean
  }
}

export interface SettingsResponse {
  timezone: string
  executionMode: string
  saveDataSuccessExecution: string
  saveDataErrorExecution: string
  saveExecutionProgress: boolean
  saveManualExecutions: boolean
  communityNodesEnabled: boolean
  versionNotifications: {
    enabled: boolean
  }
  instanceId: string
  telemetry: {
    enabled: boolean
  }
}

export interface Credential {
  id: string
  name: string
  type: string
  createdAt: string
  updatedAt: string
}

export interface CredentialCreate {
  name: string
  type: string
  data: Record<string, unknown>
}

export interface Tag {
  id: string
  name: string
  createdAt: string
  updatedAt: string
}

export interface WebSocketMessage {
  type: WebSocketMessageType
  data: Record<string, unknown>
  timestamp?: number
}

export type WebSocketMessageType =
  | 'connected'
  | 'executionUpdate'
  | 'nodeExecution'
  | 'workflowActivated'
  | 'workflowDeactivated'
  | 'response'
  | 'error'

export interface ExecutionUpdateMessage {
  executionId: string
  workflowId: string
  status: string
  startedAt?: string
  finishedAt?: string
  error?: string
}

export interface NodeExecutionMessage {
  executionId: string
  nodeId: string
  status: string
  data?: Record<string, unknown>
}
