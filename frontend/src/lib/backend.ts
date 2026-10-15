// Single entry point to the Go backend (Wails bindings + events).
// Components never import from wailsjs directly.
export {
  CancelPairing,
  CloseToTray,
  Connect,
  Discover,
  FinishPairing,
  LastHost,
  LaunchApp,
  ResizeWindow,
  SendKey,
  SendText,
  SetCloseToTray,
  SetCompactMode,
  SetVolume,
  StartPairing,
  ToggleFullscreen,
  TrayAvailable,
} from '../../wailsjs/go/main/App'
export {
  Quit,
  WindowGetSize,
  WindowIsFullscreen,
  WindowIsMaximised,
  WindowMinimise,
  WindowSetAlwaysOnTop,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime'

import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'

/** False when running under plain `vite` without the dev stub. */
export const hasRuntime = () => Boolean((window as any).runtime)

/** Subscribes to a backend event; returns an unsubscribe function. */
export function onEvent(name: string, cb: (...args: any[]) => void): () => void {
  if (!hasRuntime()) return () => {}
  EventsOn(name, cb)
  return () => EventsOff(name)
}

/** Go errors arrive as strings; normalize anything thrown into a message. */
export const errText = (e: unknown) => (e instanceof Error ? e.message : String(e))
