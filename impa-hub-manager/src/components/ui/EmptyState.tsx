import { createElement, isValidElement, type ElementType, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Button } from './Button'

interface EmptyStateProps {
  icon?: ReactNode | ElementType<{ className?: string }>
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

export function EmptyState({ icon, title, description, action, actionLabel, onAction, className }: EmptyStateProps) {
  const renderIcon = () => {
    if (!icon) return null
    if (isValidElement(icon)) return icon

    if (typeof icon === 'function' || (typeof icon === 'object' && icon !== null && '$$typeof' in icon)) {
      const Icon = icon as ElementType<{ className?: string }>
      return createElement(Icon, { className: 'h-6 w-6 text-muted-foreground' })
    }

    return icon
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
