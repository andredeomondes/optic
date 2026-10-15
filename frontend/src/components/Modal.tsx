import { ReactNode, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'

type Props = {
  title?: string
  icon?: ReactNode
  onClose: () => void
  children: ReactNode
  className?: string
}

/**
 * Glass dialog over a dimmed backdrop. Esc or a backdrop click closes it.
 * Rendered into <body> through a portal: panels use backdrop-filter, which
 * would otherwise clip an absolutely positioned overlay to the panel.
 */
export function Modal({ title, icon, onClose, children, className = '' }: Props) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [onClose])

  return createPortal(
    <div className="overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className={`glass dialog ${className}`} role="dialog" aria-modal="true" aria-label={title}>
        {title && (
          <header className="dialog-head">
            {icon && <div className="dialog-ic">{icon}</div>}
            <h2>{title}</h2>
            <button className="icon-btn" onClick={onClose} aria-label="Fechar">
              <X size={16} />
            </button>
          </header>
        )}
        {children}
      </div>
    </div>,
    document.body,
  )
}
