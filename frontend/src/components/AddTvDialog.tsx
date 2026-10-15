import { FormEvent, useState } from 'react'
import { MonitorPlay } from 'lucide-react'
import { Modal } from './Modal'

const IPV4 = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/

/** Add a TV by IP when mDNS discovery can't see it (other subnet, VPN…). */
export function AddTvDialog({ onAdd, onClose }: { onAdd: (ip: string) => void; onClose: () => void }) {
  const [ip, setIp] = useState('')
  const [err, setErr] = useState('')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const v = ip.trim()
    if (!IPV4.test(v)) {
      setErr('Digite um IP válido, como 192.168.0.18.')
      return
    }
    onAdd(v)
    onClose()
  }

  return (
    <Modal title="Adicionar TV" icon={<MonitorPlay size={20} />} onClose={onClose}>
      <form className="dialog-body" onSubmit={submit}>
        <p>Use quando a TV não aparece na busca. Ela precisa estar ligada e com o Google TV ativo.</p>
        <label className="field">
          <span>Endereço IP</span>
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
            Cancelar
          </button>
          <button type="submit" className="btn primary">
            Conectar
          </button>
        </div>
      </form>
    </Modal>
  )
}
