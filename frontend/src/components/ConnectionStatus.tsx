import { LoaderCircle, Power, RefreshCw } from 'lucide-react'
import { since, useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { InputPicker } from './InputPicker'
import { RemoteButton } from './RemoteButton'
import { SleepTimer } from './SleepTimer'

/** Remote header: TV name, live status line and the power / reconnect button. */
export function ConnectionStatus() {
  const { status, pairingMode, host, info, tvs, error, connectedAt, connect } = useRemote()
  const now = useNow()
  const name = info ? `${info.vendor} ${info.model}` : tvs.find((t) => t.ip === host)?.name || host

  return (
    <div className="top">
      <div className="tv-name">
        <div className="tv-title">
          {status !== 'idle' && <span className={`dot ${status === 'connected' ? 'live' : status === 'error' ? 'bad' : 'off'}`} />}
          {status === 'idle' ? 'Escolha uma TV' : name}
        </div>
        <small className={`status-line ${status}`} key={status}>
          {status === 'connected' && (
            <>
              <span>Conectado {connectedAt ? since(connectedAt, now) : ''}</span>
              {info?.version && <span className="chip">{info.platform === 'roku' ? 'Roku OS' : info.platform === 'webos' ? 'webOS' : 'Android'} {info.version}</span>}
              <span className="chip">{host}</span>
            </>
          )}
          {status === 'connecting' && (
            <>
              <LoaderCircle size={12} className="spin" /> Conectando…
            </>
          )}
          {status === 'pairing' && (pairingMode === 'prompt' ? 'Confirme na TV' : 'Aguardando código de pareamento')}
          {status === 'error' && (error || 'Erro')}
          {status === 'idle' && 'Selecione uma TV na lista ao lado'}
        </small>
      </div>
      {status === 'error' && host ? (
        <div className="top-actions">
          <SleepTimer />
          <button className="btn retry" onClick={() => connect(host)}>
            <RefreshCw size={14} /> Reconectar
          </button>
        </div>
      ) : (
        <div className="top-actions">
          <InputPicker />
          <SleepTimer />
          <RemoteButton tvKey="POWER" label="Ligar / desligar" className="power">
            <Power size={18} strokeWidth={1.8} />
          </RemoteButton>
        </div>
      )}
    </div>
  )
}
