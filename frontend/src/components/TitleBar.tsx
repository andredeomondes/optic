import { Copy, Maximize, Minimize, Minus, PictureInPicture2, Pin, PinOff, Square, X } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import { Quit, WindowMinimise } from '../lib/backend'

type Props = {
  compact: boolean
  pinned: boolean
  maximised: boolean
  fullscreen: boolean
  onToggleCompact: () => void
  onTogglePin: () => void
  onToggleMaximise: () => void
  onToggleFullscreen: () => void
}

/**
 * Frameless window chrome: drag area plus window buttons.
 * Minimize is the normal Windows one; close hides to the tray by default
 * (Go's OnBeforeClose, see tray.go). Quitting for real is the tray menu's "Sair".
 */
export function TitleBar(p: Props) {
  const { compact, pinned, maximised, fullscreen } = p
  return (
    <header className={`titlebar ${compact ? 'compact' : ''}`}>
      {compact ? <CompactTitle /> : <span className="title">OPTIC</span>}
      <div className="win">
        {compact && (
          <WinBtn label="Sempre no topo" onClick={p.onTogglePin} className={pinned ? 'on' : ''} pressed={pinned}>
            {pinned ? <Pin size={13} /> : <PinOff size={13} />}
          </WinBtn>
        )}
        {!compact && (
          <WinBtn label={fullscreen ? 'Sair da tela cheia (F11)' : 'Tela cheia (F11)'} onClick={p.onToggleFullscreen}>
            {fullscreen ? <Minimize size={13} /> : <Maximize size={13} />}
          </WinBtn>
        )}
        <WinBtn label={compact ? 'Janela normal' : 'Modo compacto'} onClick={p.onToggleCompact} className={compact ? 'on' : ''}>
          <PictureInPicture2 size={14} />
        </WinBtn>
        <WinBtn label="Minimizar" onClick={WindowMinimise}>
          <Minus size={14} />
        </WinBtn>
        {!compact && !fullscreen && (
          <WinBtn label={maximised ? 'Restaurar' : 'Maximizar'} onClick={p.onToggleMaximise}>
            {maximised ? <Copy size={12} /> : <Square size={11} />}
          </WinBtn>
        )}
        <WinBtn label="Fechar" onClick={Quit} className="close">
          <X size={14} />
        </WinBtn>
      </div>
    </header>
  )
}

function WinBtn(props: { label: string; onClick: () => void; className?: string; pressed?: boolean; children: React.ReactNode }) {
  return (
    <button
      onClick={() => props.onClick()}
      className={props.className}
      aria-label={props.label}
      aria-pressed={props.pressed}
      title={props.label}
    >
      {props.children}
    </button>
  )
}

/** In compact mode the connection header is hidden, so the title shows the TV. */
function CompactTitle() {
  const { status, host, info, tvs } = useRemote()
  const name = info ? info.model : tvs.find((t) => t.ip === host)?.name || host
  return (
    <span className="title compact-title">
      <span className={`dot ${status === 'connected' ? 'live' : status === 'error' ? 'bad' : 'off'}`} />
      {status === 'idle' || !name ? 'OPTIC' : name}
    </span>
  )
}
