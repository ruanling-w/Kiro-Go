// ProxySection — global outbound proxy URL. Backend accepts a single proxyURL
// (http/https/socks5/socks5h) or empty to disable; validation mirrors the server.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useProxy } from '@/hooks/queries/useSettings'
import { useUpdateProxy } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'

const PREFIXES = ['http://', 'https://', 'socks5://', 'socks5h://']

export function ProxySection() {
  const { t } = useTranslation()
  const proxy = useProxy()
  const save = useUpdateProxy()
  const [url, setUrl] = useState('')

  useEffect(() => {
    if (proxy.data) setUrl(proxy.data.proxyURL)
  }, [proxy.data])

  function onSave() {
    const trimmed = url.trim()
    if (trimmed && !PREFIXES.some((p) => trimmed.startsWith(p))) {
      toast.error(t('detail.proxyFormatError'))
      return
    }
    save.mutate(
      { proxyURL: trimmed },
      {
        onSuccess: () => toast.success(t('settings.proxySaved')),
        onError: () => toast.error(t('common.saveFailed')),
      },
    )
  }

  return (
    <SettingsSection id="proxy" title={t('settings.proxySettings')} description={t('settings.proxyDesc')}>
      {proxy.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="space-y-2">
            <Label>{t('settings.proxyHost')}</Label>
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="socks5://user:pass@host:1080"
              autoComplete="off"
            />
          </div>
          <Button disabled={save.isPending} onClick={onSave}>
            {t('settings.saveProxy')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}
