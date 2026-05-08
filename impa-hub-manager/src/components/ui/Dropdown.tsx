import { createElement, isValidElement, useEffect, useRef, useState, type ElementType, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { ChevronDown } from 'lucide-react'

interface DropdownItem {
  label: string
  icon?: ReactNode | ElementType<{ className?: string }>
  onClick: () => void
  variant?: 'default' | 'danger'
  disabled?: boolean
}

interface DropdownProps {
  trigger: ReactNode
  items: (DropdownItem | 'separator')[]
  align?: 'left' | 'right'
  className?: string
}

export function Dropdown({ trigger, items, align = 'left', className }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const renderItemIcon = (icon?: ReactNode | ElementType<{ className?: string }>) => {
    if (!icon) return null
    if (isValidElement(icon)) return icon
    if (typeof icon === 'function' || (typeof icon === 'object' && icon !== null && '$$typeof' in icon)) {
      const Icon = icon as ElementType<{ className?: string }>
      return createElement(Icon, { className: 'h-4 w-4' })
    }
    return icon
  }

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    if (open) document.addEventListener('keydown', handleEsc)
    return () => document.removeEventListener('keydown', handleEsc)
  }, [open])

  return (
    <div ref={ref} className={cn('relative inline-flex', className)}>
      <div onClick={() => setOpen(!open)} className="cursor-pointer">
        {trigger}
      </div>

      {open && (
        <div
          className={cn(
            'absolute top-full mt-1.5 z-50 min-w-[180px] rounded-xl border border-border bg-card/95 backdrop-blur-xl p-1.5 shadow-2xl shadow-black/20 animate-scale-in',
            align === 'right' ? 'right-0' : 'left-0'
          )}
        >
          {items.map((item, i) => {
            if (item === 'separator') {
              return <div key={`sep-${i}`} className="my-1 h-px bg-border" />
            }
            return (
              <button
                key={i}
                onClick={() => {
                  if (!item.disabled) {
                    item.onClick()
                    setOpen(false)
                  }
                }}
                disabled={item.disabled}
                className={cn(
                  'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors',
                  item.variant === 'danger'
                    ? 'text-destructive hover:bg-destructive/10'
                    : 'text-foreground hover:bg-white/5',
                  item.disabled && 'opacity-40 pointer-events-none'
                )}
              >
                {item.icon && <span className="shrink-0">{renderItemIcon(item.icon)}</span>}
                {item.label}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
