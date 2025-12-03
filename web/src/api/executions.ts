import apiClient from './client'
import type { WorkflowExecution, ExecutionFilters } from '@/types'
import type { PaginatedResponse } from '@/types/api'

export async function getExecutions(filters?: ExecutionFilters): Promise<PaginatedResponse<WorkflowExecution>> {
  const params = new URLSearchParams()
  if (filters?.workflowId) params.append('workflowId', filters.workflowId)
  if (filters?.status) params.append('status', filters.status)
  if (filters?.offset !== undefined) params.append('offset', String(filters.offset))
  if (filters?.limit !== undefined) params.append('limit', String(filters.limit))

  const response = await apiClient.get<PaginatedResponse<WorkflowExecution>>('/executions', { params })
  return response.data
}

export async function getExecution(id: string): Promise<WorkflowExecution> {
  const response = await apiClient.get<WorkflowExecution>(`/executions/${id}`)
  return response.data
}

export async function deleteExecution(id: string): Promise<void> {
  await apiClient.delete(`/executions/${id}`)
}

export async function retryExecution(id: string): Promise<WorkflowExecution> {
  const response = await apiClient.post<WorkflowExecution>(`/executions/${id}/retry`)
  return response.data
}

export async function cancelExecution(id: string): Promise<{ message: string; status: string }> {
  const response = await apiClient.post<{ message: string; status: string }>(`/executions/${id}/cancel`)
  return response.data
}
