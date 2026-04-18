import apiClient from './client'
import type { LoginRequest, LoginResponse, ChangePasswordRequest } from '@/types'

export const authApi = {
  login: (data: LoginRequest) =>
    apiClient.post<LoginResponse>('/auth/login', data).then(r => r.data),

  changePassword: (data: ChangePasswordRequest) =>
    apiClient.post('/auth/change-password', data).then(r => r.data),

  me: () =>
    apiClient.get('/auth/me').then(r => r.data),
}
