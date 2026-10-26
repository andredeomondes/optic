import { House, Pause, Play, Rewind, FastForward, Settings, SkipBack, SkipForward, Undo2 } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

const ICON = { size: 18, strokeWidth: 1.8 }

/** Back / Home / Settings with their keyboard hints. */
export function NavButtons() {
  const { t } = useI18n()
  return (
    <div className="row3">
      <RemoteButton tvKey="BACK" label={t('remote.back')} className="labeled" showHint>
        <Undo2 {...ICON} /> {t('remote.back')}
      </RemoteButton>
      <RemoteButton tvKey="HOME" label={t('remote.home')} className="labeled" showHint>
        <House {...ICON} /> {t('remote.home')}
      </RemoteButton>
      <RemoteButton tvKey="SETTINGS" label={t('remote.tvSettings')} className="labeled">
        <Settings {...ICON} /> {t('remote.settings')}
      </RemoteButton>
    </div>
  )
}

/** LG and Samsung have no previous/next keys, so those two are hidden on webOS and Tizen. */
export function MediaControls() {
  const { info } = useRemote()
  const { t } = useI18n()
  const skip = info?.platform !== 'webos' && info?.platform !== 'tizen'
  return (
    <div className="media-row" style={skip ? undefined : { gridTemplateColumns: 'repeat(3, 1fr)' }}>
      {skip && (
        <RemoteButton tvKey="MEDIA_PREVIOUS" label={t('remote.previous')}>
          <SkipBack {...ICON} />
        </RemoteButton>
      )}
      <RemoteButton tvKey="MEDIA_REWIND" label={t('remote.rewind')}>
        <Rewind {...ICON} />
      </RemoteButton>
      <RemoteButton tvKey="MEDIA_PLAY_PAUSE" label={t('remote.playPause')} className="play">
        <span className="pp">
          <Play size={15} strokeWidth={1.8} />
          <Pause size={15} strokeWidth={1.8} />
        </span>
      </RemoteButton>
      <RemoteButton tvKey="MEDIA_FAST_FORWARD" label={t('remote.forward')}>
        <FastForward {...ICON} />
      </RemoteButton>
      {skip && (
        <RemoteButton tvKey="MEDIA_NEXT" label={t('remote.next')}>
          <SkipForward {...ICON} />
        </RemoteButton>
      )}
    </div>
  )
}
