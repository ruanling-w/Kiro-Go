// ToolGuideTabs — Claude Code / Codex / cURL / SDK / GUI snippet tabs.
import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeBlock } from '@/components/shared/CodeBlock'
import {
  TOOL_GUIDES,
  fillSnippet,
  type SnippetVars,
} from '@/config/apiDocs'

interface ToolGuideTabsProps {
  vars: SnippetVars
}

export function ToolGuideTabs({ vars }: ToolGuideTabsProps) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('apiDocs.guidesTitle')}</CardTitle>
        <CardDescription>{t('apiDocs.guidesDesc')}</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue={TOOL_GUIDES[0]?.id}>
          <TabsList variant="line" className="mb-4 flex h-auto w-full flex-wrap justify-start gap-1">
            {TOOL_GUIDES.map((g) => {
              const Icon = g.icon
              return (
                <TabsTrigger key={g.id} value={g.id} className="gap-1.5">
                  {g.logo ? (
                    <img src={g.logo} alt="" className="size-3.5 object-contain" />
                  ) : (
                    <Icon className="size-3.5" />
                  )}
                  {t(g.labelKey)}
                </TabsTrigger>
              )
            })}
          </TabsList>

          {TOOL_GUIDES.map((g) => (
            <TabsContent key={g.id} value={g.id} className="space-y-4">
              {g.blocks.map((b) => {
                const code = fillSnippet(b.code, vars)
                return (
                  <div key={b.titleKey} className="space-y-2">
                    <h3 className="text-sm font-medium">{t(b.titleKey)}</h3>
                    <CodeBlock
                      code={code}
                      lang={b.lang}
                      filename={b.filename}
                      copyLabel={t('common.copy')}
                    />
                    {b.noteKey ? (
                      <p className="text-xs text-muted-foreground">{t(b.noteKey)}</p>
                    ) : null}
                  </div>
                )
              })}
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  )
}
