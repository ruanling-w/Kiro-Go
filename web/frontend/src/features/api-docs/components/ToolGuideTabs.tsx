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
    <Card className="min-w-0">
      <CardHeader className="gap-1.5">
        <CardTitle>{t('apiDocs.guidesTitle')}</CardTitle>
        <CardDescription>{t('apiDocs.guidesDesc')}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0">
        <Tabs defaultValue={TOOL_GUIDES[0]?.id} className="min-w-0">
          {/* Scroll tabs horizontally on narrow screens instead of wrapping into a tall stack. */}
          <div className="-mx-1 mb-4 overflow-x-auto px-1 pb-1">
            <TabsList
              variant="line"
              className="inline-flex h-auto w-max min-w-full flex-nowrap justify-start gap-1"
            >
              {TOOL_GUIDES.map((g) => {
                const Icon = g.icon
                return (
                  <TabsTrigger
                    key={g.id}
                    value={g.id}
                    className="shrink-0 flex-none gap-1.5 px-2.5"
                  >
                    {g.logo ? (
                      <img src={g.logo} alt="" className="size-3.5 object-contain" />
                    ) : (
                      <Icon className="size-3.5" />
                    )}
                    <span className="whitespace-nowrap">{t(g.labelKey)}</span>
                  </TabsTrigger>
                )
              })}
            </TabsList>
          </div>

          {TOOL_GUIDES.map((g) => (
            <TabsContent key={g.id} value={g.id} className="min-w-0 space-y-4">
              {g.blocks.map((b) => {
                const code = fillSnippet(b.code, vars)
                return (
                  <div key={b.titleKey} className="min-w-0 space-y-2">
                    <h3 className="text-sm font-medium">{t(b.titleKey)}</h3>
                    <CodeBlock
                      code={code}
                      lang={b.lang}
                      filename={b.filename}
                      copyLabel={t('common.copy')}
                    />
                    {b.noteKey ? (
                      <p className="text-xs leading-relaxed text-muted-foreground">
                        {t(b.noteKey)}
                      </p>
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
