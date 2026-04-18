import apiClient from './client'
import type { ChatwootConfig, SetChatwootConfigRequest } from '@/types'

export const chatwootApi = {
  get: (instanceId: string) =>
    apiClient.get<ChatwootConfig>(`/integrations/chatwoot/${instanceId}`).then(r => r.data),

  set: (data: SetChatwootConfigRequest) =>
    apiClient.post<ChatwootConfig>('/integrations/chatwoot/set', data).then(r => r.data),

  delete: (instanceId: string) =>
    apiClient.delete(`/integrations/chatwoot/${instanceId}`).then(r => r.data),
}
