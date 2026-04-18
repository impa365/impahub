import { cn } from '@/lib/utils'
import { ChevronRight, Home } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'

interface BreadcrumbItem {
  label: string
  href?: string
}

interface BreadcrumbsProps {
  items?: BreadcrumbItem[]
  className?: string
}

const routeLabels: Record<string, string> = {
  '': 'Dashboard',
  servers: 'Servidores',
  instances: 'Instâncias',
  chatwoot: 'Chatwoot',
  typebot: 'Typebot',
  users: 'Usuários',
  settings: 'Configurações',
}

export function Breadcrumbs({ items, className }: BreadcrumbsProps) {
  const location = useLocation()

  const breadcrumbs: BreadcrumbItem[] = items ?? (() => {
    const segments = location.pathname.split('/').filter(Boolean)
    const result: BreadcrumbItem[] = [{ label: 'Dashboard', href: '/' }]

    let path = ''
    for (const segment of segments) {
      path += `/${segment}`
      result.push({
        label: routeLabels[segment] || segment,
        href: path,
      })
    }

    return result
  })()

  if (breadcrumbs.length <= 1) return null

  return (
    <nav className={cn('flex items-center gap-1.5 text-sm', className)}>
      {breadcrumbs.map((item, i) => {
        const isLast = i === breadcrumbs.length - 1
        return (
          <span key={i} className="flex items-center gap-1.5">
            {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50" />}
            {isLast ? (
              <span className="font-medium text-foreground">{item.label}</span>
            ) : (
              <Link
                to={item.href || '/'}
                className="text-muted-foreground hover:text-foreground transition-colors font-medium"
              >
                {i === 0 ? <Home className="h-3.5 w-3.5" /> : item.label}
              </Link>
            )}
          </span>
        )
      })}
    </nav>
  )
}
