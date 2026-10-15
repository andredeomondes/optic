import { FormEvent, useState } from 'react'
import { Check, LayoutGrid, Pencil, Plus, X } from 'lucide-react'
import { AppIcon } from '../BrandLogos'
import type { AppsApi } from '../hooks/useApps'
import { useRemote } from '../hooks/useRemote'
import { AppDef, isPackageName, launchLink } from '../lib/apps'
import { Modal } from './Modal'

/** Favorite app shortcuts. Edit mode removes and drag-reorders them. */
export function AppLauncher({ apps }: { apps: AppsApi }) {
  const { connected, launch, feedback } = useRemote()
  const [editing, setEditing] = useState(false)
  const [adding, setAdding] = useState(false)
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)

  return (
    <section className="side-section launcher">
      <div className="label-row">
        <span className="label">Apps</span>
        <button className="link" onClick={() => setEditing((e) => !e)}>
          {editing ? <Check size={12} /> : <Pencil size={12} />}
          {editing ? 'pronto' : 'editar'}
        </button>
      </div>

      <div className={`app-grid ${editing ? 'editing' : ''}`}>
        {apps.favorites.map((app, i) => {
          const fb = feedback[`app:${app.id}`]
          return (
            <div
              key={app.id}
              className={`app-tile-wrap ${dragOver === i ? 'drop' : ''}`}
              draggable={editing}
              onDragStart={() => setDragFrom(i)}
              onDragOver={(e) => {
                if (dragFrom === null) return
                e.preventDefault()
                setDragOver(i)
              }}
              onDragEnd={() => {
                setDragFrom(null)
                setDragOver(null)
              }}
              onDrop={() => {
                if (dragFrom !== null && dragFrom !== i) apps.move(dragFrom, i)
                setDragFrom(null)
                setDragOver(null)
              }}
              style={{ ['--brand' as any]: app.brand, animationDelay: `${i * 40}ms` }}
            >
              <button
                className={`app-tile ${fb ? `fb-${fb}` : ''}`}
                onClick={() => !editing && launch(app.id, launchLink(app), app.name)}
                disabled={!editing && !connected}
                title={editing ? 'Arraste para reorganizar' : `Abrir ${app.name}`}
              >
                <AppIcon id={app.id} name={app.name} brand={app.brand} size={32} />
              </button>
              <span className="app-label">{app.name}</span>
              {editing && (
                <button className="tile-remove" onClick={() => apps.remove(app.id)} aria-label={`Remover ${app.name}`}>
                  <X size={10} strokeWidth={3} />
                </button>
              )}
            </div>
          )
        })}
        <div className="app-tile-wrap">
          <button className="app-tile add" onClick={() => setAdding(true)} title="Adicionar app" aria-label="Adicionar app">
            <Plus size={18} />
          </button>
          <span className="app-label">Adicionar</span>
        </div>
      </div>

      {adding && <AddAppDialog apps={apps} onClose={() => setAdding(false)} />}
    </section>
  )
}

function AddAppDialog({ apps, onClose }: { apps: AppsApi; onClose: () => void }) {
  const [name, setName] = useState('')
  const [pkg, setPkg] = useState('')
  const [err, setErr] = useState('')
  const available = apps.catalog.filter((a) => !apps.favorites.some((f) => f.id === a.id))

  const addCustom = (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return setErr('Dê um nome ao atalho.')
    if (!isPackageName(pkg.trim())) return setErr('Pacote inválido. Exemplo: com.spotify.tv.android')
    apps.addCustom(name.trim(), pkg.trim())
    onClose()
  }

  return (
    <Modal title="Adicionar app" icon={<LayoutGrid size={20} />} onClose={onClose} className="wide">
      <div className="dialog-body">
        {available.length > 0 && (
          <>
            <span className="label">Sugestões</span>
            <div className="catalog">
              {available.map((a: AppDef) => (
                <button key={a.id} className="catalog-item" onClick={() => apps.add(a.id)} style={{ ['--brand' as any]: a.brand }}>
                  <AppIcon id={a.id} name={a.name} brand={a.brand} size={18} />
                  <span>{a.name}</span>
                  <Plus size={14} className="catalog-plus" />
                </button>
              ))}
            </div>
          </>
        )}

        <form onSubmit={addCustom} className="custom-app">
          <span className="label">Outro app (pelo pacote Android)</span>
          <div className="field-row">
            <input placeholder="Nome" value={name} onChange={(e) => { setName(e.target.value); setErr('') }} />
            <input placeholder="com.exemplo.app" value={pkg} spellCheck={false} onChange={(e) => { setPkg(e.target.value); setErr('') }} />
            <button type="submit" className="btn primary">Adicionar</button>
          </div>
          {err && <div className="err">{err}</div>}
        </form>
      </div>
    </Modal>
  )
}
