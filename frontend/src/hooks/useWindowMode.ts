// Normal vs compact window. The frontend owns the mode (remembered in
// localStorage); Go only mirrors it in the tray menu and can ask to toggle it.
import { useCallback, useEffect, useRef, useState } from 'react'
import * as api from '../lib/backend'
import { load, save } from '../lib/storage'

const COMPACT_KEY = 'optic.compact.v1'
const PIN_KEY = 'optic.compactPin.v1'
const NORMAL_SIZE_KEY = 'optic.normalSize.v1'

type Size = { w: number; h: number }
const NORMAL: Size = { w: 820, h: 880 }
const NORMAL_MIN: Size = { w: 460, h: 640 }
const COMPACT: Size = { w: 360, h: 520 }
const COMPACT_MIN: Size = { w: 320, h: 440 }

export function useWindowMode() {
  const [compact, setCompact] = useState(() => load(COMPACT_KEY, false))
  const [pinned, setPinned] = useState(() => load(PIN_KEY, true))
  const [maximised, setMaximised] = useState(false)
  const [fullscreen, setFullscreen] = useState(false)
  const first = useRef(true)

  // Resize the native window whenever the mode changes (and once on launch).
  useEffect(() => {
    const initial = first.current
    first.current = false
    save(COMPACT_KEY, compact)
    if (!api.hasRuntime()) return
    api.SetCompactMode(compact).catch(() => {})

    if (compact) {
      // Remember the normal size to restore it later (read it before
      // shrinking); skip on launch, when the window has the default size.
      const remember = initial
        ? Promise.resolve()
        : api.WindowGetSize().then((s) => s.w >= NORMAL_MIN.w && s.h >= NORMAL_MIN.h && save(NORMAL_SIZE_KEY, s))
      remember.catch(() => {}).finally(() => api.ResizeWindow(COMPACT_MIN.w, COMPACT_MIN.h, COMPACT.w, COMPACT.h))
    } else if (!initial) {
      const s = load<Size>(NORMAL_SIZE_KEY, NORMAL)
      api.ResizeWindow(NORMAL_MIN.w, NORMAL_MIN.h, Math.max(s.w, NORMAL_MIN.w), Math.max(s.h, NORMAL_MIN.h))
    }
  }, [compact])

  useEffect(() => {
    save(PIN_KEY, pinned)
    if (api.hasRuntime()) api.WindowSetAlwaysOnTop(compact && pinned)
  }, [compact, pinned])

  // "Modo compacto" in the tray menu.
  useEffect(() => api.onEvent('tray:compact', (on: boolean) => setCompact(on)), [])

  // Maximized/fullscreen have no runtime events; re-read them on resize.
  useEffect(() => {
    if (!api.hasRuntime()) return
    const sync = () => {
      api.WindowIsMaximised().then(setMaximised).catch(() => {})
      api.WindowIsFullscreen().then(setFullscreen).catch(() => {})
    }
    sync()
    window.addEventListener('resize', sync)
    return () => window.removeEventListener('resize', sync)
  }, [])

  // Fullscreen is for the full remote; the compact window stays small.
  const toggleFullscreen = useCallback(() => {
    if (!api.hasRuntime() || compact) return
    api.ToggleFullscreen().then(setFullscreen).catch(() => {})
  }, [compact])

  // F11 toggles fullscreen, like a browser.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'F11') return
      e.preventDefault()
      toggleFullscreen()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [toggleFullscreen])

  const toggleCompact = useCallback(() => setCompact((c) => !c), [])
  const togglePin = useCallback(() => setPinned((p) => !p), [])
  const toggleMaximise = useCallback(() => api.hasRuntime() && api.WindowToggleMaximise(), [])

  return { compact, pinned, maximised, fullscreen, toggleCompact, togglePin, toggleMaximise, toggleFullscreen }
}
