const MAX_PASTE = 500

type KeyLike = { ctrlKey: boolean; metaKey: boolean; key: string }
type TargetLike = { tagName?: string; isContentEditable?: boolean } | null

/** Ctrl/Cmd+V pastes to the TV only when connected and not typing in a field. */
export function shouldPasteToTV(e: KeyLike, target: TargetLike, connected: boolean): boolean {
  if (!connected || !(e.ctrlKey || e.metaKey) || e.key.toLowerCase() !== 'v') return false
  const typing = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable
  return !typing
}

/** Single-line, trimmed and capped: what the TV's text field can take. */
export function cleanPaste(text: string): string {
  return text.replace(/\r/g, '').replace(/\n/g, ' ').trim().slice(0, MAX_PASTE)
}
