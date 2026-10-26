import { useEffect, useState } from 'react'
import type { Key } from '../lib/i18n'

type T = (key: Key, vars?: Record<string, string | number>) => string

/** Re-renders every `ms` so relative times ("há 12 min") stay fresh. */
export function useNow(ms = 30_000) {
  const [now, setNow] = useState(Date.now)
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), ms)
    return () => window.clearInterval(t)
  }, [ms])
  return now
}

export function since(from: number, now: number, t: T) {
  const min = Math.floor((now - from) / 60_000)
  if (min < 1) return t('time.now')
  if (min < 60) return t('time.minAgo', { n: min })
  const h = Math.floor(min / 60)
  return t('time.hourMinAgo', { h, m: min % 60 })
}
