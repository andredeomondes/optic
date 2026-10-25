/** Wheel deltas smaller than this are sensor noise (touchpads send many tiny ones). */
const MIN_DELTA = 1

/**
 * Maps a wheel delta to a TV volume key: scrolling up (negative deltaY) raises
 * the volume, scrolling down lowers it. Returns null for negligible movement.
 */
export function wheelStep(deltaY: number): 'VOLUME_UP' | 'VOLUME_DOWN' | null {
  if (Math.abs(deltaY) < MIN_DELTA) return null
  return deltaY < 0 ? 'VOLUME_UP' : 'VOLUME_DOWN'
}

/**
 * Returns a gate that lets through at most one call per `ms` window. The window
 * starts at the last call that was let through, so rejected calls do not extend it.
 */
export function createThrottle(ms: number, now: () => number = Date.now): () => boolean {
  let last = -Infinity
  return () => {
    const t = now()
    if (t - last < ms) return false
    last = t
    return true
  }
}
