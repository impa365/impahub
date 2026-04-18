export type ConnectionState = 'connected' | 'disconnected' | 'connecting'

export interface Instance {
  id: string
  server_id: string
  server_name: string
  evo_instance_id: string
  instance_name: string
  connection_status: ConnectionState
  phone?: string
  push_name?: string
  webhook_configured: boolean
  has_chatwoot: boolean
  has_typebot: boolean
}

export interface CreateInstanceRequest {
  server_id: string
  instance_name: string
  phone?: string
}

export interface QRCodeResponse {
  qrcode: string
  code?: string
}

export interface SendMessageRequest {
  number: string
  text: string
}

export interface AdvancedSettings {
  alwaysOnline: boolean
  rejectCall: boolean
  msgRejectCall: string
  readMessages: boolean
  ignoreGroups: boolean
  ignoreStatus: boolean
}

export interface PairRequest {
  phone: string
}
