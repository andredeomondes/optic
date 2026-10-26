import { useCallback, useEffect, useState } from 'react'
import { Keyboard, RotateCcw } from 'lucide-react'
import * as api from '../lib/backend'
import { useI18n } from '../hooks/useI18n'
import { accelFromEvent, labelFor } from '../lib/accel'
import { translateError } from '../lib/errors'
import type { Key } from '../lib/i18n'

type State = Awaited<ReturnType<typeof api.GetHotkeys>>

// Defaults mirror filestore.DefaultPrefs (Go); keep both in sync.
const ACTIONS: { id: string; label: Key; def: string }[] = [
  { id: 'volume_up', label: 'hotkeys.volUp', def: 'Ctrl+Alt+Up' },
  { id: 'volume_down', label: 'hotkeys.volDown', def: 'Ctrl+Alt+Down' },
  { id: 'mute', label: 'hotkeys.mute', def: 'Ctrl+Alt+M' },
  { id: 'home', label: 'hotkeys.home', def: 'Ctrl+Alt+H' },
  { id: 'palette', label: 'hotkeys.palette', def: 'Ctrl+Alt+K' },
]

const MEDIA_LABELS: Record<string, Key> = {
  media_play_pause: 'hotkeys.mediaPlayPause',
  media_next: 'hotkeys.mediaNext',
  media_prev: 'hotkeys.mediaPrev',
  media_stop: 'hotkeys.mediaStop',
}

const msg = (e: unknown) => (e instanceof Error ? e.message : String(e))

/** Global (system-wide) shortcuts: toggles, rebinding by key capture, per-binding errors. */
export function HotkeySettings() {
  const { t, lang } = useI18n()
  const te = (raw: string) => translateError(raw, lang)
  const [st, setSt] = useState<State | null>(null)
  const [capturing, setCapturing] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState<{ id: string; text: string } | null>(null)
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    if (!api.hasRuntime()) return
    api.GetHotkeys().then(setSt).catch((e) => setLoadError(msg(e)))
  }, [])

  // While capturing, drop the registered hotkeys so the pressed keys do not
  // reach the TV. Restored on finish, cancel and unmount.
  const isCapturing = capturing !== null
  useEffect(() => {
    if (!isCapturing || !api.hasRuntime()) return
    void api.SuspendHotkeys(true).catch(() => {})
    return () => {
      void api.SuspendHotkeys(false).catch(() => {})
    }
  }, [isCapturing])

  const current = useCallback(
    () => Object.fromEntries(ACTIONS.map((a) => [a.id, st?.bindings.find((b) => b.id === a.id)?.accel ?? a.def])),
    [st],
  )

  const save = useCallback(
    async (enabled: boolean, media: boolean, bindings: Record<string, string>, failId = '') => {
      setBusy(true)
      setFailure(null)
      try {
        setSt(await api.SetHotkeys(enabled, media, bindings))
      } catch (e) {
        setFailure({ id: failId, text: msg(e) })
      } finally {
        setBusy(false)
      }
    },
    [],
  )

  // Capture mode: swallow every key so nothing reaches the TV or the dialog; Esc cancels.
  useEffect(() => {
    if (!capturing || !st) return
    const id = capturing
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        setCapturing(null)
        return
      }
      const accel = accelFromEvent(e)
      if (!accel) return // modifier alone: keep waiting
      setCapturing(null)
      void save(st.enabled, st.media, { ...current(), [id]: accel }, id)
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [capturing, st, current, save])

  if (loadError) return <p className="err" role="alert">{t('hotkeys.loadFailed', { error: te(loadError) })}</p>
  if (!st) return api.hasRuntime() ? <p className="note">{t('hotkeys.loading')}</p> : <p className="note">{t('hotkeys.unsupported')}</p>

  const off = !st.supported || busy || isCapturing
  const accelOf = (id: string) => st.bindings.find((b) => b.id === id)?.accel ?? ACTIONS.find((a) => a.id === id)!.def
  const errorOf = (id: string) => te(failure?.id === id ? failure.text : st.bindings.find((b) => b.id === id)?.error || '')
  const defaults = Object.fromEntries(ACTIONS.map((a) => [a.id, a.def]))

  return (
    <div className="hotkeys" data-capturing={capturing ?? undefined}>
      {!st.supported && <p className="err">{t('hotkeys.unsupported')}</p>}
      <ul className="shortcut-list">
        <li className="stack">
          <label className="pref" htmlFor="hk-enabled">
            <span>{t('hotkeys.enabled')}</span>
            <input
              id="hk-enabled"
              type="checkbox"
              className="switch"
              checked={st.enabled}
              disabled={off}
              onChange={(e) => save(e.target.checked, st.media, current())}
            />
          </label>
        </li>
        <li className="stack">
          <label className="pref" htmlFor="hk-media">
            <span>{t('hotkeys.media')}</span>
            <input
              id="hk-media"
              type="checkbox"
              className="switch"
              checked={st.media}
              disabled={off}
              onChange={(e) => save(st.enabled, e.target.checked, current())}
            />
          </label>
          <span className="note">{t('hotkeys.mediaNote')}</span>
          {st.bindings
            .filter((b) => b.id in MEDIA_LABELS && b.error)
            .map((b) => (
              <span key={b.id} className="err" role="alert">
                {t(MEDIA_LABELS[b.id])}: {te(b.error)}
              </span>
            ))}
        </li>
        {ACTIONS.map((a) => {
          const err = errorOf(a.id)
          const isCap = capturing === a.id
          return (
            <li key={a.id} className="stack">
              <div className="hk-row">
                <span>{t(a.label)}</span>
                <span className="hk-ctl">
                  {isCap ? (
                    <span className="chip hk-capture">{t('hotkeys.press')}</span>
                  ) : (
                    <kbd className="hk-accel">{labelFor(accelOf(a.id))}</kbd>
                  )}
                  <button className="link" disabled={!st.supported || busy} onClick={() => (setFailure(null), setCapturing(isCap ? null : a.id))}>
                    <Keyboard size={12} /> {isCap ? t('common.cancel') : t('hotkeys.change')}
                  </button>
                </span>
              </div>
              {err && <span className="err" role="alert">{err}</span>}
            </li>
          )
        })}
        <li className="note">
          {t('hotkeys.altgr')}
        </li>
        <li className="note">
          <button className="link" disabled={off} onClick={() => (setCapturing(null), save(st.enabled, st.media, defaults))}>
            <RotateCcw size={12} /> {t('hotkeys.reset')}
          </button>
        </li>
        {failure && failure.id === '' && (
          <li className="note">
            <span className="err" role="alert">{te(failure.text)}</span>
          </li>
        )}
      </ul>
    </div>
  )
}
