import type { Key } from './i18n.ts'
import type { Platform, TVApp } from './types.ts'

type Ctx = { installed: TVApp[]; catalog: { pkg: string; name: string }[] }

/** Readable name for the app id a TV reports as currently open. */
export function appLabel(id: string, ctx: Ctx): string {
  if (!id) return ''
  const hit = ctx.installed.find((a) => a.id === id)
  if (hit) return hit.name === id ? id : hit.name // Roku sends names as ids
  if (ctx.installed.some((a) => a.name === id)) return id // Roku sends the channel name
  const known = ctx.catalog.find((c) => c.pkg === id)
  if (known) return known.name
  const last = id.split('.').filter(Boolean).pop() ?? id
  return last.charAt(0).toUpperCase() + last.slice(1)
}

/** i18n key for the playback state, or '' when there is none to show. */
export function playingKey(p: string): Key | '' {
  if (p === 'playing') return 'now.playing'
  if (p === 'paused') return 'now.paused'
  return ''
}

/**
 * Status-line text: "App · <state>". `state` is the already translated
 * playback label. Android TV reports raw package names, so it shows a line
 * only for apps in the catalog.
 */
export function nowPlayingText(id: string, state: string, platform: Platform | undefined, ctx: Ctx): string {
  if (!id) return ''
  if (platform === 'androidtv' && !ctx.catalog.some((c) => c.pkg === id)) return ''
  const name = appLabel(id, ctx)
  return state ? `${name} · ${state}` : name
}
