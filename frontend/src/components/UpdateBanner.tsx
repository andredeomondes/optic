import { Sparkles } from 'lucide-react'
import type { Updates } from '../hooks/useUpdates'
import * as api from '../lib/backend'

/** Discreet strip above the remote when a newer release is published. */
export function UpdateBanner({ updates }: { updates: Updates }) {
  const { release, available, skipped, skip } = updates
  if (!release || !available || skipped) return null
  return (
    <div className="update-banner" role="status">
      <Sparkles size={14} />
      <span className="msg">Nova versão {release.tag} disponível</span>
      <button className="link" onClick={() => release.url && api.BrowserOpenURL(release.url)}>
        Ver novidades
      </button>
      <button className="link muted" onClick={() => skip(release.tag)}>
        Pular esta versão
      </button>
    </div>
  )
}
