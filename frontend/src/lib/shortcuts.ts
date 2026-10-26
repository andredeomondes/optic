import type { Key } from './i18n.ts'

/** Physical keys that map straight to a TV key while connected. */
export const KEYMAP: Record<string, string> = {
  ArrowUp: 'DPAD_UP',
  ArrowDown: 'DPAD_DOWN',
  ArrowLeft: 'DPAD_LEFT',
  ArrowRight: 'DPAD_RIGHT',
  Enter: 'DPAD_CENTER',
  Escape: 'BACK',
  Backspace: 'BACK',
  h: 'HOME',
  H: 'HOME',
  m: 'MUTE',
  M: 'MUTE',
  '+': 'VOLUME_UP',
  '=': 'VOLUME_UP',
  '-': 'VOLUME_DOWN',
  ' ': 'MEDIA_PLAY_PAUSE',
}

/** Shown in the footer, tooltips and Settings › Shortcuts. Key tokens in NAMED_KEYS are translated. */
export const SHORTCUTS: { id: string; keys: string[]; labelKey: Key }[] = [
  { id: 'navigate', keys: ['↑', '↓', '←', '→'], labelKey: 'shortcut.navigate' },
  { id: 'ok', keys: ['Enter'], labelKey: 'shortcut.ok' },
  { id: 'back', keys: ['Esc'], labelKey: 'shortcut.back' },
  { id: 'home', keys: ['H'], labelKey: 'shortcut.home' },
  { id: 'mute', keys: ['M'], labelKey: 'shortcut.mute' },
  { id: 'volume', keys: ['+', '−'], labelKey: 'shortcut.volume' },
  { id: 'wheel', keys: ['Wheel'], labelKey: 'shortcut.wheel' },
  { id: 'playPause', keys: ['Space'], labelKey: 'shortcut.playPause' },
  { id: 'palette', keys: ['Ctrl', 'K'], labelKey: 'shortcut.palette' },
  { id: 'type', keys: ['Ctrl', 'T'], labelKey: 'shortcut.type' },
  { id: 'paste', keys: ['Ctrl', 'V'], labelKey: 'shortcut.paste' },
]

/** Key tokens that have a word instead of a symbol. */
export const NAMED_KEYS: Record<string, Key> = { Wheel: 'shortcut.keyWheel', Space: 'remote.keySpace' }

/** Shortcut ids shown in the remote's footer hints, in order. */
export const FOOTER_HINTS = ['navigate', 'ok', 'back', 'home', 'mute', 'palette'] as const

/** Reverse lookup for button tooltips: TV key → physical shortcut. */
export const KEY_HINT: Record<string, string> = {
  DPAD_CENTER: 'Enter',
  BACK: 'Esc',
  HOME: 'H',
  MUTE: 'M',
  VOLUME_UP: '+',
  VOLUME_DOWN: '−',
  MEDIA_PLAY_PAUSE: 'Espaço',
}
