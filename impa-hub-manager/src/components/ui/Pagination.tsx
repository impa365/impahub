import { cn } from '@/lib/utils'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  totalItems?: number
  itemsPerPage?: number
  className?: string
}

export function Pagination({ currentPage, totalPages, onPageChange, totalItems, itemsPerPage, className }: PaginationProps) {
  if (totalPages <= 1) return null

  const getVisiblePages = () => {
    const pages: (number | '...')[] = []
    const delta = 1

    pages.push(1)

    const start = Math.max(2, currentPage - delta)
    const end = Math.min(totalPages - 1, currentPage + delta)

    if (start > 2) pages.push('...')

    for (let i = start; i <= end; i++) {
      pages.push(i)
    }

    if (end < totalPages - 1) pages.push('...')

    if (totalPages > 1) pages.push(totalPages)

    return pages
  }

  return (
    <div className={cn('flex items-center justify-between gap-4 pt-4', className)}>
      {totalItems !== undefined && itemsPerPage !== undefined && (
        <p className="text-xs text-muted-foreground hidden sm:block">
          Mostrando <span className="font-semibold text-foreground">{Math.min((currentPage - 1) * itemsPerPage + 1, totalItems)}</span>
          {' '}-{' '}
          <span className="font-semibold text-foreground">{Math.min(currentPage * itemsPerPage, totalItems)}</span>
          {' '}de{' '}
          <span className="font-semibold text-foreground">{totalItems}</span>
        </p>
      )}

      <div className="flex items-center gap-1 ml-auto">
        <button
          onClick={() => onPageChange(currentPage - 1)}
          disabled={currentPage === 1}
          className="inline-flex items-center justify-center h-8 w-8 rounded-lg text-sm transition-colors hover:bg-muted disabled:opacity-40 disabled:pointer-events-none"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>

        {getVisiblePages().map((page, i) => (
          page === '...' ? (
            <span key={`dots-${i}`} className="px-1 text-muted-foreground text-sm">...</span>
          ) : (
            <button
              key={page}
              onClick={() => onPageChange(page)}
              className={cn(
                'inline-flex items-center justify-center h-8 min-w-[32px] rounded-lg text-sm font-medium transition-colors',
                currentPage === page
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'hover:bg-muted text-muted-foreground'
              )}
            >
              {page}
            </button>
          )
        ))}

        <button
          onClick={() => onPageChange(currentPage + 1)}
          disabled={currentPage === totalPages}
          className="inline-flex items-center justify-center h-8 w-8 rounded-lg text-sm transition-colors hover:bg-muted disabled:opacity-40 disabled:pointer-events-none"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}
