import { useEffect, useState } from 'react'

/** Re-renders every `ms` so relative times ("há 12 min") stay fresh. */
export function useNow(ms = 30_000) {
  const [now, setNow] = useState(Date.now)
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), ms)
    return () => window.clearInterval(t)
  }, [ms])
  return now
}

export function since(from: number, now: number) {
  const min = Math.floor((now - from) / 60_000)
  if (min < 1) return 'agora'
  if (min < 60) return `há ${min} min`
  const h = Math.floor(min / 60)
  return `há ${h} h ${min % 60} min`
}
