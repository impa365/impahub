import { Moon, Sun, Menu } from 'lucide-react'
import { useTheme } from '@/contexts/ThemeContext'
import { useAuthStore } from '@/store/authStore'
import { Breadcrumbs } from '@/components/ui'

interface HeaderProps {
  onMobileMenuToggle: () => void
}

export function Header({ onMobileMenuToggle }: HeaderProps) {
  const { theme, toggleTheme } = useTheme()
  const { user } = useAuthStore()

  const roleLabel: Record<string, string> = {
    superadmin: 'Super Admin',
    admin: 'Admin',
    user: 'Usuário',
  }

  return (
    <header className="glass sticky top-0 z-30 border-b border-border">
      <div className="flex h-16 items-center justify-between px-4 md:px-6">
        <div className="flex items-center gap-3">
          <button className="md:hidden rounded-lg p-2 text-muted-foreground hover:bg-white/5 hover:text-foreground transition-colors" onClick={onMobileMenuToggle}>
            <Menu className="h-5 w-5" />
          </button>
          <Breadcrumbs className="hidden md:flex" />
        </div>

        <div className="flex items-center gap-3">
          <button onClick={toggleTheme} className="rounded-lg p-2 text-muted-foreground hover:bg-white/5 hover:text-foreground transition-colors">
            {theme === 'dark' ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
          </button>
          <div className="hidden sm:flex items-center gap-2.5 rounded-lg px-2 py-1.5 hover:bg-white/5 transition-colors">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/20 text-primary text-xs font-semibold">
              {user?.name?.charAt(0)?.toUpperCase() || 'U'}
            </div>
            <div className="text-right">
              <p className="text-sm font-medium leading-none">{user?.name?.split(' ')[0]}</p>
              <p className="text-[10px] text-muted-foreground">{roleLabel[user?.role || 'user']}</p>
            </div>
          </div>
        </div>
      </div>
    </header>
  )
}
