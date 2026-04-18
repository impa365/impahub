import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

const getApiUrl = () => {
  // Runtime config (Docker) takes priority over build-time env
  if (typeof window !== 'undefined' && (window as any).__ENV__?.VITE_API_URL) {
    return (window as any).__ENV__.VITE_API_URL
  }
  return import.meta.env.VITE_API_URL || 'http://localhost:4050/api/v1'
}

const apiClient = axios.create({
  baseURL: getApiUrl(),
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => {
    // Unwrap backend {data: ...} wrapper pattern
    if (response.data && typeof response.data === 'object' && 'data' in response.data) {
      response.data = response.data.data
    }
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default apiClient
