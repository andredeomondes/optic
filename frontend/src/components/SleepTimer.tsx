import { Moon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useI18n } from '../hooks/useI18n'
import { useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { useSleepTimer } from '../hooks/useSleepTimer'
import { remaining, SLEEP_PRESETS } from '../lib/sleep'

/** Moon button with a preset menu; shows "Turns off in N min" while a timer is armed. */
export function SleepTimer() {
  const { connected } = useRemote()
  const { deadline, start, cancel } = useSleepTimer()
  const { t } = useI18n()
  useNow(60_000) // re-render each minute; the value itself is read fresh below
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  const moon = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false)
    }
    root.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        moon.current?.focus()
        return
      }
      if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])
      if (!items.length) return
      e.preventDefault()
      const i = items.indexOf(document.activeElement as HTMLElement)
      const step = e.key === 'ArrowDown' ? 1 : -1
      items[(i + step + items.length) % items.length].focus()
    }
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
      {deadline !== null && <span className="chip sleep-chip">{t('sleep.chip', { n: remaining(deadline, Date.now()) })}</span>}
      <button
        ref={moon}
        className={`btn sm ghost ${deadline !== null ? 'active' : ''}`}
        onClick={() => setOpen((o) => !o)}
        aria-label={t('sleep.aria')}
        aria-expanded={open}
        title={t('sleep.title')}
        disabled={!connected && deadline === null}
      >
        <Moon size={16} strokeWidth={1.8} />
      </button>
      {open && (
        <div className="sleep-menu" role="menu">
          {SLEEP_PRESETS.map((m) => (
            <button key={m} role="menuitem" onClick={() => pick(() => start(m))}>
              {t('sleep.preset', { n: m })}
            </button>
          ))}
          {deadline !== null && (
            <button role="menuitem" className="danger" onClick={() => pick(cancel)}>
              {t('sleep.cancel')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
