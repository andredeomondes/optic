// All connection state and TV actions live here; components read them via
// useRemote(). Every action reports pending/ok/err feedback keyed by an id
// so buttons can animate the result.
import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import * as api from '../lib/backend'
import { load, save } from '../lib/storage'
import type { DeviceInfo, Feedback, LogLine, Status, Toast, TV, TVState } from '../lib/types'

const MANUAL_KEY = 'optic.manualHosts.v1'
const FEEDBACK_MS = 650

let seq = 0

function useRemoteState() {
  const [devices, setDevices] = useState<TV[]>([])
  const [manual, setManual] = useState<string[]>(() => load(MANUAL_KEY, []))
  const [scanning, setScanning] = useState(false)
  const [status, setStatus] = useState<Status>('idle')
  const [host, setHost] = useState('')
  const [info, setInfo] = useState<DeviceInfo | null>(null)
  const [tvState, setTvState] = useState<TVState | null>(null)
  const [error, setError] = useState('')
  const [connectedAt, setConnectedAt] = useState<number | null>(null)
  const [log, setLog] = useState<LogLine[]>([])
  const [feedback, setFeedback] = useState<Record<string, Feedback>>({})
  const [pressedKey, setPressedKey] = useState('')
  const [toast, setToast] = useState<Toast | null>(null)
  const toastTimer = useRef<number>()

  const connected = status === 'connected'

  const addLog = useCallback((dir: string, text: string) => {
    const time = new Date().toLocaleTimeString('pt-BR')
    setLog((l) => [...l.slice(-300), { id: ++seq, dir, text, time }])
  }, [])

  const notify = useCallback((kind: Toast['kind'], text: string) => {
    window.clearTimeout(toastTimer.current)
    setToast({ id: ++seq, kind, text })
    toastTimer.current = window.setTimeout(() => setToast(null), kind === 'err' ? 4000 : 2200)
  }, [])

  /** Runs an action with feedback under `id`; errors become a toast. */
  const run = useCallback(
    async (id: string, fn: () => Promise<unknown>, success?: string) => {
      setFeedback((f) => ({ ...f, [id]: 'pending' }))
      try {
        await fn()
        setFeedback((f) => ({ ...f, [id]: 'ok' }))
        if (success) notify('ok', success)
        return true
      } catch (e) {
        setFeedback((f) => ({ ...f, [id]: 'err' }))
        notify('err', api.errText(e))
        addLog('!', api.errText(e))
        return false
      } finally {
        window.setTimeout(
          () =>
            setFeedback((f) => {
              const { [id]: _, ...rest } = f
              return rest
            }),
          FEEDBACK_MS,
        )
      }
    },
    [notify, addLog],
  )

  const connect = useCallback(
    async (ip: string) => {
      setHost(ip)
      setError('')
      setStatus('connecting')
      try {
        const i = await api.Connect(ip)
        setInfo(i)
        setStatus('connected')
        setConnectedAt(Date.now())
        addLog('•', `conectado em ${ip}`)
      } catch (e) {
        const msg = api.errText(e)
        if (msg.includes('NOT_PAIRED')) {
          try {
            await api.StartPairing(ip)
            setStatus('pairing')
            addLog('•', 'TV mostrando código de pareamento')
          } catch (pe) {
            setError(api.errText(pe))
            setStatus('error')
          }
        } else {
          setError(msg)
          setStatus('error')
        }
      }
    },
    [addLog],
  )

  const scan = useCallback(async () => {
    setScanning(true)
    try {
      setDevices(await api.Discover())
    } catch (e) {
      notify('err', api.errText(e))
    } finally {
      setScanning(false)
    }
  }, [notify])

  const addManual = useCallback(
    (ip: string) => {
      setManual((m) => {
        const next = m.includes(ip) ? m : [...m, ip]
        save(MANUAL_KEY, next)
        return next
      })
      return connect(ip)
    },
    [connect],
  )

  const removeManual = useCallback((ip: string) => {
    setManual((m) => {
      const next = m.filter((x) => x !== ip)
      save(MANUAL_KEY, next)
      return next
    })
  }, [])

  const finishPairing = useCallback(
    async (code: string) => {
      await api.FinishPairing(code)
      addLog('•', 'pareado com sucesso')
      notify('ok', 'TV pareada')
      await connect(host)
    },
    [addLog, notify, connect, host],
  )

  const cancelPairing = useCallback(() => {
    api.CancelPairing()
    setStatus('idle')
  }, [])

  const press = useCallback(
    (key: string) => {
      setPressedKey(key)
      window.setTimeout(() => setPressedKey((p) => (p === key ? '' : p)), 160)
      if (!connected) return Promise.resolve(false)
      return run(`key:${key}`, () => api.SendKey(key))
    },
    [connected, run],
  )

  const launch = useCallback(
    (id: string, link: string, name: string) =>
      run(`app:${id}`, () => api.LaunchApp(link), `Abrindo ${name} na TV…`),
    [run],
  )

  const sendText = useCallback(
    (text: string) => run('text', () => api.SendText(text), 'Texto enviado para a TV'),
    [run],
  )

  const powerOn = useCallback(
    (ip: string) => run(`power:${ip}`, () => api.PowerOn(ip), 'Ligando a TV…'),
    [run],
  )

  const setVolume = useCallback(
    (level: number) => run('volume', () => api.SetVolume(level), `Volume ${level}`),
    [run],
  )

  useEffect(() => {
    const offs = [
      api.onEvent('state', (s: TVState) => setTvState(s)),
      api.onEvent('trace', (dir: string, text: string) => addLog(dir, text)),
      api.onEvent('disconnected', () => {
        setStatus('error')
        setError('Conexão com a TV caiu.')
        setConnectedAt(null)
        addLog('•', 'desconectado')
      }),
    ]
    if (api.hasRuntime()) {
      scan()
      api.LastHost().then((h) => {
        if (h) connect(h)
      })
    }
    return () => offs.forEach((off) => off())
  }, [scan, connect, addLog])

  // Discovered TVs first, then manually added IPs not seen on the network.
  const tvs = useMemo<TV[]>(() => {
    const extra = manual
      .filter((ip) => !devices.some((d) => d.ip === ip))
      .map((ip) => ({ name: ip, model: '', ip, remotev2: true, platform: '' as const, manual: true }))
    return [...devices, ...extra]
  }, [devices, manual])

  return {
    tvs, scanning, scan, addManual, removeManual,
    status, connected, host, info, tvState, error, connectedAt,
    connect, finishPairing, cancelPairing,
    press, pressedKey, launch, sendText, setVolume, powerOn, run,
    feedback, toast, dismissToast: () => setToast(null), notify,
    log, clearLog: () => setLog([]),
  }
}

export type RemoteApi = ReturnType<typeof useRemoteState>

const Ctx = createContext<RemoteApi | null>(null)

export function RemoteProvider({ children }: { children: ReactNode }) {
  const value = useRemoteState()
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useRemote(): RemoteApi {
  const v = useContext(Ctx)
  if (!v) throw new Error('useRemote must be used inside RemoteProvider')
  return v
}
