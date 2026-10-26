import { useCallback, useMemo, useState } from 'react'
import type { Platform } from '../lib/types'
import { allApps, AppDef, loadApps, makeCustomApp, makeTVApp, saveApps } from '../lib/apps'

/** Favorite app shortcuts: add, remove, reorder, custom packages. Persisted. */
export function useApps() {
  const [state, setState] = useState(loadApps)

  const update = useCallback((fn: (s: ReturnType<typeof loadApps>) => ReturnType<typeof loadApps>) => {
    setState((s) => {
      const next = fn(s)
      saveApps(next)
      return next
    })
  }, [])

  const catalog = useMemo(() => allApps(state.custom), [state.custom])
  const favorites = useMemo(
    () => state.favorites.map((id) => catalog.find((a) => a.id === id)).filter(Boolean) as AppDef[],
    [state.favorites, catalog],
  )

  return {
    catalog,
    favorites,
    add: (id: string) => update((s) => ({ ...s, favorites: s.favorites.includes(id) ? s.favorites : [...s.favorites, id] })),
    remove: (id: string) =>
      update((s) => ({
        // TV-picked apps live only as favorites, so drop the stored def too
        custom: id.startsWith('tv:') ? s.custom.filter((c) => c.id !== id) : s.custom,
        favorites: s.favorites.filter((f) => f !== id),
      })),
    move: (from: number, to: number) =>
      update((s) => {
        const f = [...s.favorites]
        const [item] = f.splice(from, 1)
        f.splice(to, 0, item)
        return { ...s, favorites: f }
      }),
    addTV: (tv: { id: string; name: string }, platform: Platform) => {
      const app = makeTVApp(tv, platform)
      update((s) => ({
        custom: [...s.custom.filter((c) => c.id !== app.id), app],
        favorites: s.favorites.includes(app.id) ? s.favorites : [...s.favorites, app.id],
      }))
    },
    addCustom: (name: string, pkg: string) => {
      const app = makeCustomApp(name, pkg)
      update((s) => ({
        custom: s.custom.some((c) => c.id === app.id) ? s.custom : [...s.custom, app],
        favorites: s.favorites.includes(app.id) ? s.favorites : [...s.favorites, app.id],
      }))
    },
  }
}

export type AppsApi = ReturnType<typeof useApps>
