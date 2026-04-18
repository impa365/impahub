export interface EvoServer {
  id: string
  name: string
  base_url: string
  is_active: boolean
  instance_count: number
}

export interface CreateServerRequest {
  name: string
  base_url: string
  global_api_key: string
}

export interface UpdateServerRequest {
  name?: string
  base_url?: string
  global_api_key?: string
  is_active?: boolean
}

export interface TestServerResponse {
  success: boolean
  message: string
}
