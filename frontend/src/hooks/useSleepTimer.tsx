import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { decideAtDeadline } from '../lib/sleep'
import { useRemote } from './useRemote'

export type SleepTimer = {
  /** Epoch ms when the TV will be turned off, or null when no timer is armed. */
  deadline: number | null
  start: (minutes: number) => void
  cancel: () => void
}

const Ctx = createContext<SleepTimer | null>(null)

/** Single shared timer: one setTimeout to the deadline, in memory only. */
export function SleepTimerProvider({ children }: { children: ReactNode }) {
  const { connected, tvState, press, notify } = useRemote()
  const [deadline, setDeadline] = useState<number | null>(null)
  const timer = useRef<number | undefined>(undefined)

  // The timeout fires long after it was armed: read the freshest state then.
  const latest = useRef({ connected, tvOn: tvState?.on === true, press, notify })
  latest.current = { connected, tvOn: tvState?.on === true, press, notify }

  const clear = useCallback(() => {
    window.clearTimeout(timer.current)
    timer.current = undefined
  }, [])

  const cancel = useCallback(() => {
    clear()
    setDeadline(null)
  }, [clear])

  const start = useCallback(
    (minutes: number) => {
      clear()
      const at = Date.now() + minutes * 60_000
      setDeadline(at)
      timer.current = window.setTimeout(() => {
        timer.current = undefined
        setDeadline(null)
        const s = latest.current
        if (decideAtDeadline(s) === 'power') {
          s.press('POWER').then((ok) => ok && s.notify('ok', 'TV desligada pelo timer'))
        } else {
          s.notify('info', 'Timer cancelado: a TV já estava desligada ou desconectada')
        }
      }, minutes * 60_000)
    },
    [clear],
  )

  useEffect(() => clear, [clear])

  const value = useMemo(() => ({ deadline, start, cancel }), [deadline, start, cancel])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useSleepTimer(): SleepTimer {
  const v = useContext(Ctx)
  if (!v) throw new Error('useSleepTimer outside SleepTimerProvider')
  return v
}
