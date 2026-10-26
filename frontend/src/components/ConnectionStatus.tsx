import { LoaderCircle, Pause, Play, Power, RefreshCw } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { since, useNow } from '../hooks/useNow'
import { useRemote } from '../hooks/useRemote'
import { useTVApps } from '../hooks/useTVApps'
import { CATALOG } from '../lib/apps'
import { can, platformOSLabel } from '../lib/capabilities'
import { nowPlayingText, playingKey } from '../lib/nowplaying'
import { InputPicker } from './InputPicker'
import { RemoteButton } from './RemoteButton'
import { SleepTimer } from './SleepTimer'

const CATALOG_NAMES = CATALOG.map(({ pkg, name }) => ({ pkg, name }))

/** Remote header: TV name, live status line and the power / reconnect button. */
export function ConnectionStatus() {
  const { status, pairingMode, host, info, tvs, error, connectedAt, connect, tvState } = useRemote()
  const now = useNow()
  const { t } = useI18n()
  const { apps: installed } = useTVApps(host || null, can(info, 'appList'))
  const pk = playingKey(tvState?.playing ?? '')
  const playingText = pk ? t(pk) : ''
  const nowText =
    status === 'connected' && tvState?.on !== false
      ? nowPlayingText(tvState?.app ?? '', playingText, info?.platform || undefined, { installed, catalog: CATALOG_NAMES })
      : ''
  const name = info ? `${info.vendor} ${info.model}` : tvs.find((t) => t.ip === host)?.name || host

  return (
    <div className="top">
      <div className="tv-name">
        <div className="tv-title">
          {status !== 'idle' && <span className={`dot ${status === 'connected' ? 'live' : status === 'error' ? 'bad' : 'off'}`} />}
          {status === 'idle' ? t('status.pickTv') : name}
        </div>
        <small className={`status-line ${status}`} key={status}>
          {status === 'connected' && (
            <>
              <span>{t('common.connected')} {connectedAt ? since(connectedAt, now, t) : ''}</span>
              {(info?.version || info?.platform === 'tizen') && <span className="chip">{platformOSLabel(info.platform, info.version)}</span>}
              <span className="chip">{host}</span>
            </>
          )}
          {status === 'connecting' && (
            <>
              <LoaderCircle size={12} className="spin" /> {t('common.connecting')}
            </>
          )}
          {status === 'pairing' && (pairingMode === 'prompt' ? t('common.confirmOnTv') : t('status.pairing'))}
          {status === 'error' && (error || t('status.error'))}
          {status === 'idle' && t('status.idle')}
        </small>
        {nowText && (
          <div className="now-playing" title={nowText}>
            {tvState?.playing === 'playing' ? <Play size={12} /> : tvState?.playing === 'paused' ? <Pause size={12} /> : null}
            <span>{nowText}</span>
          </div>
        )}
      </div>
      {status === 'error' && host ? (
        <div className="top-actions">
          <SleepTimer />
          <button className="btn retry" onClick={() => connect(host)}>
            <RefreshCw size={14} /> {t('status.reconnect')}
          </button>
        </div>
      ) : (
        <div className="top-actions">
          <InputPicker />
          <SleepTimer />
          <RemoteButton tvKey="POWER" label={t('status.power')} className="power">
            <Power size={18} strokeWidth={1.8} />
          </RemoteButton>
        </div>
      )}
    </div>
  )
}
