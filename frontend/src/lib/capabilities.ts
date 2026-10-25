import type { Capabilities, DeviceInfo, Platform, TV } from './types.ts'

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
  return p === 'androidtv' || p === '' || p === 'roku'
}

/** Roku needs no Remote v2 or pairing; Android TV does. */
export function canConnect(tv: TV): boolean {
  return isSupported(tv.platform) && (tv.platform === 'roku' || tv.remotev2)
}

/** Whether the connected TV supports a feature (false while not connected). */
export function can(info: DeviceInfo | null, cap: keyof Capabilities): boolean {
  return info?.capabilities?.[cap] === true
}
