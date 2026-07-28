// i18n setup for the admin SPA.
//
// The locale files ship flat, dot-separated keys ("login.title") with positional
// {0}/{1} placeholders — NOT the nested keys + {{named}} interpolation that
// react-i18next assumes by default. So:
//   - keySeparator: false  → "login.title" is one literal key, not login→title.
//   - nsSeparator: false   → ":" in a key is not treated as a namespace split.
//   - a custom post-processor rewrites {0},{1},... from the values passed as
//     t(key, { 0: 'x', 1: 'y' }) or the t(key, a, b) helper in ./t.ts.
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from '@/../locales/en.json'
import vi from '@/../locales/vi.json'
import zh from '@/../locales/zh.json'

export const LANGS = ['vi', 'en', 'zh'] as const
export type Lang = (typeof LANGS)[number]

export const DEFAULT_LANG: Lang = 'vi'
export const FALLBACK_LANG: Lang = 'zh'

function initialLang(): Lang {
  const stored = localStorage.getItem('kiro_lang')
  if (stored && (LANGS as readonly string[]).includes(stored)) return stored as Lang
  return DEFAULT_LANG
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    vi: { translation: vi },
    zh: { translation: zh },
  },
  lng: initialLang(),
  fallbackLng: FALLBACK_LANG,
  keySeparator: false,
  nsSeparator: false,
  interpolation: {
    escapeValue: false, // React already escapes.
    // Locale placeholders are {0},{1},... not {{named}}.
    prefix: '{',
    suffix: '}',
  },
  returnNull: false,
})

export default i18n
