import { LoaderCircle, Power, Tv, X } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { since, useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { PLATFORM_LABEL, canConnect, canPowerOn, isSupported } from '../lib/capabilities'
import type { TV } from '../lib/types'

/** One TV in the sidebar. The selected TV expands with live details. */
export function DeviceCard({ tv, index }: { tv: TV; index: number }) {
  const { host, status, pairingMode, info, connectedAt, connect, removeManual, powerOn, feedback } = useRemote()
  const now = useNow()
  const { t } = useI18n()
  const selected = tv.ip === host
  const live = selected && status === 'connected'
  const busy = selected && (status === 'connecting' || status === 'pairing')

  const supported = isSupported(tv.platform)
  const connectable = canConnect(tv)
  const canWake = canPowerOn(tv) && !live && !busy
  const waking = feedback[`power:${tv.ip}`] === 'pending'

  const state = !supported
    ? t('device.soon')
    : live
    ? `${t('common.connected')}${connectedAt ? ` ${since(connectedAt, now, t)}` : ''}`
    : busy
      ? status === 'pairing' ? (pairingMode === 'prompt' ? t('common.confirmOnTv') : t('device.waitingCode')) : t('common.connecting')
      : selected && status === 'error'
        ? t('device.disconnected')
        : tv.manual ? t('device.manual') : t('device.available')

  return (
    <div className={`device-wrap pop ${selected ? 'selected' : ''}`} style={{ animationDelay: `${index * 60}ms` }}>
      <button
        className={`device ${selected ? 'active' : ''} ${live ? 'live' : ''}`}
        onClick={() => connect(tv.ip)}
        disabled={!connectable || busy}
        title={!supported ? t('device.soon') : !connectable ? t('device.noRemoteV2') : t('device.connect', { name: tv.name })}
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
          <span className="chip platform">{tv.platform ? PLATFORM_LABEL[tv.platform] : t('device.platformUnknown')}</span>
          <span className={`state ${live ? 'on' : selected && status === 'error' ? 'bad' : ''}`}>{state}</span>
        </div>
        {busy ? (
          <LoaderCircle size={14} className="spin muted-ic" />
        ) : (
          <div className={`dot ${live ? 'live' : selected && status === 'error' ? 'bad' : 'off'}`} />
        )}
      </button>
      {canWake && (
        <button className="device-wake" onClick={() => powerOn(tv.ip)} disabled={waking} title={tv.platform === 'roku' ? t('device.wakeRoku') : t('device.wakeWol')}>
          {waking ? <LoaderCircle size={12} className="spin" /> : <Power size={12} />}
          {t('device.wake')}
        </button>
      )}
      {tv.manual && !selected && (
        <button className="device-remove" onClick={() => removeManual(tv.ip)} aria-label={t('device.removeIp', { ip: tv.ip })} title={t('device.remove')}>
          <X size={12} />
        </button>
      )}
    </div>
  )
}
