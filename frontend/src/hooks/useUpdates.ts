import { useCallback, useEffect, useRef, useState } from 'react'
import type { Release } from '../../wailsjs/go/main/App'
import * as api from '../lib/backend'
import { load, save } from '../lib/storage'
import { isNewer } from '../lib/version'
import { useI18n } from './useI18n'
import { useRemote } from './useRemote'

const KEY = 'optic.updates.v1'
const DAY = 24 * 60 * 60 * 1000

interface Prefs {
  auto: boolean
  skipped: string | null
  lastChecked: number | null
}

export interface Updates {
  release: Release | null
  /** A newer stable release exists (it may still be skipped). */
  available: boolean
  /** The user dismissed this release's tag. */
  skipped: boolean
  checking: boolean
  auto: boolean
  lastChecked: number | null
  checkNow: () => Promise<void>
  skip: (tag: string) => void
  setAuto: (on: boolean) => void
}

/** Looks up the latest GitHub release (once a day while open) and remembers the user's choices. */
export function useUpdates(): Updates {
  const { notify } = useRemote()
  const { t } = useI18n()
  const tRef = useRef(t)
  tRef.current = t
  const [prefs, setPrefs] = useState<Prefs>(() => ({ auto: true, skipped: null, lastChecked: null, ...load<Partial<Prefs>>(KEY, {}) }))
  const [release, setRelease] = useState<Release | null>(null)
  const [checking, setChecking] = useState(false)
  const prefsRef = useRef(prefs)

  const update = useCallback((patch: Partial<Prefs>) => {
    const next = { ...prefsRef.current, ...patch }
    prefsRef.current = next
    save(KEY, next)
    setPrefs(next)
  }, [])

  // One lookup at a time: StrictMode double effects and overlapping manual/auto checks share a request.
  const inflight = useRef<Promise<boolean> | null>(null)

  const check = useCallback(
    async (manual: boolean) => {
      if (!inflight.current) {
        setChecking(true)
        inflight.current = api
          .LatestRelease()
          .then((r) => {
            setRelease(r)
            update({ lastChecked: Date.now() })
            return true
          })
          .catch(() => false)
          .finally(() => {
            inflight.current = null
            setChecking(false)
          })
      }
      const ok = await inflight.current
      if (!ok && manual) notify('err', tRef.current('update.checkFailed'))
    },
    [notify, update],
  )

  const checkNow = useCallback(() => check(true), [check])

  useEffect(() => {
    if (!prefs.auto) return
    const last = prefsRef.current.lastChecked
    if (!last || Date.now() - last > DAY) void check(false)
    const id = setInterval(() => void check(false), DAY)
    return () => clearInterval(id)
  }, [prefs.auto, check])

  return {
    release,
    available: release !== null && isNewer(__APP_VERSION__, release.tag),
    skipped: release !== null && prefs.skipped === release.tag,
    checking,
    auto: prefs.auto,
    lastChecked: prefs.lastChecked,
    checkNow,
    skip: (tag) => update({ skipped: tag }),
    setAuto: (on) => update({ auto: on }),
  }
}
