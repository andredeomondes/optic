import { FormEvent, useState } from 'react'
import { Check, LayoutGrid, Pencil, Plus, Star, X } from 'lucide-react'
import { AppIcon } from '../BrandLogos'
import type { AppsApi } from '../hooks/useApps'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { useTVApps } from '../hooks/useTVApps'
import type { Key } from '../lib/i18n'
import { AppDef, appsForPlatform, isPackageName, launchLink } from '../lib/apps'
import { can } from '../lib/capabilities'
import { filterApps } from '../lib/tvapps'
import { Modal } from './Modal'

/** Favorite app shortcuts. Edit mode removes and drag-reorders them. */
export function AppLauncher({ apps }: { apps: AppsApi }) {
  const { t } = useI18n()
  const { connected, launch, feedback, info } = useRemote()
  const favorites = appsForPlatform(apps.favorites, info?.platform || undefined)
  const [editing, setEditing] = useState(false)
  const [adding, setAdding] = useState(false)
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)

  return (
    <section className="side-section launcher">
      <div className="label-row">
        <span className="label">{t('apps.title')}</span>
        <button className="link" onClick={() => setEditing((e) => !e)}>
          {editing ? <Check size={12} /> : <Pencil size={12} />}
          {editing ? t('apps.done') : t('apps.edit')}
        </button>
      </div>

      <div className={`app-grid ${editing ? 'editing' : ''}`}>
        {favorites.map((app, i) => {
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
                if (dragFrom !== null && dragFrom !== i) apps.move(apps.favorites.indexOf(favorites[dragFrom]), apps.favorites.indexOf(favorites[i]))
                setDragFrom(null)
                setDragOver(null)
              }}
              style={{ ['--brand' as any]: app.brand, animationDelay: `${i * 40}ms` }}
            >
              <button
                className={`app-tile ${fb ? `fb-${fb}` : ''}`}
                onClick={() => !editing && launch(app.id, launchLink(app), app.name)}
                disabled={!editing && !connected}
                title={editing ? t('apps.dragReorder') : t('apps.open', { name: app.name })}
              >
                <AppIcon id={app.id} name={app.name} brand={app.brand} size={32} />
              </button>
              <span className="app-label">{app.name}</span>
              {editing && (
                <button className="tile-remove" onClick={() => apps.remove(app.id)} aria-label={t('apps.remove', { name: app.name })}>
                  <X size={10} strokeWidth={3} />
                </button>
              )}
            </div>
          )
        })}
        <div className="app-tile-wrap">
          <button className="app-tile add" onClick={() => setAdding(true)} title={t('apps.add')} aria-label={t('apps.add')}>
            <Plus size={18} />
          </button>
          <span className="app-label">{t('apps.addShort')}</span>
        </div>
      </div>

      {adding && <AddAppDialog apps={apps} onClose={() => setAdding(false)} />}
    </section>
  )
}

/** The TV's own installed apps: search, launch in one click, star to favorite. */
function TVAppsSection({ apps }: { apps: AppsApi }) {
  const { t } = useI18n()
  const { host, info, connected, launch, feedback } = useRemote()
  const { apps: tvApps, loading, error, reload } = useTVApps(host || null, can(info, 'appList'))
  const [query, setQuery] = useState('')
  const shown = filterApps(tvApps, query)
  const platform = info?.platform || undefined
  const favs = appsForPlatform(apps.favorites, platform)

  return (
    <div className="tv-apps">
      <span className="label">{t('apps.tvApps')}</span>
      <input
        placeholder={t('apps.searchPlaceholder')}
        value={query}
        spellCheck={false}
        onChange={(e) => setQuery(e.target.value)}
        aria-label={t('apps.searchAria')}
      />
      {loading ? (
        <div className="tv-apps-list" aria-busy="true">
          <div className="skeleton" style={{ height: 40 }} />
          <div className="skeleton" style={{ height: 40 }} />
          <span className="note">{t('apps.loadingTv')}</span>
        </div>
      ) : error ? (
        <div className="tv-apps-state">
          <span className="err">{t('apps.loadFailed')}</span>
          <button type="button" className="btn" onClick={reload}>{t('apps.retry')}</button>
        </div>
      ) : shown.length === 0 ? (
        <div className="note">{t('apps.none')}</div>
      ) : (
        <div className="tv-apps-list">
          {shown.map((a) => {
            const favId = `tv:${a.id}`
            const fav = favs.some((f) => f.id === favId)
            const fb = feedback[`app:${favId}`]
            return (
              <div key={a.id} className={`tv-app ${fb ? `fb-${fb}` : ''}`} style={{ ['--brand' as any]: '#8B5CF6' }}>
                <button
                  type="button"
                  className="tv-app-open"
                  onClick={() => launch(favId, a.id, a.name)}
                  disabled={!connected}
                  title={t('apps.open', { name: a.name })}
                >
                  <AppIcon id={favId} name={a.name} brand="#8B5CF6" size={16} />
                  <span>{a.name}</span>
                </button>
                <button
                  type="button"
                  className={`tv-app-star ${fav ? 'on' : ''}`}
                  onClick={() => (fav ? apps.remove(favId) : apps.addTV(a, info?.platform || 'roku'))}
                  aria-pressed={fav}
                  aria-label={fav ? t('apps.unfavoriteNamed', { name: a.name }) : t('apps.favoriteNamed', { name: a.name })}
                  title={fav ? t('apps.unfavorite') : t('apps.favorite')}
                >
                  <Star size={14} fill={fav ? 'currentColor' : 'none'} />
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

function AddAppDialog({ apps, onClose }: { apps: AppsApi; onClose: () => void }) {
  const [name, setName] = useState('')
  const [pkg, setPkg] = useState('')
  const [err, setErr] = useState<Key | ''>('')
  const { t } = useI18n()
  const { info } = useRemote()
  const platform = info?.platform || undefined
  const favIds = appsForPlatform(apps.favorites, platform)
  const available = appsForPlatform(apps.catalog, platform).filter((a) => !favIds.some((f) => f.id === a.id))

  const addCustom = (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return setErr('apps.nameRequired')
    if (!isPackageName(pkg.trim())) return setErr('apps.badPackage')
    apps.addCustom(name.trim(), pkg.trim())
    onClose()
  }

  return (
    <Modal title={t('apps.add')} icon={<LayoutGrid size={20} />} onClose={onClose} className="wide">
      <div className="dialog-body">
        {can(info, 'appList') && <TVAppsSection apps={apps} />}
        {available.length > 0 && (
          <>
            <span className="label">{t('apps.suggestions')}</span>
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
          <span className="label">{t('apps.other')}</span>
          <div className="field-row">
            <input placeholder={t('apps.namePlaceholder')} value={name} onChange={(e) => { setName(e.target.value); setErr('') }} />
            <input placeholder={t('apps.pkgPlaceholder')} value={pkg} spellCheck={false} onChange={(e) => { setPkg(e.target.value); setErr('') }} />
            <button type="submit" className="btn primary">{t('apps.addShort')}</button>
          </div>
          {err && <div className="err">{t(err)}</div>}
        </form>
      </div>
    </Modal>
  )
}
