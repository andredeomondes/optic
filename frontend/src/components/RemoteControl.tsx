import { Fragment, useEffect, useMemo, useRef } from 'react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import { can } from '../lib/capabilities'
import type { Key } from '../lib/i18n'
import { FOOTER_HINTS, SHORTCUTS } from '../lib/shortcuts'
import { createThrottle, wheelVolumeStep } from '../lib/throttle'
import { ConnectionStatus } from './ConnectionStatus'
import { DPad } from './DPad'
import { MediaControls, NavButtons } from './MediaControls'
import { TextInput } from './TextInput'
import { VolumeControl } from './VolumeControl'

const WHEEL_THROTTLE_MS = 120

/**
 * The main remote. `compact` keeps only D-pad, navigation, volume and media,
 * ready for a future compact window.
 */
export function RemoteControl({ compact = false }: { compact?: boolean }) {
  const { info, connected, press } = useRemote()
  const { t } = useI18n()
  // One gate per panel, so its window survives re-renders.
  const wheelGate = useMemo(() => createThrottle(WHEEL_THROTTLE_MS), [])

  // Read by the native wheel listener, which is registered once per panel.
  const latest = useRef({ connected, press })
  latest.current = { connected, press }
  const sectionRef = useRef<HTMLElement>(null)

  // While connected, wheel over the panel changes the TV volume and does not
  // scroll the panel. A native listener is needed: React's onWheel is passive,
  // so preventDefault would be ignored.
  useEffect(() => {
    const el = sectionRef.current
    if (!el) return
    const onWheel = (e: WheelEvent) => {
      const { connected, press } = latest.current
      if (!connected) return
      const step = wheelVolumeStep(e)
      if (!step) return
      e.preventDefault()
      if (wheelGate()) press(step)
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [wheelGate])

  return (
    <section
      className={`glass remote enter ${compact ? 'compact' : ''}`}
      style={{ animationDelay: '120ms' }}
      ref={sectionRef}
    >
      {!compact && <ConnectionStatus />}
      <VolumeControl />
      <DPad />
      <NavButtons />
      <MediaControls />
      {!compact && (
        <>
          {can(info, 'text') && (
            <>
              <div className="section-label">{t('remote.typeOnTv')}</div>
              <TextInput />
            </>
          )}
          <ShortcutHints />
        </>
      )}
    </section>
  )
}

// Footer hints: shortcut id -> translated lowercase text.
const HINT_KEY: Record<(typeof FOOTER_HINTS)[number], Key> = {
  navigate: 'remote.hint.navigate',
  ok: 'remote.hint.ok',
  back: 'remote.hint.back',
  home: 'remote.hint.home',
  mute: 'remote.hint.mute',
  palette: 'remote.hint.quick',
}

function ShortcutHints() {
  const { t } = useI18n()
  const shown = FOOTER_HINTS.map((id) => SHORTCUTS.find((s) => s.id === id)).filter((s): s is (typeof SHORTCUTS)[number] => !!s)
  return (
    <div className="hint">
      {shown.map((s, i) => (
        <Fragment key={s.id}>
          {i > 0 && ' · '}
          {s.keys.map((k) => (
            <kbd key={k}>{k}</kbd>
          ))}{' '}
          {t(HINT_KEY[s.id as (typeof FOOTER_HINTS)[number]])}
        </Fragment>
      ))}
    </div>
  )
}
