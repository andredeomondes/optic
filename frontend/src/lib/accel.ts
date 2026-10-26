// Accelerator strings in the canonical form the Go side emits:
// modifiers in order Ctrl, Alt, Shift, then the key (e.g. "Ctrl+Alt+Up").

export type KeyEventLike = {
  key: string
  code: string
  ctrlKey: boolean
  altKey: boolean
  shiftKey: boolean
  metaKey: boolean
}

const NAMED: Record<string, string> = {
  ArrowUp: 'Up',
  ArrowDown: 'Down',
  ArrowLeft: 'Left',
  ArrowRight: 'Right',
  Home: 'Home',
  End: 'End',
  PageUp: 'PageUp',
  PageDown: 'PageDown',
  ' ': 'Space',
  MediaPlayPause: 'MediaPlayPause',
  MediaTrackNext: 'MediaNext',
  MediaTrackPrevious: 'MediaPrev',
  MediaStop: 'MediaStop',
  AudioVolumeUp: 'VolumeUp',
  AudioVolumeDown: 'VolumeDown',
  AudioVolumeMute: 'VolumeMute',
}

const MODIFIER_KEYS = new Set(['Control', 'Alt', 'Shift', 'Meta', 'AltGraph', 'OS'])

function keyName(e: KeyEventLike): string | null {
  if (e.code.startsWith('Numpad')) return null // not supported
  const named = NAMED[e.key]
  if (named) return named
  if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key
  // Letters follow the layout (RegisterHotKey uses the same virtual keys), so
  // AZERTY/QWERTZ/Dvorak work. Alt/AltGr may turn key into a symbol or dead
  // key; only then fall back to the physical code.
  if (/^[a-zA-Z]$/.test(e.key)) return e.key.toUpperCase()
  const letter = /^Key([A-Z])$/.exec(e.code)
  if (letter) return letter[1]
  const digit = /^Digit([0-9])$/.exec(e.code)
  if (digit) return digit[1]
  return null
}

/** Canonical accelerator for a key event, or null (modifier-only, Win key, unsupported key). */
export function accelFromEvent(e: KeyEventLike): string | null {
  if (MODIFIER_KEYS.has(e.key) || e.metaKey) return null
  const key = keyName(e)
  if (!key) return null
  const parts: string[] = []
  if (e.ctrlKey) parts.push('Ctrl')
  if (e.altKey) parts.push('Alt')
  if (e.shiftKey) parts.push('Shift')
  parts.push(key)
  return parts.join('+')
}

const ARROWS: Record<string, string> = { Up: '↑', Down: '↓', Left: '←', Right: '→' }

/** Display form: "Ctrl+Alt+Up" → "Ctrl + Alt + ↑". */
export function labelFor(accel: string): string {
  if (!accel) return ''
  return accel
    .split('+')
    .map((p) => ARROWS[p] ?? p)
    .join(' + ')
}
