import type { TVApp } from './types'

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
