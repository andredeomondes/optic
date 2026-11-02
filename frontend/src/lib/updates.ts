// Pure helpers for the in-app update UI. Type-only imports: runs under node --test.
import type { Key } from './i18n.ts'

export type UpdateStateName = 'idle' | 'downloading' | 'verifying' | 'installing' | 'ready' | 'error'
export type UpdateKindName = 'installer' | 'portable' | 'dmg' | 'tarball' | ''

/** Whole percent of a download (0–100), or null while the size is unknown. */
export function updatePercent(got: number, total: number): number | null {
  if (!(total > 0) || !Number.isFinite(got)) return null
  return Math.max(0, Math.min(100, Math.floor((got * 100) / total)))
}

/** True while an update is running, so "Atualizar agora" stays disabled. */
export function isUpdateBusy(state: UpdateStateName): boolean {
  return state === 'downloading' || state === 'verifying' || state === 'installing'
}

/** Text for an update state; Windows installs close the app, macOS/Linux leave a step to the user. */
export function updateStateKey(state: UpdateStateName, kind: UpdateKindName): Key | null {
  const windows = kind === 'installer' || kind === 'portable'
  switch (state) {
    case 'downloading':
      return 'update.state.downloading'
    case 'verifying':
      return 'update.state.verifying'
    case 'installing':
      return windows ? 'update.state.installingWindows' : 'update.state.opening'
    case 'ready':
      return kind === 'tarball' ? 'update.state.readyFolder' : kind === 'dmg' ? 'update.state.readyInstaller' : 'update.state.ready'
    case 'error':
      return 'update.state.error'
    default:
      return null
  }
}

export interface UpdateResultLike {
  version: string
  ok: boolean
}

/** The notice shown once at startup for the previous update, if any. */
export function lastResultNotice(r: UpdateResultLike | null | undefined): { kind: 'ok' | 'err'; key: Key; vars?: Record<string, string> } | null {
  if (!r) return null
  if (r.ok) return { kind: 'ok', key: 'update.done', vars: { version: r.version.replace(/^v/, '') } }
  return { kind: 'err', key: 'update.notInstalled' }
}
