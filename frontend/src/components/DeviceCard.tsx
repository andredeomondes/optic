import { LoaderCircle, Tv, X } from 'lucide-react'
import { since, useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import type { TV } from '../lib/types'

/** One TV in the sidebar. The selected TV expands with live details. */
export function DeviceCard({ tv, index }: { tv: TV; index: number }) {
  const { host, status, info, connectedAt, connect, removeManual } = useRemote()
  const now = useNow()
  const selected = tv.ip === host
  const live = selected && status === 'connected'
  const busy = selected && (status === 'connecting' || status === 'pairing')

  const state = live
    ? `Conectado${connectedAt ? ` ${since(connectedAt, now)}` : ''}`
    : busy
      ? status === 'pairing' ? 'Aguardando código' : 'Conectando…'
      : selected && status === 'error'
        ? 'Desconectado'
        : tv.manual ? 'Adicionada manualmente' : 'Disponível'

  return (
    <div className={`device-wrap pop ${selected ? 'selected' : ''}`} style={{ animationDelay: `${index * 60}ms` }}>
      <button
        className={`device ${selected ? 'active' : ''} ${live ? 'live' : ''}`}
        onClick={() => connect(tv.ip)}
        disabled={!tv.remotev2 || busy}
        title={tv.remotev2 ? `Conectar em ${tv.name}` : 'Sem suporte ao Remote v2'}
      >
        <div className="ic">
          <Tv size={16} strokeWidth={1.8} />
        </div>
        <div className="meta">
          <b>{live && info ? `${info.vendor} ${info.model}` : tv.name}</b>
          <span>
            {tv.ip}
            {live && info?.version ? ` · Android ${info.version}` : ''}
          </span>
          <span className={`state ${live ? 'on' : selected && status === 'error' ? 'bad' : ''}`}>{state}</span>
        </div>
        {busy ? (
          <LoaderCircle size={14} className="spin muted-ic" />
        ) : (
          <div className={`dot ${live ? 'live' : selected && status === 'error' ? 'bad' : 'off'}`} />
        )}
      </button>
      {tv.manual && !selected && (
        <button className="device-remove" onClick={() => removeManual(tv.ip)} aria-label={`Remover ${tv.ip}`} title="Remover">
          <X size={12} />
        </button>
      )}
    </div>
  )
}
