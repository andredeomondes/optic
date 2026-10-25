import { Moon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { useSleepTimer } from '../hooks/useSleepTimer'
import { remaining, SLEEP_PRESETS } from '../lib/sleep'

/** Moon button with a preset menu; shows "Desliga em N min" while a timer is armed. */
export function SleepTimer() {
  const { connected } = useRemote()
  const { deadline, start, cancel } = useSleepTimer()
  useNow(60_000) // re-render each minute; the value itself is read fresh below
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    window.addEventListener('mousedown', onDown)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('mousedown', onDown)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  const pick = (fn: () => void) => {
    fn()
    setOpen(false)
  }

  return (
    <div className="sleep" ref={root}>
      {deadline !== null && <span className="chip sleep-chip">Desliga em {remaining(deadline, Date.now())} min</span>}
      <button
        className={`btn sm ghost ${deadline !== null ? 'active' : ''}`}
        onClick={() => setOpen((o) => !o)}
        aria-label="Timer para desligar"
        aria-expanded={open}
        title="Desligar a TV depois de um tempo"
        disabled={!connected && deadline === null}
      >
        <Moon size={16} strokeWidth={1.8} />
      </button>
      {open && (
        <div className="sleep-menu" role="menu">
          {SLEEP_PRESETS.map((m) => (
            <button key={m} role="menuitem" onClick={() => pick(() => start(m))}>
              {m} min
            </button>
          ))}
          {deadline !== null && (
            <button role="menuitem" className="danger" onClick={() => pick(cancel)}>
              Cancelar timer
            </button>
          )}
        </div>
      )}
    </div>
  )
}
