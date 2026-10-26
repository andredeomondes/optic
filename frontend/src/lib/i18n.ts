// UI strings for pt-BR (source of truth) and en.
// Key convention: dot-separated, lowercase area first, camelCase leaf:
//   'settings.tabs.window', 'remote.power', 'lang.auto'.
// Variables use {name}; both languages must use the same set per key.
// Add a key to ptBR and en together; the test fails if they drift.

export type Lang = 'pt-BR' | 'en'
export type LangPref = 'auto' | Lang

export const ptBR = {
  'lang.auto': 'Automático (sistema)',
  'lang.pt': 'Português',
  'lang.en': 'English',
  'settings.language': 'Idioma',
} as const

export type Key = keyof typeof ptBR

export const en: Record<Key, string> = {
  'lang.auto': 'Automatic (system)',
  'lang.pt': 'Português',
  'lang.en': 'English',
  'settings.language': 'Language',
}

/** First preferred language decides: "pt*" is pt-BR, anything else is en. */
export function detectLang(languages: readonly string[]): Lang {
  const first = (languages[0] || '').toLowerCase()
  return first.startsWith('pt') ? 'pt-BR' : 'en'
}

export function resolveLang(pref: LangPref, languages: readonly string[]): Lang {
  return pref === 'auto' ? detectLang(languages) : pref
}

export function translate(lang: Lang, key: Key, vars?: Record<string, string | number>): string {
  const text = lang === 'en' ? en[key] : ptBR[key]
  if (!vars) return text
  return text.replace(/\{(\w+)\}/g, (m, name: string) => (name in vars ? String(vars[name]) : m))
}

/** Sorted unique variable names in a string. */
export function placeholders(s: string): string[] {
  return [...new Set([...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort()
}
