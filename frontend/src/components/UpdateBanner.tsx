import { Sparkles } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import type { Updates } from '../hooks/useUpdates'
import * as api from '../lib/backend'
import { safeReleaseURL } from '../lib/version'

/** Discreet strip above the remote when a newer release is published. */
export function UpdateBanner({ updates }: { updates: Updates }) {
  const { release, available, skipped, skip } = updates
  const { t } = useI18n()
  if (!release || !available || skipped) return null
  return (
    <div className="update-banner" role="status">
      <Sparkles size={14} />
      <span className="msg">{t('update.available', { tag: release.tag })}</span>
      <button className="link" onClick={() => api.BrowserOpenURL(safeReleaseURL(release.url))}>
        {t('update.whatsNew')}
      </button>
      <button className="link muted" onClick={() => skip(release.tag)}>
        {t('update.skip')}
      </button>
    </div>
  )
}
