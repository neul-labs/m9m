import apiClient from './client'
import type {
  WorkflowVersion,
  VersionComparison,
  VersionCreateRequest,
  VersionRestoreRequest,
  VersionListResponse,
} from '@/types/version'

export async function listVersions(
  workflowId: string,
  limit = 50,
  offset = 0
): Promise<VersionListResponse> {
  const response = await apiClient.get<VersionListResponse>(
    `/workflows/${workflowId}/versions`,
    { params: { limit, offset } }
  )
  return response.data
}

export async function getVersion(
  workflowId: string,
  versionId: string
): Promise<WorkflowVersion> {
  const response = await apiClient.get<WorkflowVersion>(
    `/workflows/${workflowId}/versions/${versionId}`
  )
  return response.data
}

export async function createVersion(
  workflowId: string,
  data: VersionCreateRequest
): Promise<WorkflowVersion> {
  const response = await apiClient.post<WorkflowVersion>(
    `/workflows/${workflowId}/versions`,
    data
  )
  return response.data
}

export async function deleteVersion(
  workflowId: string,
  versionId: string
): Promise<void> {
  await apiClient.delete(`/workflows/${workflowId}/versions/${versionId}`)
}

export async function restoreVersion(
  workflowId: string,
  versionId: string,
  data?: VersionRestoreRequest
): Promise<{ message: string; restoredFrom: WorkflowVersion; backupCreated: boolean }> {
  const response = await apiClient.post(
    `/workflows/${workflowId}/versions/${versionId}/restore`,
    data || { createBackup: true }
  )
  return response.data
}

export async function compareVersions(
  workflowId: string,
  fromVersionId: string,
  toVersionId: string
): Promise<VersionComparison> {
  const response = await apiClient.get<VersionComparison>(
    `/workflows/${workflowId}/versions/compare`,
    { params: { from: fromVersionId, to: toVersionId } }
  )
  return response.data
}
