import { ReactNode, useState } from 'react'
import { Command, Plus, RefreshCw, Settings } from 'lucide-react'
import opticSymbol from '../assets/optic-symbol.svg'
import { useRemote } from '../hooks/useRemote'
import { AddTvDialog } from './AddTvDialog'
import { DeviceCard } from './DeviceCard'

type Props = {
  launcher: ReactNode
  onOpenSettings: () => void
  onOpenCommands: () => void
}

export function DeviceSidebar({ launcher, onOpenSettings, onOpenCommands }: Props) {
  const { tvs, scanning, scan, addManual } = useRemote()
  const [adding, setAdding] = useState(false)

  return (
    <aside className="glass side enter" style={{ animationDelay: '40ms' }}>
      <div className="brand">
        <img className="brand-logo" src={opticSymbol} alt="" width={36} height={36} />
        <h1>
          OPTIC<small>Control your TV from your PC</small>
        </h1>
      </div>

      <section className="side-section">
        <div className="label-row">
          <span className="label">TVs na rede</span>
          <button className="link" onClick={scan} disabled={scanning} aria-label="Buscar TVs">
            <RefreshCw size={12} className={scanning ? 'spin' : ''} />
            {scanning ? 'buscando' : 'buscar'}
          </button>
        </div>

        <div className="devices">
          {tvs.length === 0 && !scanning && <p className="empty">Nenhuma TV encontrada.</p>}
          {scanning && tvs.length === 0 && <div className="skeleton" />}
          {tvs.map((tv, i) => (
            <DeviceCard key={tv.ip} tv={tv} index={i} />
          ))}
        </div>

        <button className="add-tv" onClick={() => setAdding(true)}>
          <Plus size={14} /> Adicionar TV
        </button>
      </section>

      {launcher}

      <footer className="side-foot">
        <button className="foot-btn" onClick={onOpenSettings}>
          <Settings size={15} /> Configurações
        </button>
        <button className="foot-btn kbd-btn" onClick={onOpenCommands} title="Comandos rápidos">
          <Command size={13} /> K
        </button>
      </footer>

      {adding && <AddTvDialog onAdd={addManual} onClose={() => setAdding(false)} />}
    </aside>
  )
}
