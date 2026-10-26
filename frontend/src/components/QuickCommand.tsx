import { useEffect, useMemo, useRef, useState } from 'react'
import { ArrowRight, CornerDownLeft, Search } from 'lucide-react'
import { AppIcon } from '../BrandLogos'
import { AppDef } from '../lib/apps'
import { useRemote } from '../hooks/useRemote'
import { can } from '../lib/capabilities'
import { Command, suggest } from '../lib/commands'
import { CommandIcon } from './icons'

type Props = {
  apps: AppDef[]
  onRun: (c: Command) => void
  onClose: () => void
}

/** Ctrl+K palette: type, ↑↓ to pick, Enter to run, Esc to close. */
export function QuickCommand({ apps, onRun, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [index, setIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)
  const { info } = useRemote()
  const toast = can(info, 'toast')
  const inputs = can(info, 'inputs')
  const platform = info?.platform
  const results = useMemo(() => suggest(query, apps, { toast, inputs, platform }), [query, apps, toast, inputs, platform])

  useEffect(() => setIndex(0), [query])
  useEffect(() => {
    listRef.current?.children[index]?.scrollIntoView({ block: 'nearest' })
  }, [index])

  const run = (c: Command | undefined) => {
    if (!c) return
    onClose()
    onRun(c)
  }

  return (
    <div className="overlay palette-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="glass palette" role="dialog" aria-modal="true" aria-label="Comandos rápidos">
        <div className="palette-input">
          <Search size={16} />
          <input
            autoFocus
            value={query}
            placeholder="Digite um comando… (abrir youtube, volume 30, mudo)"
            spellCheck={false}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'ArrowDown') {
                e.preventDefault()
                setIndex((i) => Math.min(i + 1, results.length - 1))
              } else if (e.key === 'ArrowUp') {
                e.preventDefault()
                setIndex((i) => Math.max(i - 1, 0))
              } else if (e.key === 'Enter') {
                e.preventDefault()
                run(results[index])
              } else if (e.key === 'Escape') {
                e.preventDefault()
                e.stopPropagation()
                onClose()
              }
            }}
          />
          <kbd>Esc</kbd>
        </div>

        <div className="palette-list" ref={listRef} role="listbox">
          {results.length === 0 && <div className="palette-empty">Nenhum comando para “{query}”.</div>}
          {results.map((c, i) => (
            <button
              key={c.id}
              role="option"
              aria-selected={i === index}
              className={`palette-item ${i === index ? 'active' : ''}`}
              onMouseMove={() => setIndex(i)}
              onClick={() => run(c)}
            >
              <span className="palette-ic">
                {c.app ? <AppIcon id={c.app.id} name={c.app.name} brand={c.app.brand} size={16} /> : <CommandIcon name={c.icon} />}
              </span>
              <span className="palette-title">
                {c.title}
                {c.hint && <small>{c.hint}</small>}
              </span>
              {i === index ? <CornerDownLeft size={14} className="palette-go" /> : <ArrowRight size={14} className="palette-go dim" />}
            </button>
          ))}
        </div>

        <div className="palette-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> escolher</span>
          <span><kbd>Enter</kbd> executar</span>
          <span><kbd>Ctrl</kbd><kbd>K</kbd> abrir/fechar</span>
        </div>
      </div>
    </div>
  )
}
