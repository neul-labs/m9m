import apiClient from './client'
import type { Credential, CredentialCreate } from '@/types/api'

export async function getCredentials(): Promise<Credential[]> {
  const response = await apiClient.get<Credential[]>('/credentials')
  return response.data
}

export async function getCredential(id: string): Promise<Credential> {
  const response = await apiClient.get<Credential>(`/credentials/${id}`)
  return response.data
}

export async function createCredential(credential: CredentialCreate): Promise<Credential> {
  const response = await apiClient.post<Credential>('/credentials', credential)
  return response.data
}

export async function updateCredential(id: string, credential: Partial<CredentialCreate>): Promise<Credential> {
  const response = await apiClient.put<Credential>(`/credentials/${id}`, credential)
  return response.data
}

export async function deleteCredential(id: string): Promise<void> {
  await apiClient.delete(`/credentials/${id}`)
}
