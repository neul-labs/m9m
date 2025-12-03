import apiClient from './client'
import type { NodeType } from '@/types'

export async function getNodeTypes(): Promise<NodeType[]> {
  const response = await apiClient.get<NodeType[]>('/node-types')
  return response.data
}

export async function getNodeType(name: string): Promise<NodeType> {
  const response = await apiClient.get<NodeType>(`/node-types/${encodeURIComponent(name)}`)
  return response.data
}
