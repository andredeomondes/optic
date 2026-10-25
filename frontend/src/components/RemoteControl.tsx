import { Fragment } from 'react'
import { useRemote } from '../hooks/useRemote'
import { can } from '../lib/capabilities'
import { SHORTCUTS } from '../lib/shortcuts'
import { ConnectionStatus } from './ConnectionStatus'
import { DPad } from './DPad'
import { MediaControls, NavButtons } from './MediaControls'
import { TextInput } from './TextInput'
import { VolumeControl } from './VolumeControl'

/**
 * The main remote. `compact` keeps only D-pad, navigation, volume and media,
 * ready for a future compact window.
 */
export function RemoteControl({ compact = false }: { compact?: boolean }) {
  const { info } = useRemote()
  return (
    <section className={`glass remote enter ${compact ? 'compact' : ''}`} style={{ animationDelay: '120ms' }}>
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
