// PromptFilterSection — 3 builtin filter toggles + a custom rule list
// (regex|contains). Rules are edited locally and saved as a whole config.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { usePromptFilter } from '@/hooks/queries/useSettings'
import { useUpdatePromptFilter } from '@/hooks/mutations/useSettingsMutations'
import type { PromptFilterConfig, PromptFilterRule } from '@/types/settings'
import { SettingsSection } from './SettingsSection'
import { SettingsToggleRow } from './SettingsToggleRow'

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
    <SettingsSection
      id="prompt-filter"
      title={t('settings.promptFilter')}
      description={t('settings.promptFilterDesc')}
    >
      {query.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="min-w-0 space-y-3">
            <p className="text-sm font-medium">{t('settings.builtinFilters')}</p>
            <SettingsToggleRow
              label={t('settings.filterClaudeCode')}
              hint={t('settings.filterClaudeCodeHint')}
              checked={cfg.filterClaudeCode}
              onChange={(v) => setBuiltin('filterClaudeCode', v)}
            />
            <SettingsToggleRow
              label={t('settings.filterEnvNoise')}
              hint={t('settings.filterEnvNoiseHint')}
              checked={cfg.filterEnvNoise}
              onChange={(v) => setBuiltin('filterEnvNoise', v)}
            />
            <SettingsToggleRow
              label={t('settings.filterStripBoundaries')}
              hint={t('settings.filterStripBoundariesHint')}
              checked={cfg.filterStripBoundaries}
              onChange={(v) => setBuiltin('filterStripBoundaries', v)}
            />
          </div>

          <div className="min-w-0 space-y-2">
            <p className="text-sm font-medium">{t('settings.customRules')}</p>
            {cfg.rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('promptFilter.noRules')}</p>
            ) : (
              <div className="space-y-2">
                {cfg.rules.map((rule, i) => (
                  <div
                    key={i}
                    className="flex min-w-0 flex-col gap-2 rounded-lg border border-border/60 p-2.5 sm:flex-row sm:items-center"
                  >
                    <span className="shrink-0 text-xs font-medium text-muted-foreground sm:w-20">
                      {rule.type === 'regex'
                        ? t('promptFilter.typeRegex')
                        : t('promptFilter.typeContains')}
                    </span>
                    <Input
                      className="min-w-0 flex-1"
                      value={rule.pattern}
                      onChange={(e) => updateRule(i, { pattern: e.target.value })}
                      placeholder={
                        rule.type === 'regex'
                          ? t('promptFilter.matchPlaceholderRegex')
                          : t('promptFilter.matchPlaceholderContains')
                      }
                    />
                    <div className="flex shrink-0 items-center justify-end gap-2">
                      <Switch
                        checked={rule.enabled !== false}
                        onCheckedChange={(v) => updateRule(i, { enabled: v })}
                      />
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => removeRule(i)}
                        aria-label={t('common.remove')}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
            <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-center sm:w-auto"
                onClick={() => addRule('regex')}
              >
                <Plus className="size-4" />
                {t('promptFilter.addRegex')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-center sm:w-auto"
                onClick={() => addRule('contains')}
              >
                <Plus className="size-4" />
                {t('promptFilter.addContains')}
              </Button>
            </div>
          </div>

          <Button className="w-full sm:w-auto" disabled={save.isPending} onClick={onSave}>
            {t('settings.savePromptFilter')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}
