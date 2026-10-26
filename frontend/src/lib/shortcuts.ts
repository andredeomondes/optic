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

/** Shown in the footer, tooltips and Settings › Atalhos. */
export const SHORTCUTS: { id: string; keys: string[]; label: string }[] = [
  { id: 'navigate', keys: ['↑', '↓', '←', '→'], label: 'Navegar' },
  { id: 'ok', keys: ['Enter'], label: 'OK' },
  { id: 'back', keys: ['Esc'], label: 'Voltar' },
  { id: 'home', keys: ['H'], label: 'Início' },
  { id: 'mute', keys: ['M'], label: 'Mudo' },
  { id: 'volume', keys: ['+', '−'], label: 'Volume' },
  { id: 'wheel', keys: ['Roda do mouse'], label: 'Volume pela roda' },
  { id: 'playPause', keys: ['Espaço'], label: 'Play / pause' },
  { id: 'palette', keys: ['Ctrl', 'K'], label: 'Comandos rápidos' },
  { id: 'type', keys: ['Ctrl', 'T'], label: 'Digitar na TV' },
  { id: 'paste', keys: ['Ctrl', 'V'], label: 'Colar na TV' },
]

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
