import { FormEvent, useState } from 'react'
import { Keyboard, LoaderCircle, Send, X } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { OWN_KEYBOARD_APPS } from '../lib/apps'

export const TEXT_INPUT_ID = 'tv-text-input'

/** "Type on the TV": type with the PC keyboard, Enter sends. */
export function TextInput() {
  const { connected, sendText, feedback, tvState } = useRemote()
  const [text, setText] = useState('')
  const { t } = useI18n()
  const fb = feedback.text

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!text.trim() || !connected) return
    if (await sendText(text)) setText('')
  }

  const ownKeyboard = tvState?.app ? OWN_KEYBOARD_APPS[tvState.app] : undefined
  const hint = !connected
    ? null
    : ownKeyboard
      ? { kind: 'warn', text: t('text.ownKeyboard', { app: ownKeyboard }) }
      : tvState?.keyboard
        ? { kind: 'ok', text: t('text.keyboardOpen') }
        : { kind: 'info', text: t('text.openField') }

  return (
    <>
    <form className={`type-box ${fb ? `fb-${fb}` : ''}`} onSubmit={submit}>
      <Keyboard size={16} className="type-ic" aria-hidden="true" />
      <input
        id={TEXT_INPUT_ID}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            if (text) setText('')
            else e.currentTarget.blur()
            e.stopPropagation()
          }
        }}
        placeholder={connected ? t('text.placeholder') : t('text.placeholderOff')}
        disabled={!connected}
        spellCheck={false}
        autoComplete="off"
        aria-label={t('text.aria')}
      />
      {text && (
        <button type="button" className="icon-btn" onClick={() => setText('')} aria-label={t('text.clear')} title={t('text.clearTitle')}>
          <X size={14} />
        </button>
      )}
      <button type="submit" className="send-btn" disabled={!connected || !text.trim() || fb === 'pending'} aria-label={t('text.send')} title={t('text.sendTitle')}>
        {fb === 'pending' ? <LoaderCircle size={16} className="spin" /> : <Send size={16} />}
      </button>
    </form>
    {hint && <p className={`type-hint ${hint.kind}`} key={hint.text}>{hint.text}</p>}
    </>
  )
}
