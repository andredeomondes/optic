import { ReactNode } from 'react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { KEY_HINT, NAMED_KEYS } from '../lib/shortcuts'

type Props = {
  tvKey: string
  label: string
  children: ReactNode
  className?: string
  /** Show the keyboard shortcut under the label. */
  showHint?: boolean
}

/** A button that sends one TV key and animates pressed / ok / error. */
export function RemoteButton({ tvKey, label, children, className = '', showHint }: Props) {
  const { press, pressedKey, feedback, connected } = useRemote()
  const fb = feedback[`key:${tvKey}`]
  const { t } = useI18n()
  const raw = KEY_HINT[tvKey]
  const hint = raw && raw in NAMED_KEYS ? t(NAMED_KEYS[raw]) : raw

  return (
    <button
      className={`btn ${className} ${pressedKey === tvKey ? 'press' : ''} ${fb ? `fb-${fb}` : ''}`}
      onClick={() => press(tvKey)}
      aria-label={label}
      title={hint ? `${label} · ${hint}` : label}
      disabled={!connected}
    >
      {children}
      {showHint && hint && <kbd className="btn-kbd">{hint}</kbd>}
    </button>
  )
}
