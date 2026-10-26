import { useCallback, useEffect, useRef, useState } from 'react'
import * as api from '../lib/backend'
import type { TVApp } from '../lib/types'

const cache = new Map<string, TVApp[]>()

/** Apps installed on the connected TV, cached per host. */
export function useTVApps(host: string | null, enabled: boolean) {
  const [apps, setApps] = useState<TVApp[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const seq = useRef(0)

  const load = useCallback(
    (force: boolean) => {
      const n = ++seq.current
      if (!host || !enabled) {
        setApps([])
        setLoading(false)
        setError('')
        return
      }
      const hit = force ? undefined : cache.get(host)
      if (hit) {
        setApps(hit)
        setLoading(false)
        setError('')
        return
      }
      setApps([])
      setLoading(true)
      setError('')
      api
        .InstalledApps()
        .then((list) => {
          if (n !== seq.current) return // host changed meanwhile
          const next = list ?? []
          cache.set(host, next)
          setApps(next)
          setLoading(false)
        })
        .catch((e) => {
          if (n !== seq.current) return
          setError(api.errText(e))
          setLoading(false)
        })
    },
    [host, enabled],
  )

  useEffect(() => {
    load(false)
    return () => {
      seq.current++ // drop any pending response on unmount or change
    }
  }, [load])

  const reload = useCallback(() => load(true), [load])
  return { apps, loading, error, reload }
}
