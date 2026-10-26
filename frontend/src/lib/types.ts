export type Platform = 'androidtv' | 'tizen' | 'webos' | 'roku' | ''
export type Capabilities = {
  text: boolean
  appList: boolean
  absoluteVolume: boolean
  powerOn: boolean
  inputs: boolean
  pointer: boolean
  toast: boolean
}
export type TV = { name: string; model: string; ip: string; remotev2: boolean; platform: Platform; mac?: string; manual?: boolean }
export type TVApp = { id: string; name: string }
export type TVInput = { id: string; name: string }
export type PairingInfo = { name: string; mode: 'code' | 'prompt' }
export type DeviceInfo = { model: string; vendor: string; version: string; appVersion: string; platform: Platform; capabilities: Capabilities }
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
