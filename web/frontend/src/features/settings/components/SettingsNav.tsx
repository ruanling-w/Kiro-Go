// SettingsNav — sticky scroll-spy sidebar. Observes each section's id via
// IntersectionObserver and highlights the active one; clicking scrolls to it.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

export interface NavSection {
  id: string
  labelKey: string
}

export const SETTINGS_SECTIONS: NavSection[] = [
  { id: 'usage', labelKey: 'settings.navUsage' },
  { id: 'thinking', labelKey: 'settings.navThinking' },
  { id: 'endpoint', labelKey: 'settings.navEndpoint' },
  { id: 'telegram', labelKey: 'settings.navTelegram' },
  { id: 'proxy', labelKey: 'settings.navProxy' },
  { id: 'prompt-filter', labelKey: 'settings.navPromptFilter' },
  { id: 'api-endpoints', labelKey: 'tabs.api' },
  { id: 'password', labelKey: 'settings.navPassword' },
  { id: 'danger', labelKey: 'settings.navDanger' },
]

export function SettingsNav() {
  const { t } = useTranslation()
  const [active, setActive] = useState(SETTINGS_SECTIONS[0].id)

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        if (visible[0]) setActive(visible[0].target.id)
      },
      { rootMargin: '-80px 0px -60% 0px', threshold: 0 },
    )
    for (const s of SETTINGS_SECTIONS) {
      const el = document.getElementById(s.id)
      if (el) observer.observe(el)
    }
    return () => observer.disconnect()
  }, [])

  return (
    <nav className="lg:sticky lg:top-4 lg:h-fit lg:w-52 lg:shrink-0">
      <ul className="flex gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
        {SETTINGS_SECTIONS.map((s) => (
          <li key={s.id}>
            <a
              href={`#${s.id}`}
              onClick={(e) => {
                e.preventDefault()
                document.getElementById(s.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
              }}
              className={cn(
                'block whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                active === s.id
                  ? 'bg-secondary text-secondary-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              )}
            >
              {t(s.labelKey)}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
