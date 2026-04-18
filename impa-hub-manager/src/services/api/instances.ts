import apiClient from './client'
import type { Instance, CreateInstanceRequest, QRCodeResponse, SendMessageRequest, AdvancedSettings, PairRequest } from '@/types'

export const instancesApi = {
  list: () =>
    apiClient.get<Instance[]>('/instances').then(r => r.data ?? []),

  get: (id: string) =>
    apiClient.get<Instance>(`/instances/${id}`).then(r => r.data),

  create: (data: CreateInstanceRequest) =>
    apiClient.post<Instance>('/instances', data).then(r => r.data),

  delete: (id: string) =>
    apiClient.delete(`/instances/${id}`).then(r => r.data),

  connect: (id: string) =>
    apiClient.post(`/instances/${id}/connect`).then(r => r.data),

  disconnect: (id: string) =>
    apiClient.post(`/instances/${id}/disconnect`).then(r => r.data),

  logout: (id: string) =>
    apiClient.post(`/instances/${id}/logout`).then(r => r.data),

  reconnect: (id: string) =>
    apiClient.post(`/instances/${id}/reconnect`).then(r => r.data),

  qrcode: (id: string) =>
    apiClient.get<QRCodeResponse>(`/instances/${id}/qr`).then(r => r.data),

  sendMessage: (id: string, data: SendMessageRequest) =>
    apiClient.post(`/instances/${id}/send`, data).then(r => r.data),

  refreshStatus: (id: string) =>
    apiClient.get(`/instances/${id}/status`).then(r => r.data),

  pair: (id: string, data: PairRequest) =>
    apiClient.post(`/instances/${id}/pair`, data).then(r => r.data),

  getAdvancedSettings: (id: string) =>
    apiClient.get<AdvancedSettings>(`/instances/${id}/advanced-settings`).then(r => r.data),

  updateAdvancedSettings: (id: string, data: AdvancedSettings) =>
    apiClient.put(`/instances/${id}/advanced-settings`, data).then(r => r.data),
}
