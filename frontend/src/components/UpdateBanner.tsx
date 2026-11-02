import { Download, ExternalLink, Sparkles } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import type { UpdateRun, Updates } from '../hooks/useUpdates'
import * as api from '../lib/backend'
import { translateError } from '../lib/errors'
import { isUpdateBusy, updatePercent, updateStateKey } from '../lib/updates'
import { RELEASES_URL, safeReleaseURL } from '../lib/version'

/** Discreet strip above the remote when a newer release is published. */
export function UpdateBanner({ updates }: { updates: Updates }) {
  const { release, available, skipped, skip, run } = updates
  const { t } = useI18n()
  const active = run.state !== 'idle'
  if (!release || !available || (skipped && !active)) return null
  return (
    <div className="update-banner" role="status">
      <div className="update-row">
        <Sparkles size={14} />
        <span className="msg">{t('update.available', { tag: release.tag })}</span>
        <UpdateNowButton updates={updates} />
        <button className="link" onClick={() => api.BrowserOpenURL(safeReleaseURL(release.url))}>
          {t('update.whatsNew')}
        </button>
        {!active && (
          <button className="link muted" onClick={() => skip(release.tag)}>
            {t('update.skip')}
          </button>
        )}
      </div>
      <UpdateStatus run={run} />
    </div>
  )
}

/** "Atualizar agora": only when Go says this release can be installed here; disabled while running. */
export function UpdateNowButton({ updates }: { updates: Updates }) {
  const { t } = useI18n()
  const { support, run, startUpdate } = updates
  if (!support?.auto) return null
  return (
    <button className="link strong" onClick={startUpdate} disabled={isUpdateBusy(run.state)}>
      <Download size={12} /> {t('update.now')}
    </button>
  )
}

/** Progress bar and state text of a running or finished update. */
export function UpdateStatus({ run }: { run: UpdateRun }) {
  const { t, lang } = useI18n()
  const key = updateStateKey(run.state, run.kind)
  if (!key) return null
  const pct = run.state === 'downloading' ? updatePercent(run.got, run.total) : null
  const text = run.state === 'error' && run.error ? translateError(run.error, lang) : t(key)
  return (
    <div className={`update-status ${run.state}`} aria-live="polite">
      {run.state === 'downloading' && (
        <div className="update-progress">
          <div
            className="bar"
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={pct ?? undefined}
            aria-label={t('update.state.downloading')}
          >
            <span style={{ width: `${pct ?? 0}%` }} />
          </div>
          <span className="pct">{pct === null ? '…' : `${pct}%`}</span>
        </div>
      )}
      <span className="update-text">{text}</span>
      {run.state === 'error' && (
        <button className="link" onClick={() => api.BrowserOpenURL(RELEASES_URL)}>
          <ExternalLink size={12} /> {t('update.openReleases')}
        </button>
      )}
    </div>
  )
}
