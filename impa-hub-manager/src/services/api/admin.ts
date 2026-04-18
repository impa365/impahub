import apiClient from './client'
import type { User, CreateUserRequest, UpdateQuotasRequest } from '@/types'

export const adminApi = {
  listUsers: () =>
    apiClient.get<User[]>('/admin/users').then(r => r.data ?? []),

  createUser: (data: CreateUserRequest) =>
    apiClient.post<User>('/admin/users', data).then(r => r.data),

  updateQuotas: (userId: string, data: UpdateQuotasRequest) =>
    apiClient.put(`/admin/users/${userId}/quotas`, data).then(r => r.data),

  toggleActive: (userId: string, isActive: boolean) =>
    apiClient.put(`/admin/users/${userId}`, { is_active: isActive }).then(r => r.data),

  resetPassword: (userId: string, newPassword: string) =>
    apiClient.post(`/admin/users/${userId}/reset-password`, { new_password: newPassword }).then(r => r.data),

  deleteUser: (userId: string) =>
    apiClient.delete(`/admin/users/${userId}`).then(r => r.data),
}
