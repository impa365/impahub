import apiClient from './client'
import type { EvoServer, CreateServerRequest, UpdateServerRequest, TestServerResponse } from '@/types'

export const serversApi = {
  list: () =>
    apiClient.get<EvoServer[]>('/servers').then(r => r.data ?? []),

  get: (id: string) =>
    apiClient.get<EvoServer>(`/servers/${id}`).then(r => r.data),

  create: (data: CreateServerRequest) =>
    apiClient.post<EvoServer>('/servers', data).then(r => r.data),

  update: (id: string, data: UpdateServerRequest) =>
    apiClient.put<EvoServer>(`/servers/${id}`, data).then(r => r.data),

  delete: (id: string) =>
    apiClient.delete(`/servers/${id}`).then(r => r.data),

  test: (id: string) =>
    apiClient.post<TestServerResponse>(`/servers/${id}/test`).then(r => r.data),
}
