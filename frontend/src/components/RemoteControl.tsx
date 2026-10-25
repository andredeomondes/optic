import { Fragment, useMemo, WheelEvent } from 'react'
import { useRemote } from '../hooks/useRemote'
import { can } from '../lib/capabilities'
import { SHORTCUTS } from '../lib/shortcuts'
import { createThrottle, wheelStep } from '../lib/throttle'
import { ConnectionStatus } from './ConnectionStatus'
import { DPad } from './DPad'
import { MediaControls, NavButtons } from './MediaControls'
import { TextInput } from './TextInput'
import { VolumeControl } from './VolumeControl'

const WHEEL_THROTTLE_MS = 120

/**
 * The main remote. `compact` keeps only D-pad, navigation, volume and media,
 * ready for a future compact window.
 */
export function RemoteControl({ compact = false }: { compact?: boolean }) {
  const { info, connected, press } = useRemote()
  // One gate per panel, so its window survives re-renders.
  const wheelGate = useMemo(() => createThrottle(WHEEL_THROTTLE_MS), [])

  // Wheel over the panel changes the TV volume. React attaches wheel listeners
  // as passive, so preventDefault would be ignored; the panel still scrolls.
  function onWheel(e: WheelEvent<HTMLElement>) {
    const step = wheelStep(e.deltaY)
    if (!step || !connected || !wheelGate()) return
    press(step)
  }

  return (
    <section
      className={`glass remote enter ${compact ? 'compact' : ''}`}
      style={{ animationDelay: '120ms' }}
      onWheel={onWheel}
    >
      {!compact && <ConnectionStatus />}
      <VolumeControl />
      <DPad />
      <NavButtons />
      <MediaControls />
      {!compact && (
        <>
          {can(info, 'text') && (
            <>
              <div className="section-label">Digitar na TV</div>
              <TextInput />
            </>
          )}
          <ShortcutHints />
        </>
      )}
    </section>
  )
}

function ShortcutHints() {
  const shown = SHORTCUTS.filter((s) => ['Navegar', 'OK', 'Voltar', 'Início', 'Mudo', 'Comandos rápidos'].includes(s.label))
  return (
    <div className="hint">
      {shown.map((s, i) => (
        <Fragment key={s.label}>
          {i > 0 && ' · '}
          {s.keys.map((k) => (
            <kbd key={k}>{k}</kbd>
          ))}{' '}
          {s.label.toLowerCase()}
        </Fragment>
      ))}
    </div>
  )
}
