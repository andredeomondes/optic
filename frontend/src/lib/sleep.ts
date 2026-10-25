// Pure helpers for the sleep timer (no local imports so node --test can load it).

/** Minutes offered in the timer menu. */
export const SLEEP_PRESETS = [15, 30, 60, 90] as const

/** Whole minutes left until `deadline`, rounded up, never negative. */
export function remaining(deadline: number, now: number): number {
  return Math.max(0, Math.ceil((deadline - now) / 60_000))
}

/** Only toggle power when it can turn an ON TV off: pressing POWER on a sleeping TV would wake it. */
export function decideAtDeadline(s: { connected: boolean; tvOn: boolean }): 'power' | 'skip' {
  return s.connected && s.tvOn ? 'power' : 'skip'
}
