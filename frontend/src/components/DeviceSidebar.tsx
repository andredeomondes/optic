import { ReactNode, useState } from 'react'
import { Command, Plus, RefreshCw, Settings } from 'lucide-react'
import opticSymbol from '../assets/optic-symbol.svg'
import { useI18n } from '../hooks/useI18n'
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
  const { t } = useI18n()
  const [adding, setAdding] = useState(false)

  return (
    <aside className="glass side enter" style={{ animationDelay: '40ms' }}>
      <div className="brand">
        <img className="brand-logo" src={opticSymbol} alt="" width={36} height={36} />
        <h1>
          OPTIC<small>{t('brand.tagline')}</small>
        </h1>
      </div>

      <section className="side-section">
        <div className="label-row">
          <span className="label">{t('sidebar.tvsOnNetwork')}</span>
          <button className="link" onClick={scan} disabled={scanning} aria-label={t('sidebar.scanLabel')}>
            <RefreshCw size={12} className={scanning ? 'spin' : ''} />
            {scanning ? t('sidebar.scanning') : t('sidebar.scan')}
          </button>
        </div>

        <div className="devices">
          {tvs.length === 0 && !scanning && <p className="empty">{t('sidebar.empty')}</p>}
          {scanning && tvs.length === 0 && <div className="skeleton" />}
          {tvs.map((tv, i) => (
            <DeviceCard key={tv.ip} tv={tv} index={i} />
          ))}
        </div>

        <button className="add-tv" onClick={() => setAdding(true)}>
          <Plus size={14} /> {t('sidebar.addTv')}
        </button>
      </section>

      {launcher}

      <footer className="side-foot">
        <button className="foot-btn" onClick={onOpenSettings}>
          <Settings size={15} /> {t('sidebar.settings')}
        </button>
        <button className="foot-btn kbd-btn" onClick={onOpenCommands} title={t('sidebar.quickCommands')}>
          <Command size={13} /> K
        </button>
      </footer>

      {adding && <AddTvDialog onAdd={addManual} onClose={() => setAdding(false)} />}
    </aside>
  )
}
