import { FormEvent, useState } from 'react'
import { KeyRound, LoaderCircle } from 'lucide-react'
import { errText } from '../lib/backend'
import { Modal } from './Modal'

export function PairDialog({ onSubmit, onCancel }: { onSubmit: (code: string) => Promise<void>; onCancel: () => void }) {
  const [code, setCode] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (code.length !== 6) {
      setErr('O código tem 6 caracteres.')
      return
    }
    setBusy(true)
    try {
      await onSubmit(code)
    } catch (e) {
      setErr(errText(e))
      setBusy(false)
    }
  }

  return (
    <Modal title="Parear com a TV" icon={<KeyRound size={20} />} onClose={onCancel}>
      <form className="dialog-body" onSubmit={submit}>
        <p>Digite o código de 6 caracteres que apareceu na tela da TV.</p>
        <input
          autoFocus
          value={code}
          maxLength={6}
          spellCheck={false}
          placeholder="A1B2C3"
          className={`code-input ${err ? 'shake' : ''}`}
          onChange={(e) => {
            setCode(e.target.value.toUpperCase().replace(/[^0-9A-F]/g, ''))
            setErr('')
          }}
        />
        {err && <div className="err">{err}</div>}
        <div className="actions">
          <button type="button" className="btn" onClick={onCancel}>
            Cancelar
          </button>
          <button type="submit" className="btn primary" disabled={busy}>
            {busy ? <LoaderCircle size={16} className="spin" /> : 'Parear'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
