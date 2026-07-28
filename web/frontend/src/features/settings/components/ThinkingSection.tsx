// ThinkingSection — thinking tag suffix + openai/claude output format selects.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useThinking } from '@/hooks/queries/useSettings'
import { useUpdateThinking } from '@/hooks/mutations/useSettingsMutations'
import type { ThinkingFormat } from '@/types/settings'
import { SettingsSection } from './SettingsSection'

const FORMATS: { value: ThinkingFormat; key: string }[] = [
  { value: 'reasoning_content', key: 'settings.formatReasoningContent' },
  { value: 'thinking', key: 'settings.formatThinkingClaude' },
  { value: 'think', key: 'settings.formatThinkOpenAI' },
]

export function ThinkingSection() {
  const { t } = useTranslation()
  const query = useThinking()
  const save = useUpdateThinking()
  const [suffix, setSuffix] = useState('')
  const [openai, setOpenai] = useState<ThinkingFormat>('reasoning_content')
  const [claude, setClaude] = useState<ThinkingFormat>('thinking')

  useEffect(() => {
    if (query.data) {
      setSuffix(query.data.suffix)
      setOpenai(query.data.openaiFormat)
      setClaude(query.data.claudeFormat)
    }
  }, [query.data])

  return (
    <SettingsSection id="thinking" title={t('settings.thinkingSettings')} description={t('settings.thinkingDesc')}>
      {query.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="space-y-2">
            <Label>{t('settings.thinkingSuffix')}</Label>
            <Input value={suffix} onChange={(e) => setSuffix(e.target.value)} placeholder="-thinking" />
            <p className="text-sm text-muted-foreground">{t('settings.thinkingSuffixHint')}</p>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t('settings.openaiFormat')}</Label>
              <Select value={openai} onValueChange={(v) => setOpenai(v as ThinkingFormat)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {FORMATS.map((f) => (
                    <SelectItem key={f.value} value={f.value}>{t(f.key)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t('settings.claudeFormat')}</Label>
              <Select value={claude} onValueChange={(v) => setClaude(v as ThinkingFormat)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {FORMATS.map((f) => (
                    <SelectItem key={f.value} value={f.value}>{t(f.key)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <Button
            disabled={save.isPending}
            onClick={() =>
              save.mutate(
                { suffix, openaiFormat: openai, claudeFormat: claude },
                {
                  onSuccess: () => toast.success(t('settings.thinkingSaved')),
                  onError: () => toast.error(t('common.saveFailed')),
                },
              )
            }
          >
            {t('settings.saveThinking')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}
