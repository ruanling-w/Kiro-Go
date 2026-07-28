// PromptFilterSection — 3 builtin filter toggles + a custom rule list
// (regex|contains). Rules are edited locally and saved as a whole config.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { usePromptFilter } from '@/hooks/queries/useSettings'
import { useUpdatePromptFilter } from '@/hooks/mutations/useSettingsMutations'
import type { PromptFilterConfig, PromptFilterRule } from '@/types/settings'
import { SettingsSection } from './SettingsSection'

const EMPTY: PromptFilterConfig = {
  filterClaudeCode: false,
  filterEnvNoise: false,
  filterStripBoundaries: false,
  rules: [],
}

export function PromptFilterSection() {
  const { t } = useTranslation()
  const query = usePromptFilter()
  const save = useUpdatePromptFilter()
  const [cfg, setCfg] = useState<PromptFilterConfig>(EMPTY)

  useEffect(() => {
    if (query.data) setCfg({ ...EMPTY, ...query.data, rules: query.data.rules ?? [] })
  }, [query.data])

  function setBuiltin<K extends keyof PromptFilterConfig>(key: K, value: PromptFilterConfig[K]) {
    setCfg((c) => ({ ...c, [key]: value }))
  }

  function addRule(type: PromptFilterRule['type']) {
    setCfg((c) => ({ ...c, rules: [...c.rules, { type, pattern: '', enabled: true }] }))
  }

  function updateRule(i: number, patch: Partial<PromptFilterRule>) {
    setCfg((c) => ({ ...c, rules: c.rules.map((r, idx) => (idx === i ? { ...r, ...patch } : r)) }))
  }

  function removeRule(i: number) {
    setCfg((c) => ({ ...c, rules: c.rules.filter((_, idx) => idx !== i) }))
  }

  function onSave() {
    const rules = cfg.rules.filter((r) => r.pattern.trim() !== '')
    save.mutate(
      { ...cfg, rules },
      {
        onSuccess: () => toast.success(t('settings.promptFilterSaved')),
        onError: () => toast.error(t('common.saveFailed')),
      },
    )
  }

  return (
    <SettingsSection id="prompt-filter" title={t('settings.promptFilter')} description={t('settings.promptFilterDesc')}>
      {query.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="space-y-3">
            <p className="text-sm font-medium">{t('settings.builtinFilters')}</p>
            <ToggleRow
              label={t('settings.filterClaudeCode')}
              hint={t('settings.filterClaudeCodeHint')}
              checked={cfg.filterClaudeCode}
              onChange={(v) => setBuiltin('filterClaudeCode', v)}
            />
            <ToggleRow
              label={t('settings.filterEnvNoise')}
              hint={t('settings.filterEnvNoiseHint')}
              checked={cfg.filterEnvNoise}
              onChange={(v) => setBuiltin('filterEnvNoise', v)}
            />
            <ToggleRow
              label={t('settings.filterStripBoundaries')}
              hint={t('settings.filterStripBoundariesHint')}
              checked={cfg.filterStripBoundaries}
              onChange={(v) => setBuiltin('filterStripBoundaries', v)}
            />
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium">{t('settings.customRules')}</p>
            {cfg.rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('promptFilter.noRules')}</p>
            ) : (
              <div className="space-y-2">
                {cfg.rules.map((rule, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className="w-20 shrink-0 text-xs text-muted-foreground">
                      {rule.type === 'regex' ? t('promptFilter.typeRegex') : t('promptFilter.typeContains')}
                    </span>
                    <Input
                      value={rule.pattern}
                      onChange={(e) => updateRule(i, { pattern: e.target.value })}
                      placeholder={
                        rule.type === 'regex'
                          ? t('promptFilter.matchPlaceholderRegex')
                          : t('promptFilter.matchPlaceholderContains')
                      }
                    />
                    <Switch
                      checked={rule.enabled !== false}
                      onCheckedChange={(v) => updateRule(i, { enabled: v })}
                    />
                    <Button variant="ghost" size="icon-sm" onClick={() => removeRule(i)} aria-label={t('common.remove')}>
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => addRule('regex')}>
                <Plus className="size-4" />
                {t('promptFilter.addRegex')}
              </Button>
              <Button variant="outline" size="sm" onClick={() => addRule('contains')}>
                <Plus className="size-4" />
                {t('promptFilter.addContains')}
              </Button>
            </div>
          </div>

          <Button disabled={save.isPending} onClick={onSave}>
            {t('settings.savePromptFilter')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}

function ToggleRow({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string
  hint: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div>
        <Label>{label}</Label>
        <p className="mt-1 text-sm text-muted-foreground">{hint}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}
