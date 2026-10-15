import { load, save } from './storage'

export type AppDef = {
  id: string
  name: string
  /** Android package; launched as market://launch?id=<pkg>. */
  pkg: string
  brand: string
  custom?: boolean
}

/** Known Android TV apps. Future: merge with apps reported by the TV. */
export const CATALOG: AppDef[] = [
  { id: 'youtube', name: 'YouTube', pkg: 'com.google.android.youtube.tv', brand: '#FF0000' },
  { id: 'netflix', name: 'Netflix', pkg: 'com.netflix.ninja', brand: '#E50914' },
  { id: 'prime', name: 'Prime Video', pkg: 'com.amazon.amazonvideo.livingroom', brand: '#1A98FF' },
  { id: 'globoplay', name: 'Globoplay', pkg: 'com.globo.globotv', brand: '#FF3D55' },
  { id: 'disney', name: 'Disney+', pkg: 'com.disney.disneyplus', brand: '#113CCF' },
  { id: 'max', name: 'Max', pkg: 'com.wbd.stream', brand: '#002BE7' },
  { id: 'spotify', name: 'Spotify', pkg: 'com.spotify.tv.android', brand: '#1DB954' },
  { id: 'twitch', name: 'Twitch', pkg: 'tv.twitch.android.app', brand: '#9146FF' },
  { id: 'crunchyroll', name: 'Crunchyroll', pkg: 'com.crunchyroll.crunchyroid', brand: '#F47521' },
  { id: 'plex', name: 'Plex', pkg: 'com.plexapp.android', brand: '#E5A00D' },
  { id: 'kodi', name: 'Kodi', pkg: 'org.xbmc.kodi', brand: '#17B2E7' },
  { id: 'vlc', name: 'VLC', pkg: 'org.videolan.vlc', brand: '#FF8800' },
  { id: 'playstore', name: 'Play Store', pkg: 'com.android.vending', brand: '#01875F' },
  { id: 'paramount', name: 'Paramount+', pkg: 'com.cbs.ott', brand: '#0064FF' },
  { id: 'appletv', name: 'Apple TV', pkg: 'com.apple.atve.androidtv.appletv', brand: '#000000' },
  { id: 'deezer', name: 'Deezer', pkg: 'deezer.android.tv', brand: '#A238FF' },
  { id: 'ytmusic', name: 'YouTube Music', pkg: 'com.google.android.youtube.tvmusic', brand: '#FF0000' },
  { id: 'ytkids', name: 'YouTube Kids', pkg: 'com.google.android.youtube.tvkids', brand: '#FF0000' },
  { id: 'steamlink', name: 'Steam Link', pkg: 'com.valvesoftware.steamlink', brand: '#171A21' },
]

export const launchLink = (app: AppDef) => `market://launch?id=${app.pkg}`

type Stored = { favorites: string[]; custom: AppDef[] }
const KEY = 'optic.apps.v1'
const DEFAULT: Stored = { favorites: ['youtube', 'netflix', 'prime', 'globoplay'], custom: [] }

export function loadApps(): Stored {
  const s = load<Stored>(KEY, DEFAULT)
  return { favorites: s.favorites ?? DEFAULT.favorites, custom: s.custom ?? [] }
}

export function saveApps(s: Stored) {
  save(KEY, s)
}

export function allApps(custom: AppDef[]): AppDef[] {
  return [...CATALOG, ...custom]
}

export function makeCustomApp(name: string, pkg: string): AppDef {
  return { id: `custom:${pkg}`, name, pkg, brand: '#7C5CFF', custom: true }
}

/** Android package names: dot-separated identifiers, at least two parts. */
export const isPackageName = (s: string) => /^[a-zA-Z][\w]*(\.[a-zA-Z_][\w]*)+$/.test(s)

/**
 * Apps that draw their own on-screen keyboard instead of Android's, so the
 * TV never reports a text field and typed text can't reach them.
 * Verified: Netflix on a TCL Google TV.
 */
export const OWN_KEYBOARD_APPS: Record<string, string> = {
  'com.netflix.ninja': 'Netflix',
}
