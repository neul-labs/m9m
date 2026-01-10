import apiClient from './client'
import type {
  Schedule,
  ScheduleFilters,
  ScheduleCreateRequest,
  ScheduleUpdateRequest,
  ExecutionHistory,
} from '@/types/schedule'

interface ScheduleListResponse {
  data: Schedule[]
  total: number
}

export async function getSchedules(filters?: ScheduleFilters): Promise<ScheduleListResponse> {
  const params = new URLSearchParams()
  if (filters?.workflowId) params.append('workflowId', filters.workflowId)
  if (filters?.enabled !== undefined) params.append('enabled', String(filters.enabled))
  if (filters?.search) params.append('search', filters.search)
  if (filters?.offset !== undefined) params.append('offset', String(filters.offset))
  if (filters?.limit !== undefined) params.append('limit', String(filters.limit))

  const response = await apiClient.get<ScheduleListResponse>('/schedules', { params })
  return response.data
}

export async function getSchedule(id: string): Promise<Schedule> {
  const response = await apiClient.get<Schedule>(`/schedules/${id}`)
  return response.data
}

export async function createSchedule(data: ScheduleCreateRequest): Promise<Schedule> {
  const response = await apiClient.post<Schedule>('/schedules', data)
  return response.data
}

export async function updateSchedule(id: string, data: ScheduleUpdateRequest): Promise<Schedule> {
  const response = await apiClient.put<Schedule>(`/schedules/${id}`, data)
  return response.data
}

export async function deleteSchedule(id: string): Promise<void> {
  await apiClient.delete(`/schedules/${id}`)
}

export async function enableSchedule(id: string): Promise<Schedule> {
  const response = await apiClient.post<Schedule>(`/schedules/${id}/enable`)
  return response.data
}

export async function disableSchedule(id: string): Promise<Schedule> {
  const response = await apiClient.post<Schedule>(`/schedules/${id}/disable`)
  return response.data
}

export async function getScheduleHistory(id: string, limit = 50): Promise<ExecutionHistory> {
  const response = await apiClient.get<ExecutionHistory>(`/schedules/${id}/history`, {
    params: { limit },
  })
  return response.data
}
