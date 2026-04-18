export interface ChatwootConfig {
  id: string
  instance_id: string
  instance_name: string
  enabled: boolean
  account_id: string
  token: string
  url: string
  sign_msg: boolean
  sign_delimiter: string
  reopen_conversation: boolean
  conversation_pending: boolean
  merge_brazil_contacts: boolean
  auto_create: boolean
  inbox_name: string
  inbox_id: number
  is_active: boolean
  webhook_url: string
  groups_ignore: boolean
  ignore_jids: string
  created_at?: string
  updated_at?: string
}

export interface SetChatwootConfigRequest {
  instance_id: string
  account_id: string
  token: string
  url: string
  enabled?: boolean
  sign_msg?: boolean
  sign_delimiter?: string
  reopen_conversation?: boolean
  conversation_pending?: boolean
  auto_create?: boolean
  inbox_name?: string
  groups_ignore?: boolean
  ignore_jids?: string
}
