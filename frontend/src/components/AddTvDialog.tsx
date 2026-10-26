import { FormEvent, useState } from 'react'
import { MonitorPlay } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { Modal } from './Modal'

const IPV4 = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/

/** Add a TV by IP when mDNS discovery can't see it (other subnet, VPN…). */
export function AddTvDialog({ onAdd, onClose }: { onAdd: (ip: string) => void; onClose: () => void }) {
  const [ip, setIp] = useState('')
  const [err, setErr] = useState('')
  const { t } = useI18n()

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const v = ip.trim()
    if (!IPV4.test(v)) {
      setErr(t('addTv.invalidIp'))
      return
    }
    onAdd(v)
    onClose()
  }

  return (
    <Modal title={t('addTv.title')} icon={<MonitorPlay size={20} />} onClose={onClose}>
      <form className="dialog-body" onSubmit={submit}>
        <p>{t('addTv.help')}</p>
        <label className="field">
          <span>{t('addTv.ipLabel')}</span>
          <input
            autoFocus
            value={ip}
            inputMode="decimal"
            placeholder="192.168.0.18"
            className={err ? 'invalid shake' : ''}
            onChange={(e) => {
              setIp(e.target.value)
              setErr('')
            }}
          />
        </label>
        {err && <div className="err">{err}</div>}
        <div className="actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary">
            {t('addTv.connect')}
          </button>
        </div>
      </form>
    </Modal>
  )
}
