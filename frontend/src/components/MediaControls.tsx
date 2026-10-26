import { House, Pause, Play, Rewind, FastForward, Settings, SkipBack, SkipForward, Undo2 } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

const ICON = { size: 18, strokeWidth: 1.8 }

/** Voltar / Início / Ajustes with their keyboard hints. */
export function NavButtons() {
  return (
    <div className="row3">
      <RemoteButton tvKey="BACK" label="Voltar" className="labeled" showHint>
        <Undo2 {...ICON} /> Voltar
      </RemoteButton>
      <RemoteButton tvKey="HOME" label="Início" className="labeled" showHint>
        <House {...ICON} /> Início
      </RemoteButton>
      <RemoteButton tvKey="SETTINGS" label="Ajustes da TV" className="labeled">
        <Settings {...ICON} /> Ajustes
      </RemoteButton>
    </div>
  )
}

/** LG has no previous/next keys, so those two are hidden on webOS. */
export function MediaControls() {
  const { info } = useRemote()
  const skip = info?.platform !== 'webos'
  return (
    <div className="media-row" style={skip ? undefined : { gridTemplateColumns: 'repeat(3, 1fr)' }}>
      {skip && (
        <RemoteButton tvKey="MEDIA_PREVIOUS" label="Anterior">
          <SkipBack {...ICON} />
        </RemoteButton>
      )}
      <RemoteButton tvKey="MEDIA_REWIND" label="Retroceder">
        <Rewind {...ICON} />
      </RemoteButton>
      <RemoteButton tvKey="MEDIA_PLAY_PAUSE" label="Play / pause" className="play">
        <span className="pp">
          <Play size={15} strokeWidth={1.8} />
          <Pause size={15} strokeWidth={1.8} />
        </span>
      </RemoteButton>
      <RemoteButton tvKey="MEDIA_FAST_FORWARD" label="Avançar">
        <FastForward {...ICON} />
      </RemoteButton>
      {skip && (
        <RemoteButton tvKey="MEDIA_NEXT" label="Próximo">
          <SkipForward {...ICON} />
        </RemoteButton>
      )}
    </div>
  )
}
