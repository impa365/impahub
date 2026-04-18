import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { cva, type VariantProps } from 'class-variance-authority'
import { Info, CheckCircle2, AlertTriangle, XCircle, X } from 'lucide-react'

const alertVariants = cva(
  'relative flex items-start gap-3 rounded-xl border p-4 text-sm',
  {
    variants: {
      variant: {
        info: 'bg-blue-500/8 border-blue-500/20 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/15',
        success: 'bg-emerald-500/8 border-emerald-500/20 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/15',
        warning: 'bg-amber-500/8 border-amber-500/20 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/15',
        error: 'bg-red-500/8 border-red-500/20 text-red-700 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/15',
      },
    },
    defaultVariants: { variant: 'info' },
  }
)

const iconMap = {
  info: Info,
  success: CheckCircle2,
  warning: AlertTriangle,
  error: XCircle,
}

interface AlertProps extends VariantProps<typeof alertVariants> {
  title?: string
  children: ReactNode
  onDismiss?: () => void
  className?: string
}

export function Alert({ variant = 'info', title, children, onDismiss, className }: AlertProps) {
  const Icon = iconMap[variant!]

  return (
    <div className={cn(alertVariants({ variant }), className)}>
      <Icon className="h-5 w-5 shrink-0 mt-0.5" />
      <div className="flex-1 space-y-1">
        {title && <p className="font-semibold">{title}</p>}
        <div className="text-sm opacity-90">{children}</div>
      </div>
      {onDismiss && (
        <button onClick={onDismiss} className="shrink-0 p-0.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/10 transition-colors">
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}
