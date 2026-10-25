import { useEffect } from 'react'
import { shouldPasteToTV } from '../lib/paste'
import { KEYMAP } from '../lib/shortcuts'
import { TEXT_INPUT_ID } from '../components/TextInput'

type Opts = {
  connected: boolean
  press: (key: string) => void
  togglePalette: () => void
  paste: () => void
}

const isTyping = (t: EventTarget | null) => {
  const el = t as HTMLElement | null
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
}

/** Global keyboard: Ctrl+K palette, Ctrl+T text box, keys → TV while connected. */
export function useShortcuts({ connected, press, togglePalette, paste }: Opts) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        togglePalette()
        return
      }
      if (e.ctrlKey && e.key.toLowerCase() === 't') {
        e.preventDefault()
        document.getElementById(TEXT_INPUT_ID)?.focus()
        return
      }
      // Dialogs own the keyboard while open.
      if (document.querySelector('.overlay')) return
      if (shouldPasteToTV(e, e.target as HTMLElement | null, connected)) {
        e.preventDefault()
        paste()
        return
      }
      if (!connected || isTyping(e.target) || e.ctrlKey || e.altKey || e.metaKey) return
      const key = KEYMAP[e.key]
      if (!key) return
      e.preventDefault()
      press(key)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [connected, press, togglePalette, paste])
}
