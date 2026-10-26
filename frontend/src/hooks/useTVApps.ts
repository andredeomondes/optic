import { useCallback, useEffect, useRef, useState } from 'react'
import * as api from '../lib/backend'
import type { TVApp } from '../lib/types'

const cache = new Map<string, TVApp[]>()
const listeners = new Set<(host: string, list: TVApp[]) => void>()

/** Stores a host's list and tells every mounted useTVApps. */
function publish(host: string, list: TVApp[]) {
  cache.set(host, list)
  listeners.forEach((fn) => fn(host, list))
}

/** Apps installed on the connected TV, cached per host and shared by all users of the hook. */
export function useTVApps(host: string | null, enabled: boolean) {
  const [apps, setApps] = useState<TVApp[]>(() => (host && enabled ? (cache.get(host) ?? []) : []))
  const [loading, setLoading] = useState(!!host && enabled && !cache.has(host))
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
          publish(host, list ?? [])
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

  // Another instance loaded this host: take its list.
  useEffect(() => {
    if (!host || !enabled) return
    const fn = (h: string, list: TVApp[]) => {
      if (h !== host) return
      setApps(list)
      setLoading(false)
      setError('')
    }
    listeners.add(fn)
    return () => {
      listeners.delete(fn)
    }
  }, [host, enabled])

  const reload = useCallback(() => load(true), [load])
  return { apps, loading, error, reload }
}
