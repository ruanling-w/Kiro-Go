// ApiKeyRevealCell — shows the masked key with a reveal toggle. Revealing fetches
// the cleartext once (GET /api-keys/{id}/reveal) and lets the user copy it. The
// cleartext is kept only in local component state, never cached.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { CopyButton } from '@/components/shared/CopyButton'
import { revealApiKey } from '@/services/apikeys.service'
import { toast } from 'sonner'

export function ApiKeyRevealCell({ id, masked }: { id: string; masked: string }) {
  const { t } = useTranslation()
  const [revealed, setRevealed] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function toggle() {
    if (revealed) {
      setRevealed(null)
      return
    }
    setLoading(true)
    try {
      const res = await revealApiKey(id)
      setRevealed(res.key)
    } catch {
      toast.error(t('apiKeys.revealFailed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex items-center gap-1">
      <code className="max-w-40 truncate font-mono text-xs">{revealed ?? masked}</code>
      <Button
        variant="ghost"
        size="icon-xs"
        onClick={toggle}
        disabled={loading}
        aria-label={revealed ? t('apiKeys.hideKey') : t('apiKeys.showKey')}
      >
        {revealed ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
      </Button>
      {revealed && <CopyButton value={revealed} size="icon-xs" label={t('apiKeys.copyKey')} />}
    </div>
  )
}
