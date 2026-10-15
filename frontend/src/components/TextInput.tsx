import { FormEvent, useState } from 'react'
import { Keyboard, LoaderCircle, Send, X } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import { OWN_KEYBOARD_APPS } from '../lib/apps'

export const TEXT_INPUT_ID = 'tv-text-input'

/** "Digitar na TV": type with the PC keyboard, Enter sends. */
export function TextInput() {
  const { connected, sendText, feedback, tvState } = useRemote()
  const [text, setText] = useState('')
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
      ? { kind: 'warn', text: `${ownKeyboard} usa teclado próprio: digite com as setas e OK.` }
      : tvState?.keyboard
        ? { kind: 'ok', text: 'Teclado da TV aberto. O texto substitui o que está no campo.' }
        : { kind: 'info', text: 'Abra um campo de texto na TV (ex.: busca) antes de enviar.' }

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
        placeholder={connected ? 'Digite na TV…' : 'Conecte uma TV para digitar'}
        disabled={!connected}
        spellCheck={false}
        autoComplete="off"
        aria-label="Texto para enviar à TV"
      />
      {text && (
        <button type="button" className="icon-btn" onClick={() => setText('')} aria-label="Limpar" title="Limpar · Esc">
          <X size={14} />
        </button>
      )}
      <button type="submit" className="send-btn" disabled={!connected || !text.trim() || fb === 'pending'} aria-label="Enviar" title="Enviar · Enter">
        {fb === 'pending' ? <LoaderCircle size={16} className="spin" /> : <Send size={16} />}
      </button>
    </form>
    {hint && <p className={`type-hint ${hint.kind}`} key={hint.text}>{hint.text}</p>}
    </>
  )
}
