import apiClient from './client'
import type { TypebotConfig, TypebotSetting, TypebotSession, CreateTypebotConfigRequest, UpdateTypebotConfigRequest, SetSettingsRequest, IgnoreJidRequest, ChangeTypebotStatusRequest, StartTypebotRequest } from '@/types'

export const typebotApi = {
  // CRUD Configs
  create: (data: CreateTypebotConfigRequest) =>
    apiClient.post<TypebotConfig>('/integrations/typebot/create', data).then(r => r.data),

  findAll: () =>
    apiClient.get<TypebotConfig[]>('/integrations/typebot/find').then(r => r.data ?? []),

  findByInstance: (instanceId: string) =>
    apiClient.get<TypebotConfig[]>(`/integrations/typebot/find/${instanceId}`).then(r => r.data ?? []),

  fetch: (typebotId: string) =>
    apiClient.get<TypebotConfig>(`/integrations/typebot/fetch/${typebotId}`).then(r => r.data),

  update: (typebotId: string, data: UpdateTypebotConfigRequest) =>
    apiClient.put<TypebotConfig>(`/integrations/typebot/update/${typebotId}`, data).then(r => r.data),

  delete: (typebotId: string) =>
    apiClient.delete(`/integrations/typebot/delete/${typebotId}`).then(r => r.data),

  // Settings
  setSettings: (instanceId: string, data: SetSettingsRequest) =>
    apiClient.post<TypebotSetting>(`/integrations/typebot/settings/${instanceId}`, data).then(r => r.data),

  fetchSettings: (instanceId: string) =>
    apiClient.get<TypebotSetting>(`/integrations/typebot/fetchSettings/${instanceId}`).then(r => r.data),

  // Ignore JIDs
  ignoreJid: (instanceId: string, data: IgnoreJidRequest) =>
    apiClient.post(`/integrations/typebot/ignoreJid/${instanceId}`, data).then(r => r.data),

  // Sessions
  sessions: (instanceId: string) =>
    apiClient.get<TypebotSession[]>(`/integrations/typebot/fetchSessions/${instanceId}`).then(r => r.data ?? []),

  changeStatus: (instanceId: string, data: ChangeTypebotStatusRequest) =>
    apiClient.post(`/integrations/typebot/changeStatus/${instanceId}`, data).then(r => r.data),

  start: (instanceId: string, data: StartTypebotRequest) =>
    apiClient.post(`/integrations/typebot/start/${instanceId}`, data).then(r => r.data),
}
