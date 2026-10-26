import { useCallback, useEffect, useState } from 'react'
import { Keyboard, RotateCcw } from 'lucide-react'
import * as api from '../lib/backend'
import { accelFromEvent, labelFor } from '../lib/accel'

type State = Awaited<ReturnType<typeof api.GetHotkeys>>

const ACTIONS: { id: string; label: string; def: string }[] = [
  { id: 'volume_up', label: 'Volume +', def: 'Ctrl+Alt+Up' },
  { id: 'volume_down', label: 'Volume −', def: 'Ctrl+Alt+Down' },
  { id: 'mute', label: 'Mudo', def: 'Ctrl+Alt+M' },
  { id: 'home', label: 'Início', def: 'Ctrl+Alt+H' },
  { id: 'palette', label: 'Comandos rápidos', def: 'Ctrl+Alt+K' },
]

const msg = (e: unknown) => (e instanceof Error ? e.message : String(e))

/** Global (system-wide) shortcuts: toggles, rebinding by key capture, per-binding errors. */
export function HotkeySettings() {
  const [st, setSt] = useState<State | null>(null)
  const [capturing, setCapturing] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState<{ id: string; text: string } | null>(null)

  useEffect(() => {
    if (!api.hasRuntime()) return
    api.GetHotkeys().then(setSt).catch(() => {})
  }, [])

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

  if (!st) return <p className="note">Atalhos globais indisponíveis neste sistema</p>

  const off = !st.supported || busy
  const accelOf = (id: string) => st.bindings.find((b) => b.id === id)?.accel ?? ACTIONS.find((a) => a.id === id)!.def
  const errorOf = (id: string) => (failure?.id === id ? failure.text : st.bindings.find((b) => b.id === id)?.error || '')
  const defaults = Object.fromEntries(ACTIONS.map((a) => [a.id, a.def]))

  return (
    <div className="hotkeys" data-capturing={capturing ?? undefined}>
      {!st.supported && <p className="err">Atalhos globais indisponíveis neste sistema</p>}
      <ul className="shortcut-list">
        <li>
          <label className="pref" htmlFor="hk-enabled">
            <span>Atalhos globais (funcionam com outra janela em foco)</span>
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
            <span>Teclas de mídia do teclado controlam a TV</span>
            <input
              id="hk-media"
              type="checkbox"
              className="switch"
              checked={st.media}
              disabled={off}
              onChange={(e) => save(st.enabled, e.target.checked, current())}
            />
          </label>
          <span className="note">Enquanto ligado, o player de música do PC não recebe essas teclas</span>
        </li>
        {ACTIONS.map((a) => {
          const err = errorOf(a.id)
          const isCap = capturing === a.id
          return (
            <li key={a.id} className="stack">
              <div className="hk-row">
                <span>{a.label}</span>
                <span className="hk-ctl">
                  {isCap ? (
                    <span className="chip hk-capture">Pressione o atalho…</span>
                  ) : (
                    <kbd className="hk-accel">{labelFor(accelOf(a.id))}</kbd>
                  )}
                  <button className="link" disabled={off} onClick={() => (setFailure(null), setCapturing(isCap ? null : a.id))}>
                    <Keyboard size={12} /> {isCap ? 'Cancelar' : 'Alterar'}
                  </button>
                </span>
              </div>
              {err && <span className="err" role="alert">{err}</span>}
            </li>
          )
        })}
        <li className="note">
          <button className="link" disabled={off} onClick={() => (setCapturing(null), save(st.enabled, st.media, defaults))}>
            <RotateCcw size={12} /> Restaurar padrão
          </button>
        </li>
        {failure && failure.id === '' && (
          <li className="note">
            <span className="err" role="alert">{failure.text}</span>
          </li>
        )}
      </ul>
    </div>
  )
}
