import { useCallback, useState } from 'react'
import { AppLauncher } from './components/AppLauncher'
import { DeviceSidebar } from './components/DeviceSidebar'
import { PairDialog } from './components/PairDialog'
import { QuickCommand } from './components/QuickCommand'
import { RemoteControl } from './components/RemoteControl'
import { SettingsPanel } from './components/SettingsPanel'
import { TEXT_INPUT_ID } from './components/TextInput'
import { TitleBar } from './components/TitleBar'
import { ToastHost } from './components/Toast'
import { UpdateBanner } from './components/UpdateBanner'
import { useApps } from './hooks/useApps'
import { RemoteProvider, useRemote } from './hooks/useRemote'
import { SleepTimerProvider, useSleepTimer } from './hooks/useSleepTimer'
import { useShortcuts } from './hooks/useShortcuts'
import { useUpdates } from './hooks/useUpdates'
import { useWindowMode } from './hooks/useWindowMode'
import { launchLink } from './lib/apps'
import type { Command } from './lib/commands'

export default function App() {
  return (
    <RemoteProvider>
      <SleepTimerProvider>
        <Shell />
      </SleepTimerProvider>
    </RemoteProvider>
  )
}

function Shell() {
  const remote = useRemote()
  const sleep = useSleepTimer()
  const apps = useApps()
  const win = useWindowMode()
  const updates = useUpdates()
  const [palette, setPalette] = useState(false)
  const [settings, setSettings] = useState(false)

  const togglePalette = useCallback(() => setPalette((p) => !p), [])
  useShortcuts({ connected: remote.connected, press: remote.press, togglePalette, paste: remote.pasteToTV })

  const runCommand = (c: Command) => {
    const a = c.action
    if (a.type === 'ui') {
      if (a.target === 'settings') setSettings(true)
      if (a.target === 'scan') remote.scan()
      if (a.target === 'focus-text') setTimeout(() => document.getElementById(TEXT_INPUT_ID)?.focus(), 50)
      if (a.target === 'paste') {
        if (remote.connected) remote.pasteToTV()
        else remote.notify('err', 'Conecte uma TV primeiro.')
      }
      return
    }
    if (a.type === 'sleep' && a.minutes === 0) {
      if (sleep.deadline === null) {
        remote.notify('info', 'Nenhum timer ativo')
      } else {
        sleep.cancel()
        remote.notify('ok', 'Timer cancelado')
      }
      return
    }
    if (!remote.connected) {
      remote.notify('err', 'Conecte uma TV primeiro.')
      return
    }
    if (a.type === 'sleep') {
      sleep.start(a.minutes)
      remote.notify('ok', `A TV desliga em ${a.minutes} min`)
    }
    if (a.type === 'key') remote.press(a.key).then((ok) => ok && remote.notify('ok', `Comando enviado: ${c.title}`))
    if (a.type === 'launch') remote.launch(a.app.id, launchLink(a.app), a.app.name)
    if (a.type === 'link') remote.launch('link', a.url, a.label)
    if (a.type === 'volume') remote.setVolume(a.level)
    if (a.type === 'text') remote.sendText(a.text)
    if (a.type === 'toast') remote.sendToast(a.text)
  }

  return (
    <div className={`shell ${win.compact ? 'compact' : ''}`}>
      <div className="blob b1" />
      <div className="blob b2" />
      <div className="blob b3" />

      <TitleBar
        compact={win.compact}
        pinned={win.pinned}
        maximised={win.maximised}
        fullscreen={win.fullscreen}
        onToggleCompact={win.toggleCompact}
        onTogglePin={win.togglePin}
        onToggleMaximise={win.toggleMaximise}
        onToggleFullscreen={win.toggleFullscreen}
      />

      {!win.compact && <UpdateBanner updates={updates} />}

      {win.compact ? (
        <main className="app compact" key="compact">
          <RemoteControl compact />
        </main>
      ) : (
        <main className="app" key="normal">
          <DeviceSidebar
            launcher={<AppLauncher apps={apps} />}
            onOpenSettings={() => setSettings(true)}
            onOpenCommands={() => setPalette(true)}
          />
          <RemoteControl />
        </main>
      )}

      <ToastHost />

      {palette && <QuickCommand apps={apps.catalog} onRun={runCommand} onClose={() => setPalette(false)} />}
      {settings && <SettingsPanel updates={updates} onClose={() => setSettings(false)} />}
      {remote.status === 'pairing' && <PairDialog mode={remote.pairingMode} onSubmit={remote.finishPairing} onCancel={remote.cancelPairing} />}
    </div>
  )
}
