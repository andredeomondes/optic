import { FormEvent, useState } from 'react'
import { KeyRound, LoaderCircle } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { errText } from '../lib/backend'
import { Modal } from './Modal'

type Props = { mode: 'code' | 'prompt'; onSubmit: (code: string) => Promise<void>; onCancel: () => void }

export function PairDialog({ mode, onSubmit, onCancel }: Props) {
  return mode === 'prompt' ? <PromptDialog onCancel={onCancel} /> : <CodeDialog onSubmit={onSubmit} onCancel={onCancel} />
}

/** The TV asks for confirmation on screen; the backend call is already running. */
function PromptDialog({ onCancel }: { onCancel: () => void }) {
  const { t } = useI18n()
  // "{allow}" marks where the bold word goes.
  const [before, after] = t('pair.prompt').split('{allow}')
  return (
    <Modal title={t('pair.title')} icon={<KeyRound size={20} />} onClose={onCancel}>
      <div className="dialog-body">
        <p>
          {before}
          <strong>{t('pair.allow')}</strong>
          {after}
        </p>
        <div className="pair-wait">
          <LoaderCircle size={22} className="spin" />
        </div>
        <div className="actions">
          <button type="button" className="btn" onClick={onCancel}>
            {t('common.cancel')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

function CodeDialog({ onSubmit, onCancel }: Omit<Props, 'mode'>) {
  const [code, setCode] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const { t } = useI18n()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (code.length !== 6) {
      setErr(t('pair.codeLength'))
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
    <Modal title={t('pair.title')} icon={<KeyRound size={20} />} onClose={onCancel}>
      <form className="dialog-body" onSubmit={submit}>
        <p>{t('pair.codeHelp')}</p>
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
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={busy}>
            {busy ? <LoaderCircle size={16} className="spin" /> : t('pair.submit')}
          </button>
        </div>
      </form>
    </Modal>
  )
}
