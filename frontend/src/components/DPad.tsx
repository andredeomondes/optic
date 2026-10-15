import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

export function DPad() {
  const { connected } = useRemote()
  return (
    <div className={`dpad ${connected ? 'on' : ''}`}>
      <RemoteButton tvKey="DPAD_UP" label="Cima" className="arr up">
        <ChevronUp size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_LEFT" label="Esquerda" className="arr left">
        <ChevronLeft size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_RIGHT" label="Direita" className="arr right">
        <ChevronRight size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_DOWN" label="Baixo" className="arr down">
        <ChevronDown size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_CENTER" label="OK" className="ok">
        OK
      </RemoteButton>
    </div>
  )
}
