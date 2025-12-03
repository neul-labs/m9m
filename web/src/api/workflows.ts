import apiClient from './client'
import type { Workflow, WorkflowFilters, DataItem } from '@/types'
import type { PaginatedResponse } from '@/types/api'

export async function getWorkflows(filters?: WorkflowFilters): Promise<PaginatedResponse<Workflow>> {
  const params = new URLSearchParams()
  if (filters?.search) params.append('search', filters.search)
  if (filters?.active !== undefined) params.append('active', String(filters.active))
  if (filters?.offset !== undefined) params.append('offset', String(filters.offset))
  if (filters?.limit !== undefined) params.append('limit', String(filters.limit))

  const response = await apiClient.get<PaginatedResponse<Workflow>>('/workflows', { params })
  return response.data
}

export async function getWorkflow(id: string): Promise<Workflow> {
  const response = await apiClient.get<Workflow>(`/workflows/${id}`)
  return response.data
}

export async function createWorkflow(workflow: Partial<Workflow>): Promise<Workflow> {
  const response = await apiClient.post<Workflow>('/workflows', workflow)
  return response.data
}

export async function updateWorkflow(id: string, workflow: Partial<Workflow>): Promise<Workflow> {
  const response = await apiClient.put<Workflow>(`/workflows/${id}`, workflow)
  return response.data
}

export async function deleteWorkflow(id: string): Promise<void> {
  await apiClient.delete(`/workflows/${id}`)
}

export async function activateWorkflow(id: string): Promise<{ message: string; active: boolean }> {
  const response = await apiClient.post<{ message: string; active: boolean }>(`/workflows/${id}/activate`)
  return response.data
}

export async function deactivateWorkflow(id: string): Promise<{ message: string; active: boolean }> {
  const response = await apiClient.post<{ message: string; active: boolean }>(`/workflows/${id}/deactivate`)
  return response.data
}

export async function executeWorkflow(id: string, inputData?: DataItem[]): Promise<import('@/types').WorkflowExecution> {
  const response = await apiClient.post(`/workflows/${id}/execute`, inputData)
  return response.data
}

export async function duplicateWorkflow(id: string): Promise<Workflow> {
  const original = await getWorkflow(id)
  const duplicate: Partial<Workflow> = {
    ...original,
    id: undefined,
    name: `${original.name} (copy)`,
    active: false,
    createdAt: undefined,
    updatedAt: undefined,
  }
  return createWorkflow(duplicate)
}
