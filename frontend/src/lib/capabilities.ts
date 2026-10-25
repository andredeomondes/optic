import type { Capabilities, DeviceInfo, Platform } from './types.ts'

/** Same labels as the Go side (Platform.Label). */
export const PLATFORM_LABEL: Record<Platform, string> = {
  androidtv: 'Android TV',
  tizen: 'Samsung',
  webos: 'LG webOS',
  roku: 'Roku',
  '': 'Desconhecida',
}

/** Platforms OPTIC can control today; the rest are only listed. */
export function isSupported(p: Platform): boolean {
  return p === 'androidtv' || p === ''
}

/** Whether the connected TV supports a feature (false while not connected). */
export function can(info: DeviceInfo | null, cap: keyof Capabilities): boolean {
  return info?.capabilities?.[cap] === true
}
