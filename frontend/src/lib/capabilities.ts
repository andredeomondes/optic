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
  return p === 'androidtv' || p === '' || p === 'roku' || p === 'webos' || p === 'tizen'
}

/** Roku, webOS and Tizen need no Remote v2; Android TV does. */
export function canConnect(tv: TV): boolean {
  return isSupported(tv.platform) && (tv.platform === 'roku' || tv.platform === 'webos' || tv.platform === 'tizen' || tv.remotev2)
}

/** Whether the TV can be switched on from the network (Wake-on-LAN or Roku ECP). */
export function canPowerOn(tv: TV): boolean {
  return isSupported(tv.platform) && (!!tv.mac || tv.platform === 'roku')
}

/** Whether the connected TV supports a feature (false while not connected). */
export function can(info: DeviceInfo | null, cap: keyof Capabilities): boolean {
  return info?.capabilities?.[cap] === true
}
