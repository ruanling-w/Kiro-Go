// CopyButton — copies text to clipboard with a transient check-mark + toast.
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { cn } from '@/lib/utils'

interface CopyButtonProps {
  value: string
  label?: string
  size?: 'icon' | 'icon-sm' | 'icon-xs' | 'sm' | 'default'
  variant?: 'ghost' | 'outline' | 'secondary'
  className?: string
}

export function CopyButton({
  value,
  label,
  size = 'icon-sm',
  variant = 'ghost',
  className,
}: CopyButtonProps) {
  const { copied, copy } = useCopyToClipboard()
  const Icon = copied ? Check : Copy
  const isIcon = size.startsWith('icon')
  return (
    <Button
      type="button"
      variant={variant}
      size={size}
      className={className}
      onClick={() => void copy(value)}
      aria-label={label ?? 'Copy'}
    >
      <Icon className={cn('size-4', copied && 'text-emerald-500')} />
      {!isIcon && label ? <span>{label}</span> : null}
    </Button>
  )
}
