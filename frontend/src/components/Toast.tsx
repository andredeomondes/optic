import { CircleAlert, CircleCheck, Info, X } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'

/** Bottom-center notification for action results and errors. */
export function ToastHost() {
  const { toast, dismissToast } = useRemote()
  const { t } = useI18n()
  if (!toast) return null
  const Icon = toast.kind === 'ok' ? CircleCheck : toast.kind === 'err' ? CircleAlert : Info
  return (
    <div className={`toast ${toast.kind}`} key={toast.id} role="status" aria-live="polite">
      <Icon size={16} />
      <span>{toast.text}</span>
      <button className="icon-btn" onClick={dismissToast} aria-label={t('toast.close')}>
        <X size={12} />
      </button>
    </div>
  )
}
