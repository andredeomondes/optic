// Semantic version comparison for the update notice. No local imports: runs under node --test.
export interface Version {
  major: number
  minor: number
  patch: number
  pre: string
}

const SEMVER = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$/

export function parseVersion(v: string): Version | null {
  const m = SEMVER.exec(v.trim())
  if (!m) return null
  return { major: Number(m[1]), minor: Number(m[2]), patch: Number(m[3]), pre: m[4] ?? '' }
}

/** True when `latest` is a stable release strictly newer than `current`. Unparseable input is never newer. */
export function isNewer(current: string, latest: string): boolean {
  const c = parseVersion(current)
  const l = parseVersion(latest)
  if (!c || !l || l.pre) return false
  if (l.major !== c.major) return l.major > c.major
  if (l.minor !== c.minor) return l.minor > c.minor
  return l.patch > c.patch
}

export const RELEASES_URL = 'https://github.com/andredeomondes/optic/releases'

/** True when `url` points at GitHub over https, the only place release links may open. */
export function isReleaseURL(url: string): boolean {
  return url.startsWith('https://github.com/')
}

/** The release link when safe, otherwise the releases page. */
export function safeReleaseURL(url: string): string {
  return isReleaseURL(url) ? url : RELEASES_URL
}
