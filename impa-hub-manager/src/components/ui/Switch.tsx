import { cn } from '@/lib/utils'

interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  disabled?: boolean
  className?: string
}

export function Switch({ checked, onChange, label, disabled, className }: SwitchProps) {
  return (
    <label className={cn('flex items-center justify-between gap-3 cursor-pointer select-none group', disabled && 'opacity-50 cursor-not-allowed', className)}>
      {label && (
        <span className={cn(
          'text-sm font-medium transition-colors duration-200',
          checked ? 'text-foreground' : 'text-muted-foreground'
        )}>
          {label}
        </span>
      )}
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => !disabled && onChange(!checked)}
        className={cn(
          'relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
          checked
            ? 'bg-primary'
            : 'bg-input'
        )}
      >
        <span
          className={cn(
            'pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm ring-0 transition-transform duration-200',
            checked ? 'translate-x-4' : 'translate-x-0.5'
          )}
          style={{ marginTop: '2.5px' }}
        />
      </button>
    </label>
  )
}
