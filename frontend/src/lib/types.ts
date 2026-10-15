export type TV = { name: string; model: string; ip: string; remotev2: boolean; manual?: boolean }
export type DeviceInfo = { model: string; vendor: string; version: string; appVersion: string }
export type TVState = {
  on: boolean
  volume: number
  volumeMax: number
  muted: boolean
  /** Foreground app package, e.g. com.netflix.ninja. */
  app?: string
  /** Android on-screen keyboard is open on the TV. */
  keyboard?: boolean
}
export type Status = 'idle' | 'connecting' | 'pairing' | 'connected' | 'error'
export type LogLine = { id: number; dir: string; text: string; time: string }

/** Visual result of an action, keyed by action id (e.g. "key:HOME"). */
export type Feedback = 'pending' | 'ok' | 'err'

export type Toast = { id: number; kind: 'ok' | 'err' | 'info'; text: string }
