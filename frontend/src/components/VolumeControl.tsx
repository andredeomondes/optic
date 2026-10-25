import { useEffect, useState } from 'react'
import { ChevronDown, ChevronUp, Minus, Plus, Volume2, VolumeX } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

/** Volume −/slider/+, mute, and the channel rocker. */
export function VolumeControl() {
  const { tvState, info, connected, setVolume, feedback } = useRemote()
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
        <RemoteButton tvKey="VOLUME_DOWN" label="Volume −" className="ghost sm">
          <Minus size={16} />
        </RemoteButton>
        <div className="slider-wrap" title={known
            ? 'Arraste para ajustar'
            : info?.platform === 'roku'
              ? 'Esta TV não informa o volume: use + e −'
              : 'Volume aparece após a primeira mudança'}>
          <input
            type="range"
            min={0}
            max={max}
            step={1}
            value={shown}
            disabled={!connected || !known}
            aria-label="Volume"
            style={{ ['--pct' as any]: `${muted ? 0 : pct}%` }}
            onChange={(e) => setDraft(Number(e.target.value))}
            onPointerUp={() => draft !== null && draft !== level && setVolume(draft)}
            onKeyUp={() => draft !== null && draft !== level && setVolume(draft)}
          />
        </div>
        <span className="vol-num" key={level}>
          {known ? (muted ? '—' : shown) : '·'}
        </span>
        <RemoteButton tvKey="VOLUME_UP" label="Volume +" className="ghost sm">
          <Plus size={16} />
        </RemoteButton>
        <RemoteButton tvKey="MUTE" label={muted ? 'Ativar som' : 'Mudo'} className={`ghost sm mute ${muted ? 'muted' : ''}`}>
          {muted ? <VolumeX size={16} /> : <Volume2 size={16} />}
        </RemoteButton>
      </div>

      <div className="ch-rocker">
        <RemoteButton tvKey="CHANNEL_UP" label="Canal +" className="ghost">
          <ChevronUp size={18} />
        </RemoteButton>
        <span>CH</span>
        <RemoteButton tvKey="CHANNEL_DOWN" label="Canal −" className="ghost">
          <ChevronDown size={18} />
        </RemoteButton>
      </div>
    </div>
  )
}
