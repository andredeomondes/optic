import type { TVApp } from './types'

type Ctx = { installed: TVApp[]; catalog: { pkg: string; name: string }[] }

/** Readable name for the app id a TV reports as currently open. */
export function appLabel(id: string, ctx: Ctx): string {
  if (!id) return ''
  const hit = ctx.installed.find((a) => a.id === id)
  if (hit) return hit.name === id ? id : hit.name // Roku sends names as ids
  const known = ctx.catalog.find((c) => c.pkg === id)
  if (known) return known.name
  const last = id.split('.').filter(Boolean).pop() ?? id
  return last.charAt(0).toUpperCase() + last.slice(1)
}

export function playingLabel(p: string): string {
  if (p === 'playing') return 'Tocando'
  if (p === 'paused') return 'Pausado'
  return ''
}
