import { useCallback, useEffect, useRef, useState } from 'react'
import type { Release, UpdateSupport } from '../../wailsjs/go/main/App'
import * as api from '../lib/backend'
import { load, save } from '../lib/storage'
import { isUpdateBusy, lastResultNotice, type UpdateKindName, type UpdateStateName } from '../lib/updates'
import { isNewer, RELEASES_URL } from '../lib/version'
import { useI18n } from './useI18n'
import { useRemote } from './useRemote'

const KEY = 'optic.updates.v1'
const DAY = 24 * 60 * 60 * 1000

interface Prefs {
  auto: boolean
  skipped: string | null
  lastChecked: number | null
}

/** Where the in-app update is; fed by the update:state and update:progress events. */
export interface UpdateRun {
  state: UpdateStateName
  kind: UpdateKindName
  /** pt-BR text from Go (translate before showing). */
  error: string
  got: number
  total: number
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
  /** Whether "Atualizar agora" can install the release here (null until known). */
  support: UpdateSupport | null
  run: UpdateRun
  checkNow: () => Promise<void>
  skip: (tag: string) => void
  setAuto: (on: boolean) => void
  startUpdate: () => void
}

const IDLE: UpdateRun = { state: 'idle', kind: '', error: '', got: 0, total: 0 }

// The previous update's outcome is shown once per app start, even if the hook remounts.
let resultAsked = false

/** Looks up the latest GitHub release (once a day while open), remembers the user's choices and runs the in-app update. */
export function useUpdates(): Updates {
  const { notify } = useRemote()
  const { t } = useI18n()
  const tRef = useRef(t)
  tRef.current = t
  const [prefs, setPrefs] = useState<Prefs>(() => ({ auto: true, skipped: null, lastChecked: null, ...load<Partial<Prefs>>(KEY, {}) }))
  const [release, setRelease] = useState<Release | null>(null)
  const [checking, setChecking] = useState(false)
  const [support, setSupport] = useState<UpdateSupport | null>(null)
  const [run, setRun] = useState<UpdateRun>(IDLE)
  const runRef = useRef(run)
  runRef.current = run
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

  const available = release !== null && isNewer(__APP_VERSION__, release.tag)

  // Ask Go whether this release can be installed from here.
  const tag = available ? release.tag : ''
  useEffect(() => {
    if (!tag) return
    let live = true
    api
      .UpdateSupport()
      .then((s) => live && setSupport(s))
      .catch(() => live && setSupport(null))
    return () => {
      live = false
    }
  }, [tag])

  useEffect(() => {
    const offProgress = api.onEvent('update:progress', (p: { got: number; total: number }) =>
      setRun((r) => ({ ...r, got: p.got, total: p.total })),
    )
    const offState = api.onEvent('update:state', (s: { state: UpdateStateName; kind: UpdateKindName; error?: string }) =>
      setRun((r) => ({ ...r, state: s.state, kind: s.kind, error: s.error ?? '' })),
    )
    return () => {
      offProgress()
      offState()
    }
  }, [])

  // Tell the user once how the last update went.
  useEffect(() => {
    if (resultAsked) return
    resultAsked = true
    api
      .LastUpdateResult()
      .then((r) => {
        const n = lastResultNotice(r)
        if (!n) return
        const text = tRef.current(n.key, n.vars)
        if (n.kind === 'ok') notify('ok', text)
        else notify('err', text, { label: tRef.current('update.openReleases'), run: () => api.BrowserOpenURL(RELEASES_URL) })
      })
      .catch(() => {})
  }, [notify])

  const startUpdate = useCallback(() => {
    if (isUpdateBusy(runRef.current.state)) return
    setRun({ ...IDLE, state: 'downloading', kind: support?.kind ?? '' })
    api.StartUpdate().catch((e) => setRun((r) => ({ ...r, state: 'error', error: api.errText(e) })))
  }, [support])

  return {
    release,
    available,
    skipped: release !== null && prefs.skipped === release.tag,
    checking,
    auto: prefs.auto,
    lastChecked: prefs.lastChecked,
    support,
    run,
    checkNow,
    skip: (tag) => update({ skipped: tag }),
    setAuto: (on) => update({ auto: on }),
    startUpdate,
  }
}
