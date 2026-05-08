import type { ReactNode, ComponentType } from 'react'
import { cn } from '@/lib/utils'
import { Button } from './Button'

interface EmptyStateProps {
  icon?: ReactNode | ComponentType<{ className?: string }>
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
    icon?: ReactNode
  }
  actionLabel?: string
  onAction?: () => void
  className?: string
}

// Helper to check if value is a React element (already rendered JSX)
const isReactElement = (value: unknown): boolean => {
  if (typeof value !== 'object' || value === null) return false
  const el = value as { $$typeof?: symbol; type?: unknown; props?: unknown }
  return el.$$typeof === Symbol.for('react.element') || 
    (el.type !== undefined && el.props !== undefined)
}

// Helper to check if value is a React component (function or forwardRef)
const isComponent = (value: unknown): value is ComponentType<{ className?: string }> => {
  return typeof value === 'function' || (typeof value === 'object' && value !== null && 'render' in value)
}

export function EmptyState({ icon, title, description, action, actionLabel, onAction, className }: EmptyStateProps) {
  const renderIcon = () => {
    if (!icon) return null
    if (isReactElement(icon)) return icon as ReactNode
    if (isComponent(icon)) {
      const IconComponent = icon
      return <IconComponent className="h-6 w-6 text-muted-foreground" />
    }
    return icon as ReactNode
  }
  return (
    <div className={cn('flex flex-col items-center justify-center py-16 px-4 text-center', className)}>
      <div className="mx-auto w-12 h-12 rounded-md bg-muted flex items-center justify-center mb-4">
        {renderIcon()}
      </div>
      <h3 className="text-base font-medium mb-1.5">{title}</h3>
      {description && (
        <p className="text-muted-foreground text-sm mb-8 max-w-sm">{description}</p>
      )}
      {action && (
        <Button onClick={action.onClick}>
          {action.icon}
          {action.label}
        </Button>
      )}
      {!action && actionLabel && onAction && (
        <Button onClick={onAction}>{actionLabel}</Button>
      )}
    </div>
  )
}
