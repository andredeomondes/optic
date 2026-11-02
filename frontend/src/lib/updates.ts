// Pure helpers for the in-app update UI. Imports only lib files without local imports: runs under node --test.
import { translateError } from './errors.ts'
import { translate, type Key, type Lang } from './i18n.ts'

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
  error?: string
}

export interface UpdateNotice {
  kind: 'ok' | 'err'
  text: string
  /** The script's own reason, shown as a second line. */
  detail?: string
}

// Reasons written by the Windows update script (internal/update/apply.go).
const NOT_CLOSED = 'O OPTIC não fechou a tempo.'
const SCRIPT_CHECKSUM = 'checksum'
const CHECKSUM_TEXT = 'O download não confere (checksum). Tente de novo.'
const MAX_DETAIL = 160

/** The notice shown once at startup for the previous update, if any. */
export function lastResultNotice(r: UpdateResultLike | null | undefined, lang: Lang): UpdateNotice | null {
  if (!r) return null
  if (r.ok) return { kind: 'ok', text: translate(lang, 'update.done', { version: r.version.replace(/^v/, '') }) }
  const reason = (r.error ?? '').trim()
  if (reason === '') return { kind: 'err', text: translate(lang, 'update.notInstalledGeneric') }
  if (reason === NOT_CLOSED) return { kind: 'err', text: translate(lang, 'update.notClosed') }
  if (reason === SCRIPT_CHECKSUM) return { kind: 'err', text: translateError(CHECKSUM_TEXT, lang) }
  // UAC refused, Smart App Control, installer exit code, a failed move…
  const detail = reason.length > MAX_DETAIL ? `${reason.slice(0, MAX_DETAIL - 1)}…` : reason
  return { kind: 'err', text: translate(lang, 'update.notInstalled'), detail }
}
