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
export const SHORTCUTS: { keys: string[]; label: string }[] = [
  { keys: ['↑', '↓', '←', '→'], label: 'Navegar' },
  { keys: ['Enter'], label: 'OK' },
  { keys: ['Esc'], label: 'Voltar' },
  { keys: ['H'], label: 'Início' },
  { keys: ['M'], label: 'Mudo' },
  { keys: ['+', '−'], label: 'Volume' },
  { keys: ['Roda do mouse'], label: 'Volume pela roda' },
  { keys: ['Espaço'], label: 'Play / pause' },
  { keys: ['Ctrl', 'K'], label: 'Comandos rápidos' },
  { keys: ['Ctrl', 'T'], label: 'Digitar na TV' },
  { keys: ['Ctrl', 'V'], label: 'Colar na TV' },
]

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
