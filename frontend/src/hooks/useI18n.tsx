// Language preference + translate(). The pref ('auto' | 'pt-BR' | 'en') is
// stored locally; 'auto' follows the system language.
import { createContext, ReactNode, useContext, useEffect, useMemo, useState } from 'react'
import * as api from '../lib/backend'
import { type Key, type Lang, type LangPref, resolveLang, translate } from '../lib/i18n'
import { load, save } from '../lib/storage'

const LANG_KEY = 'optic.lang.v1'

type I18n = {
  lang: Lang
  pref: LangPref
  setPref: (p: LangPref) => void
  t: (key: Key, vars?: Record<string, string | number>) => string
}

const Ctx = createContext<I18n | null>(null)

const validPref = (p: unknown): p is LangPref => p === 'auto' || p === 'pt-BR' || p === 'en'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<LangPref>(() => {
    const p = load<unknown>(LANG_KEY, 'auto')
    return validPref(p) ? p : 'auto'
  })
  const lang = resolveLang(pref, navigator.languages?.length ? navigator.languages : [navigator.language || ''])

  useEffect(() => {
    document.documentElement.lang = lang
    // Binding arrives with the backend language support; skip until then.
    const setLanguage = (api as any).SetLanguage as ((l: string) => Promise<unknown>) | undefined
    if (typeof setLanguage === 'function') void Promise.resolve(setLanguage(lang)).catch(() => {})
  }, [lang])

  const value = useMemo<I18n>(
    () => ({
      lang,
      pref,
      setPref: (p) => {
        setPrefState(p)
        save(LANG_KEY, p)
      },
      t: (key, vars) => translate(lang, key, vars),
    }),
    [lang, pref],
  )
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useI18n(): I18n {
  const v = useContext(Ctx)
  if (!v) throw new Error('useI18n outside I18nProvider')
  return v
}
