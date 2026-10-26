import { LoaderCircle, Power, Tv, X } from 'lucide-react'
import { since, useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { PLATFORM_LABEL, canConnect, canPowerOn, isSupported } from '../lib/capabilities'
import type { TV } from '../lib/types'

/** One TV in the sidebar. The selected TV expands with live details. */
export function DeviceCard({ tv, index }: { tv: TV; index: number }) {
  const { host, status, pairingMode, info, connectedAt, connect, removeManual, powerOn, feedback } = useRemote()
  const now = useNow()
  const selected = tv.ip === host
  const live = selected && status === 'connected'
  const busy = selected && (status === 'connecting' || status === 'pairing')

  const supported = isSupported(tv.platform)
  const connectable = canConnect(tv)
  const canWake = canPowerOn(tv) && !live && !busy
  const waking = feedback[`power:${tv.ip}`] === 'pending'

  const state = !supported
    ? 'Suporte em breve'
    : live
    ? `Conectado${connectedAt ? ` ${since(connectedAt, now)}` : ''}`
    : busy
      ? status === 'pairing' ? (pairingMode === 'prompt' ? 'Confirme na TV' : 'Aguardando código') : 'Conectando…'
      : selected && status === 'error'
        ? 'Desconectado'
        : tv.manual ? 'Adicionada manualmente' : 'Disponível'

  return (
    <div className={`device-wrap pop ${selected ? 'selected' : ''}`} style={{ animationDelay: `${index * 60}ms` }}>
      <button
        className={`device ${selected ? 'active' : ''} ${live ? 'live' : ''}`}
        onClick={() => connect(tv.ip)}
        disabled={!connectable || busy}
        title={!supported ? 'Suporte em breve' : !connectable ? 'Sem suporte ao Remote v2' : `Conectar em ${tv.name}`}
      >
        <div className="ic">
          <Tv size={16} strokeWidth={1.8} />
        </div>
        <div className="meta">
          <b>{live && info ? `${info.vendor} ${info.model}` : tv.name}</b>
          <span>
            {tv.ip}
            {live && info?.version && info.platform === 'androidtv' ? ` · Android ${info.version}` : ''}
          </span>
          <span className="chip platform">{PLATFORM_LABEL[tv.platform]}</span>
          <span className={`state ${live ? 'on' : selected && status === 'error' ? 'bad' : ''}`}>{state}</span>
        </div>
        {busy ? (
          <LoaderCircle size={14} className="spin muted-ic" />
        ) : (
          <div className={`dot ${live ? 'live' : selected && status === 'error' ? 'bad' : 'off'}`} />
        )}
      </button>
      {canWake && (
        <button className="device-wake" onClick={() => powerOn(tv.ip)} disabled={waking} title={tv.platform === 'roku' ? 'Ligar a TV pela rede' : 'Ligar a TV (Wake-on-LAN)'}>
          {waking ? <LoaderCircle size={12} className="spin" /> : <Power size={12} />}
          Ligar
        </button>
      )}
      {tv.manual && !selected && (
        <button className="device-remove" onClick={() => removeManual(tv.ip)} aria-label={`Remover ${tv.ip}`} title="Remover">
          <X size={12} />
        </button>
      )}
    </div>
  )
}
