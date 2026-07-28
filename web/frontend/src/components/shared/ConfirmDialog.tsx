// ConfirmDialog — the app-wide replacement for native confirm(). Promise-based:
// call useConfirm().confirm({...}) → resolves true/false. A single dialog
// instance lives at the app root (ConfirmDialogHost); the hook drives it via a
// tiny zustand store so any component can ask for confirmation without wiring
// its own dialog.
import { create } from 'zustand'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface ConfirmOptions {
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
}

interface ConfirmState {
  open: boolean
  options: ConfirmOptions | null
  resolve: ((ok: boolean) => void) | null
  confirm: (options: ConfirmOptions) => Promise<boolean>
  close: (ok: boolean) => void
}

const useConfirmStore = create<ConfirmState>((set, get) => ({
  open: false,
  options: null,
  resolve: null,
  confirm: (options) =>
    new Promise<boolean>((resolve) => {
      set({ open: true, options, resolve })
    }),
  close: (ok) => {
    get().resolve?.(ok)
    set({ open: false, resolve: null })
  },
}))

/** Returns a stable `confirm(options) => Promise<boolean>`. */
export function useConfirm() {
  return useConfirmStore((s) => s.confirm)
}

interface ControlledConfirmProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
  onConfirm: () => void
}

/** Controlled variant: caller owns open state (simpler for one-off per-page use). */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  destructive,
  onConfirm,
}: ControlledConfirmProps) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {cancelLabel ?? t('common.cancel')}
          </Button>
          <Button
            variant={destructive ? 'destructive' : 'default'}
            onClick={() => {
              onConfirm()
              onOpenChange(false)
            }}
          >
            {confirmLabel ?? t('common.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Mount once at the app root so useConfirm() has a dialog to drive. */
export function ConfirmDialogHost() {
  const { t } = useTranslation()
  const open = useConfirmStore((s) => s.open)
  const options = useConfirmStore((s) => s.options)
  const close = useConfirmStore((s) => s.close)

  return (
    <Dialog open={open} onOpenChange={(o) => !o && close(false)}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{options?.title}</DialogTitle>
          {options?.description && (
            <DialogDescription>{options.description}</DialogDescription>
          )}
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => close(false)}>
            {options?.cancelLabel ?? t('common.cancel')}
          </Button>
          <Button
            variant={options?.destructive ? 'destructive' : 'default'}
            onClick={() => close(true)}
          >
            {options?.confirmLabel ?? t('common.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
