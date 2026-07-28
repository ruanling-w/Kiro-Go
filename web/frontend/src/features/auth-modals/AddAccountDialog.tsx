// AddAccountDialog — provider/method picker → renders the chosen flow. Two-step:
// first a grid of methods (from the flow registry, grouped by provider bucket),
// then the selected flow's component. "Back" returns to the picker; closing or a
// successful add resets to the picker and closes.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { PROVIDERS } from '@/config/providers'
import { FLOW_ENTRIES, type FlowEntry } from './flows/registry'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Scope the method picker to one provider bucket (provider-specific page). */
  provider?: string
}

export function AddAccountDialog({ open, onOpenChange, provider }: Props) {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<FlowEntry | null>(null)

  function close() {
    setSelected(null)
    onOpenChange(false)
  }

  const Flow = selected?.Component
  const buckets = provider ? PROVIDERS.filter((p) => p.key === provider) : PROVIDERS

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) close()
        else onOpenChange(true)
      }}
    >
      <DialogContent className="sm:max-w-2xl [&>*]:min-w-0">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {selected && (
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => setSelected(null)}
                aria-label={t('common.back')}
              >
                <ChevronLeft className="size-4" />
              </Button>
            )}
            {selected ? t(selected.labelKey) : t('accounts.add')}
          </DialogTitle>
        </DialogHeader>

        {!selected ? (
          <div className="space-y-5">
            {buckets.map((p) => {
              const entries = FLOW_ENTRIES.filter((f) => f.group === p.key)
              if (entries.length === 0) return null
              return (
                <div key={p.key} className="space-y-2">
                  {buckets.length > 1 && (
                    <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      {t(p.labelKey)}
                    </p>
                  )}
                  <div className="grid grid-cols-2 gap-2">
                    {entries.map((entry) => {
                      const Icon = entry.icon
                      return (
                        <button
                          key={entry.id}
                          type="button"
                          onClick={() => setSelected(entry)}
                          className="flex items-center gap-2 rounded-lg border p-3 text-left text-sm hover:bg-muted"
                        >
                          <Icon className="size-4 shrink-0 text-muted-foreground" />
                          <span>{t(entry.labelKey)}</span>
                        </button>
                      )
                    })}
                  </div>
                </div>
              )
            })}
          </div>
        ) : Flow ? (
          <Flow onDone={close} />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
