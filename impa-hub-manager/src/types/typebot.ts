export interface TypebotConfig {
  id: string
  instance_id: string
  instance_name: string
  enabled: boolean
  description: string
  url: string
  typebot: string
  trigger_type: 'all' | 'keyword' | 'none' | 'advanced'
  trigger_operator: 'contains' | 'equals' | 'startsWith' | 'endsWith' | 'regex'
  trigger_value: string
  expire: number
  keyword_finish: string
  delay_message: number
  unknown_message: string
  listening_from_me: boolean
  stop_bot_from_me: boolean
  keep_open: boolean
  debounce_time: number
  prefilled_variables?: Record<string, unknown>
  ignore_jids?: string[]
}

export interface TypebotSetting {
  id: string
  instance_id: string
  expire: number
  keyword_finish: string
  delay_message: number
  unknown_message: string
  listening_from_me: boolean
  stop_bot_from_me: boolean
  keep_open: boolean
  debounce_time: number
  typebot_id_fallback?: string
  ignore_jids?: string[]
  fallback?: TypebotConfig
}

export interface TypebotSession {
  id: string
  typebot_config_id: string
  remote_jid: string
  push_name: string
  session_id: string
  status: 'opened' | 'closed' | 'paused'
  await_user: boolean
  created_at: string
  updated_at: string
}

export interface CreateTypebotConfigRequest {
  instance_id: string
  enabled?: boolean
  description?: string
  url: string
  typebot: string
  trigger_type?: string
  trigger_operator?: string
  trigger_value?: string
  expire?: number
  keyword_finish?: string
  delay_message?: number
  unknown_message?: string
  listening_from_me?: boolean
  stop_bot_from_me?: boolean
  keep_open?: boolean
  debounce_time?: number
  prefilled_variables?: Record<string, unknown>
  ignore_jids?: string[]
}

export interface UpdateTypebotConfigRequest {
  enabled?: boolean
  description?: string
  url?: string
  typebot?: string
  trigger_type?: string
  trigger_operator?: string
  trigger_value?: string
  expire?: number
  keyword_finish?: string
  delay_message?: number
  unknown_message?: string
  listening_from_me?: boolean
  stop_bot_from_me?: boolean
  keep_open?: boolean
  debounce_time?: number
  prefilled_variables?: Record<string, unknown>
  ignore_jids?: string[]
}

export interface SetSettingsRequest {
  expire?: number
  keyword_finish?: string
  delay_message?: number
  unknown_message?: string
  listening_from_me?: boolean
  stop_bot_from_me?: boolean
  keep_open?: boolean
  debounce_time?: number
  typebot_id_fallback?: string
  ignore_jids?: string[]
}

export interface IgnoreJidRequest {
  action: 'add' | 'remove'
  jid: string
}

export interface ChangeTypebotStatusRequest {
  remote_jid: string
  status: 'opened' | 'closed' | 'paused' | 'delete'
}

export interface StartTypebotRequest {
  remote_jid: string
  typebot_config_id?: string
  url?: string
  typebot?: string
  prefilled_variables?: Record<string, unknown>
}
