export type UserRole = 'superadmin' | 'admin' | 'user'

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
  is_active: boolean
  max_instances: number
  max_chatwoot_conns: number
  max_evo_servers: number
  can_use_chatwoot: boolean
  can_use_typebot: boolean
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface CreateUserRequest {
  name: string
  email: string
  password: string
  role: UserRole
  max_instances: number
  max_chatwoot_conns: number
  max_evo_servers: number
  can_use_chatwoot: boolean
  can_use_typebot: boolean
}

export interface UpdateQuotasRequest {
  max_instances: number
  max_chatwoot_conns: number
  max_evo_servers: number
  can_use_chatwoot: boolean
  can_use_typebot: boolean
}
