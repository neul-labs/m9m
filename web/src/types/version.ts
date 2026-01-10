import type { Workflow } from './workflow'

export interface WorkflowVersion {
  id: string
  workflowId: string
  versionTag: string
  versionNum: number
  workflow: Workflow
  author: string
  description: string
  changes: string[]
  isCurrent: boolean
  createdAt: string
  tags?: string[]
}

export interface VersionComparison {
  fromVersion: WorkflowVersion
  toVersion: WorkflowVersion
  changes: VersionChanges
}

export interface VersionChanges {
  nodesAdded: string[]
  nodesRemoved: string[]
  nodesModified: string[]
  connectionsChanged: boolean
  settingsChanged: boolean
  summary: string
}

export interface VersionCreateRequest {
  versionTag?: string
  description?: string
  tags?: string[]
}

export interface VersionRestoreRequest {
  createBackup?: boolean
  description?: string
}

export interface VersionListFilters {
  workflowId?: string
  author?: string
  tags?: string[]
  limit?: number
  offset?: number
}

export interface VersionListResponse {
  data: WorkflowVersion[]
  total: number
  count: number
  limit: number
  offset: number
}
