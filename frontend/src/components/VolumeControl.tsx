import { useEffect, useState } from 'react'
import { ChevronDown, ChevronUp, Minus, Plus, Volume2, VolumeX } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

/** Volume −/slider/+, mute, and the channel rocker. */
export function VolumeControl() {
  const { tvState, info, connected, setVolume, feedback } = useRemote()
  const { t } = useI18n()
  const max = tvState?.volumeMax || 100
  const level = tvState?.volume ?? 0
  const muted = tvState?.muted ?? false
  // Local value while dragging; the TV's report wins once it arrives.
  const [draft, setDraft] = useState<number | null>(null)
  useEffect(() => setDraft(null), [level])

  const shown = draft ?? level
  const pct = Math.round((shown / max) * 100)
  const known = Boolean(tvState?.volumeMax)

  return (
    <div className="volume-row">
      <div className={`volume-card ${feedback.volume ? `fb-${feedback.volume}` : ''}`}>
        <RemoteButton tvKey="VOLUME_DOWN" label={t('remote.volumeDown')} className="ghost sm">
          <Minus size={16} />
        </RemoteButton>
        <div className="slider-wrap" title={known
            ? t('remote.volDrag')
            : info?.platform === 'roku' || info?.platform === 'tizen'
              ? t('remote.volNoReport')
              : t('remote.volAfterChange')}>
          <input
            type="range"
            min={0}
            max={max}
            step={1}
            value={shown}
            disabled={!connected || !known}
            aria-label={t('remote.volume')}
            style={{ ['--pct' as any]: `${muted ? 0 : pct}%` }}
            onChange={(e) => setDraft(Number(e.target.value))}
            onPointerUp={() => draft !== null && draft !== level && setVolume(draft)}
            onKeyUp={() => draft !== null && draft !== level && setVolume(draft)}
          />
        </div>
        <span className="vol-num" key={level}>
          {known ? (muted ? '—' : shown) : '·'}
        </span>
        <RemoteButton tvKey="VOLUME_UP" label={t('remote.volumeUp')} className="ghost sm">
          <Plus size={16} />
        </RemoteButton>
        <RemoteButton tvKey="MUTE" label={muted ? t('remote.unmute') : t('remote.mute')} className={`ghost sm mute ${muted ? 'muted' : ''}`}>
          {muted ? <VolumeX size={16} /> : <Volume2 size={16} />}
        </RemoteButton>
      </div>

      <div className="ch-rocker">
        <RemoteButton tvKey="CHANNEL_UP" label={t('remote.channelUp')} className="ghost">
          <ChevronUp size={18} />
        </RemoteButton>
        <span>CH</span>
        <RemoteButton tvKey="CHANNEL_DOWN" label={t('remote.channelDown')} className="ghost">
          <ChevronDown size={18} />
        </RemoteButton>
      </div>
    </div>
  )
}
