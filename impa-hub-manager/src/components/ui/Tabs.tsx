import { createElement, isValidElement, type ElementType, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Tab {
  id: string
  label: string
  icon?: ReactNode | ElementType<{ className?: string }>
  count?: number
}

type TabInput = Tab | string

interface TabsProps {
  tabs: TabInput[]
  activeTab: string
  onChange: (id: string) => void
  variant?: 'underline' | 'pills' | 'boxed'
  className?: string
}

function normalizeTab(tab: TabInput, index: number): Tab {
  if (typeof tab === 'string') return { id: String(index), label: tab }
  return tab
}

export function Tabs({ tabs, activeTab, onChange, variant = 'underline', className }: TabsProps) {
  const normalizedTabs = tabs.map(normalizeTab)

  const renderTabIcon = (icon?: ReactNode | ElementType<{ className?: string }>) => {
    if (!icon) return null
    if (isValidElement(icon)) return icon
    if (typeof icon === 'function' || (typeof icon === 'object' && icon !== null && '$$typeof' in icon)) {
      const Icon = icon as ElementType<{ className?: string }>
      return createElement(Icon, { className: 'h-4 w-4' })
    }
    return icon
  }

  return (
    <div
      className={cn(
        'flex gap-1',
        variant === 'underline' && 'border-b border-border gap-0',
        variant === 'boxed' && 'bg-muted/50 p-1 rounded-xl',
        className
      )}
    >
      {normalizedTabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onChange(tab.id)}
          className={cn(
            'inline-flex items-center gap-2 text-sm font-medium transition-all duration-200 cursor-pointer select-none',
            variant === 'underline' && cn(
              'px-4 py-2.5 border-b-2 -mb-px',
              activeTab === tab.id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'
            ),
            variant === 'pills' && cn(
              'px-4 py-2 rounded-lg',
              activeTab === tab.id
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            ),
            variant === 'boxed' && cn(
              'px-4 py-2 rounded-lg flex-1 justify-center',
              activeTab === tab.id
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            ),
          )}
        >
          {renderTabIcon(tab.icon)}
          {tab.label}
          {tab.count !== undefined && (
            <span className={cn(
              'text-[10px] font-bold rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1',
              activeTab === tab.id
                ? variant === 'pills' ? 'bg-white/20 text-white' : 'bg-muted text-foreground'
                : 'bg-muted text-muted-foreground'
            )}>
              {tab.count}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}

interface TabPanelProps {
  value?: string
  index?: number
  activeTab: string
  children: ReactNode
  className?: string
}

export function TabPanel({ value, index, activeTab, children, className }: TabPanelProps) {
  const panelId = value ?? String(index ?? '')
  if (panelId !== activeTab) return null
  return (
    <div className={cn('animate-fade-in', className)}>
      {children}
    </div>
  )
}
