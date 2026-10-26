import { Cable } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useRemote } from '../hooks/useRemote'
import { can } from '../lib/capabilities'
import type { TVInput } from '../lib/types'

/** "Entrada" button with a menu of the TV's inputs (Roku, LG). */
export function InputPicker() {
  const { connected, info, inputs, switchInput, notify } = useRemote()
  const [open, setOpen] = useState(false)
  const [list, setList] = useState<TVInput[]>([])
  const root = useRef<HTMLDivElement>(null)
  const trigger = useRef<HTMLButtonElement>(null)
  // Bumped on TV change and on each fetch; a stale or duplicate response is dropped.
  const req = useRef(0)
  const fetching = useRef(false)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        trigger.current?.focus()
        return
      }
      if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])
      if (!items.length) return
      e.preventDefault()
      const i = items.indexOf(document.activeElement as HTMLElement)
      const step = e.key === 'ArrowDown' ? 1 : -1
      items[(i + step + items.length) % items.length].focus()
    }
    window.addEventListener('mousedown', onDown)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('mousedown', onDown)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  // Focus the first item once the list has rendered.
  useEffect(() => {
    if (open && list.length) root.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus()
  }, [open, list])

  // A different TV has a different list.
  useEffect(() => {
    req.current++
    fetching.current = false
    setOpen(false)
    setList([])
  }, [info])

  if (!can(info, 'inputs')) return null

  const toggle = async () => {
    if (open) {
      setOpen(false)
      return
    }
    if (fetching.current) return
    const mine = ++req.current
    fetching.current = true
    try {
      const l = await inputs()
      if (req.current !== mine) return
      setList(l)
      setOpen(true)
    } catch {
      if (req.current === mine) notify('err', 'Não foi possível listar as entradas')
    } finally {
      if (req.current === mine) fetching.current = false
    }
  }

  const pick = (i: TVInput) => {
    setOpen(false)
    trigger.current?.focus()
    switchInput(i.id, i.name)
  }

  return (
    <div className="input-pick" ref={root}>
      <button
        ref={trigger}
        className="btn sm ghost"
        onClick={toggle}
        aria-label="Entrada"
        aria-expanded={open}
        aria-haspopup="menu"
        title="Trocar a entrada da TV"
        disabled={!connected}
      >
        <Cable size={16} strokeWidth={1.8} />
      </button>
      {open && (
        <div className="input-menu" role="menu">
          {list.length === 0 ? (
            <span className="input-empty">Nenhuma entrada</span>
          ) : (
            list.map((i) => (
              <button key={i.id} role="menuitem" onClick={() => pick(i)}>
                {i.name}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  )
}
