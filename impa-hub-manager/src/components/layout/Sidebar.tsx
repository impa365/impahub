import { NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Server,
  Smartphone,
  MessageSquare,
  Bot,
  Users,
  Settings,
  LogOut,
  ChevronLeft,
  Zap,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/store/authStore'
import { Tooltip } from '@/components/ui'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

const mainNav = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/servers', icon: Server, label: 'Servidores' },
  { to: '/instances', icon: Smartphone, label: 'Instâncias' },
  { to: '/chatwoot', icon: MessageSquare, label: 'Chatwoot' },
  { to: '/typebot', icon: Bot, label: 'Typebot' },
]

const adminNav = [
  { to: '/users', icon: Users, label: 'Usuários' },
]

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const navigate = useNavigate()
  const { user, logout } = useAuthStore()
  const isSuperAdmin = user?.role === 'superadmin'

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200',
      isActive
        ? 'bg-primary/15 text-primary shadow-lg shadow-primary/10'
        : 'text-muted-foreground hover:bg-white/5 hover:text-foreground'
    )

  const userInitials = user?.name
    ?.split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2) || '?'

  return (
    <aside
      className={cn(
        'flex flex-col bg-sidebar text-sidebar-foreground border-r border-sidebar-border transition-all duration-300 ease-out h-screen',
        collapsed ? 'w-20' : 'w-64'
      )}
    >
      {/* Logo */}
      <div className="flex h-16 items-center justify-between px-4 border-b border-sidebar-border">
        {!collapsed && (
          <div className="flex items-center gap-2.5">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-primary/70 shadow-lg">
              <Zap className="h-5 w-5 text-primary-foreground" />
            </div>
            <div>
              <span className="text-sm font-bold tracking-tight">ImpaHub</span>
              <p className="text-[10px] text-sidebar-foreground/60">Manager</p>
            </div>
          </div>
        )}
        {collapsed && (
          <div className="flex h-9 w-9 mx-auto items-center justify-center rounded-lg bg-gradient-to-br from-primary to-primary/70 shadow-lg">
            <Zap className="h-5 w-5 text-primary-foreground" />
          </div>
        )}
        <Tooltip content={collapsed ? 'Expandir' : 'Recolher'} side="right">
          <button
            onClick={onToggle}
            className="shrink-0 rounded-lg p-1.5 text-sidebar-foreground/50 hover:bg-white/10 hover:text-sidebar-foreground transition-colors"
          >
            <ChevronLeft className={cn('h-4 w-4 transition-transform duration-300', collapsed && 'rotate-180')} />
          </button>
        </Tooltip>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 px-3 pt-6 overflow-y-auto">
        <div className="space-y-1">
          {!collapsed && (
            <p className="mb-3 px-3 text-[11px] font-semibold uppercase tracking-wider text-sidebar-foreground/40">
              Menu
            </p>
          )}
          {mainNav.map((item) => (
            <Tooltip key={item.to} content={item.label} side="right" disabled={!collapsed}>
              <NavLink to={item.to} end={item.to === '/'} className={linkClass}>
                <item.icon className="h-4 w-4 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </NavLink>
            </Tooltip>
          ))}
        </div>

        {isSuperAdmin && (
          <div className="mt-8 space-y-1">
            {!collapsed && (
              <p className="mb-3 px-3 text-[11px] font-semibold uppercase tracking-wider text-sidebar-foreground/40">
                Admin
              </p>
            )}
            {adminNav.map((item) => (
              <Tooltip key={item.to} content={item.label} side="right" disabled={!collapsed}>
                <NavLink to={item.to} className={linkClass}>
                  <item.icon className="h-4 w-4 shrink-0" />
                  {!collapsed && <span>{item.label}</span>}
                </NavLink>
              </Tooltip>
            ))}
          </div>
        )}
      </nav>

      {/* Footer */}
      <div className="border-t border-sidebar-border p-3 space-y-3">
        {!collapsed && (
          <div className="glass-sm p-3 space-y-2">
            <div className="flex items-center gap-2">
              <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center">
                <span className="text-xs font-semibold text-primary">{userInitials}</span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs font-semibold truncate">{user?.name}</p>
                <p className="text-[10px] text-muted-foreground capitalize">{user?.role === 'superadmin' ? 'Super Admin' : user?.role}</p>
              </div>
            </div>
          </div>
        )}

        <Tooltip content="Configurações" side="right" disabled={!collapsed}>
          <NavLink to="/settings" className={linkClass}>
            <Settings className="h-4 w-4 shrink-0" />
            {!collapsed && <span>Configurações</span>}
          </NavLink>
        </Tooltip>
        <Tooltip content="Sair" side="right" disabled={!collapsed}>
          <button
            onClick={handleLogout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            {!collapsed && <span>Sair</span>}
          </button>
        </Tooltip>
      </div>
    </aside>
  )
}
