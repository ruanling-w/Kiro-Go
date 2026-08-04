// uiStore — client-only UI state: theme (3-state), language, list filters,
// selection, and privacy mode. None of this touches the server; it persists to
// localStorage where it should survive reloads (theme, lang, privacy) and stays
// ephemeral where it should not (filters, selection).
//
// Theme is a 3-state preference (system/light/dark). `system` resolves against
// the OS media query at apply-time; the resolver toggles the `.dark` class on
// <html> (matching the @custom-variant in index.css) and is re-run on OS changes
// while the preference is `system`.

import { create } from 'zustand'
import i18n, { type Lang, LANGS, DEFAULT_LANG } from '@/lib/i18n'

export type ThemePref = 'system' | 'light' | 'dark'

const THEME_KEY = 'kiro_theme'
const LANG_KEY = 'kiro_lang'
const PRIVACY_KEY = 'kiro_privacy'
const DOCS_BASE_KEY = 'kiro_docs_base'

const prefersDark = () =>
  typeof window !== 'undefined' &&
  window.matchMedia('(prefers-color-scheme: dark)').matches

function resolveDark(pref: ThemePref): boolean {
  return pref === 'dark' || (pref === 'system' && prefersDark())
}

function applyTheme(pref: ThemePref) {
  const root = document.documentElement
  root.classList.toggle('dark', resolveDark(pref))
  root.dataset.themePref = pref
}

function initialTheme(): ThemePref {
  const v = localStorage.getItem(THEME_KEY)
  return v === 'light' || v === 'dark' || v === 'system' ? v : 'system'
}

function initialLang(): Lang {
  const v = localStorage.getItem(LANG_KEY)
  return v && (LANGS as readonly string[]).includes(v) ? (v as Lang) : DEFAULT_LANG
}

export type AccountStatusFilter = 'all' | 'enabled' | 'disabled' | 'banned'
export type ApiKeyStatusFilter = 'all' | 'active' | 'expired' | 'disabled'

interface UiState {
  theme: ThemePref
  lang: Lang
  privacyMode: boolean

  // Accounts list filters.
  accountKeyword: string
  accountStatus: AccountStatusFilter
  providerFilter: string
  selectedAccounts: Set<string>

  // API keys list filters.
  apiKeyKeyword: string
  apiKeyStatus: ApiKeyStatusFilter

  // API docs page — base URL persists; selected key id is ephemeral.
  // Empty string = not chosen yet; ConnectionCard fills via resolveGatewayBaseURL.
  docsBaseURL: string
  docsApiKeyId: string | null

  setTheme: (t: ThemePref) => void
  cycleTheme: () => void
  setLang: (l: Lang) => void
  togglePrivacy: () => void

  setAccountKeyword: (v: string) => void
  setAccountStatus: (v: AccountStatusFilter) => void
  setProviderFilter: (v: string) => void
  toggleAccountSelected: (id: string) => void
  setAccountSelection: (ids: string[]) => void
  clearAccountSelection: () => void

  setApiKeyKeyword: (v: string) => void
  setApiKeyStatus: (v: ApiKeyStatusFilter) => void

  setDocsBaseURL: (v: string) => void
  setDocsApiKeyId: (v: string | null) => void
}

const THEME_ORDER: ThemePref[] = ['system', 'light', 'dark']

export const useUiStore = create<UiState>((set, get) => ({
  theme: initialTheme(),
  lang: initialLang(),
  privacyMode: localStorage.getItem(PRIVACY_KEY) !== '0',

  accountKeyword: '',
  accountStatus: 'all',
  providerFilter: '',
  selectedAccounts: new Set(),

  apiKeyKeyword: '',
  apiKeyStatus: 'all',

  docsBaseURL: (typeof window !== 'undefined' && localStorage.getItem(DOCS_BASE_KEY)) || '',
  docsApiKeyId: null,

  setTheme: (t) => {
    localStorage.setItem(THEME_KEY, t)
    applyTheme(t)
    set({ theme: t })
  },
  cycleTheme: () => {
    const next = THEME_ORDER[(THEME_ORDER.indexOf(get().theme) + 1) % THEME_ORDER.length]
    get().setTheme(next)
  },
  setLang: (l) => {
    localStorage.setItem(LANG_KEY, l)
    void i18n.changeLanguage(l)
    document.documentElement.lang = l
    set({ lang: l })
  },
  togglePrivacy: () => {
    const next = !get().privacyMode
    localStorage.setItem(PRIVACY_KEY, next ? '1' : '0')
    set({ privacyMode: next })
  },

  setAccountKeyword: (v) => set({ accountKeyword: v }),
  setAccountStatus: (v) => set({ accountStatus: v }),
  setProviderFilter: (v) => set({ providerFilter: v }),
  toggleAccountSelected: (id) =>
    set((s) => {
      const next = new Set(s.selectedAccounts)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return { selectedAccounts: next }
    }),
  setAccountSelection: (ids) => set({ selectedAccounts: new Set(ids) }),
  clearAccountSelection: () => set({ selectedAccounts: new Set() }),

  setApiKeyKeyword: (v) => set({ apiKeyKeyword: v }),
  setApiKeyStatus: (v) => set({ apiKeyStatus: v }),

  setDocsBaseURL: (v) => {
    localStorage.setItem(DOCS_BASE_KEY, v)
    set({ docsBaseURL: v })
  },
  setDocsApiKeyId: (v) => set({ docsApiKeyId: v }),
}))

/**
 * Initializes theme on boot and keeps `system` in sync with the OS. Call once
 * from main.tsx before render. Returns a cleanup for the media listener.
 */
export function initTheme(): () => void {
  applyTheme(useUiStore.getState().theme)
  document.documentElement.lang = useUiStore.getState().lang

  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  const onChange = () => {
    if (useUiStore.getState().theme === 'system') applyTheme('system')
  }
  mq.addEventListener('change', onChange)
  return () => mq.removeEventListener('change', onChange)
}
