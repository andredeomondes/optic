import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { RemoteButton } from './RemoteButton'

export function DPad() {
  const { connected } = useRemote()
  const { t } = useI18n()
  return (
    <div className={`dpad ${connected ? 'on' : ''}`}>
      <RemoteButton tvKey="DPAD_UP" label={t('remote.up')} className="arr up">
        <ChevronUp size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_LEFT" label={t('remote.left')} className="arr left">
        <ChevronLeft size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_RIGHT" label={t('remote.right')} className="arr right">
        <ChevronRight size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_DOWN" label={t('remote.down')} className="arr down">
        <ChevronDown size={24} />
      </RemoteButton>
      <RemoteButton tvKey="DPAD_CENTER" label={t('remote.ok')} className="ok">
        {t('remote.ok')}
      </RemoteButton>
    </div>
  )
}
