import type { TVApp, TVInput } from './types'

/** Lowercase, accent-free, trimmed: the form used to compare names. */
export const normalize = (s: string): string =>
  s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase().trim()

/** Apps sorted by name (pt-BR); the input is not changed. */
export const sortApps = (apps: TVApp[]): TVApp[] =>
  [...apps].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))

/** Apps whose name contains the query, ignoring case and accents. */
export function filterApps(apps: TVApp[], query: string): TVApp[] {
  const q = normalize(query)
  const sorted = sortApps(apps)
  return q ? sorted.filter((a) => normalize(a.name).includes(q)) : sorted
}

const squash = (s: string): string => normalize(s).replace(/\s+/g, '')

/** Picks an input by name: exact, then ignoring spaces ("hdmi2"), then prefix. */
export function matchInput(inputs: TVInput[], query: string): TVInput | undefined {
  const q = normalize(query)
  if (!q) return undefined
  const sq = squash(query)
  return (
    inputs.find((i) => normalize(i.name) === q) ??
    inputs.find((i) => squash(i.name) === sq) ??
    inputs.find((i) => squash(i.name).startsWith(sq))
  )
}
