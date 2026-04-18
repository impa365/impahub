import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { useAuthStore } from '@/store/authStore'
import LoginPage from '@/pages/Login'
import DashboardPage from '@/pages/Dashboard'
import ServersPage from '@/pages/Servers'
import InstancesPage from '@/pages/Instances'
import ChatwootPage from '@/pages/Chatwoot'
import TypebotPage from '@/pages/Typebot'
import UsersPage from '@/pages/Users'
import SettingsPage from '@/pages/Settings'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user)
  if (user?.role !== 'superadmin') return <Navigate to="/" replace />
  return <>{children}</>
}

export default function App() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  return (
    <Routes>
      <Route path="/login" element={isAuthenticated ? <Navigate to="/" replace /> : <LoginPage />} />

      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<DashboardPage />} />
        <Route path="/servers" element={<ServersPage />} />
        <Route path="/instances" element={<InstancesPage />} />
        <Route path="/chatwoot" element={<ChatwootPage />} />
        <Route path="/typebot" element={<TypebotPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route
          path="/users"
          element={
            <AdminRoute>
              <UsersPage />
            </AdminRoute>
          }
        />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
