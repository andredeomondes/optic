import { useEffect, useRef, useState } from 'react'
import { AppWindow, ClipboardList, Copy, ExternalLink, Eraser, Info, Keyboard, RefreshCw, ScrollText, Settings } from 'lucide-react'
import { useRemote } from '../hooks/useRemote'
import type { Updates } from '../hooks/useUpdates'
import * as api from '../lib/backend'
import { platformOSLabel } from '../lib/capabilities'
import { buildReport } from '../lib/report'
import { SHORTCUTS } from '../lib/shortcuts'
import { safeReleaseURL } from '../lib/version'
import { Modal } from './Modal'

// Wails on Windows opens target=_blank links in a bare in-app window, so go through the runtime.
const ISSUE_URL = 'https://github.com/andredeomondes/optic/issues/new?template=tv-report.yml'

type Tab = 'atalhos' | 'janela' | 'diagnostico' | 'atualizacoes' | 'sobre'

/** Configurações: shortcuts, diagnostics (device info + protocol logs), about. */
export function SettingsPanel({ updates, onClose, initialTab = 'atalhos' }: { updates: Updates; onClose: () => void; initialTab?: Tab }) {
  const [tab, setTab] = useState<Tab>(initialTab)

  return (
    <Modal title="Configurações" icon={<Settings size={20} />} onClose={onClose} className="settings">
      <nav className="tabs" role="tablist">
        <TabBtn id="atalhos" tab={tab} setTab={setTab} icon={<Keyboard size={14} />} label="Atalhos" />
        <TabBtn id="janela" tab={tab} setTab={setTab} icon={<AppWindow size={14} />} label="Janela" />
        <TabBtn id="diagnostico" tab={tab} setTab={setTab} icon={<ScrollText size={14} />} label="Diagnóstico" />
        <TabBtn id="atualizacoes" tab={tab} setTab={setTab} icon={<RefreshCw size={14} />} label="Atualizações" />
        <TabBtn id="sobre" tab={tab} setTab={setTab} icon={<Info size={14} />} label="Sobre" />
      </nav>
      <div className="tab-body" key={tab}>
        {tab === 'atalhos' && <Shortcuts />}
        {tab === 'janela' && <WindowPrefs />}
        {tab === 'diagnostico' && <Diagnostics />}
        {tab === 'atualizacoes' && <UpdatesTab updates={updates} />}
        {tab === 'sobre' && <About />}
      </div>
    </Modal>
  )
}

function TabBtn({ id, tab, setTab, icon, label }: { id: Tab; tab: Tab; setTab: (t: Tab) => void; icon: React.ReactNode; label: string }) {
  return (
    <button role="tab" aria-selected={tab === id} className={`tab ${tab === id ? 'active' : ''}`} onClick={() => setTab(id)}>
      {icon} {label}
    </button>
  )
}

function Shortcuts() {
  return (
    <ul className="shortcut-list">
      {SHORTCUTS.map((s) => (
        <li key={s.label}>
          <span>{s.label}</span>
          <span className="keys">
            {s.keys.map((k) => (
              <kbd key={k}>{k}</kbd>
            ))}
          </span>
        </li>
      ))}
      <li className="note">Atalhos de teclado funcionam com a janela do OPTIC em foco e fora do campo de texto.</li>
    </ul>
  )
}

/** Tray behaviour; the preference lives in Go because OnBeforeClose reads it. */
function WindowPrefs() {
  const [tray, setTray] = useState<boolean | null>(null)
  const [closeToTray, setCloseToTray] = useState(false)

  useEffect(() => {
    if (!api.hasRuntime()) return
    api.TrayAvailable().then(setTray).catch(() => setTray(false))
    api.CloseToTray().then(setCloseToTray).catch(() => {})
    return api.onEvent('prefs:closeToTray', setCloseToTray)
  }, [])

  return (
    <ul className="shortcut-list">
      <li>
        <label className="pref" htmlFor="close-to-tray">
          <span>Fechar esconde na bandeja</span>
          <input
            id="close-to-tray"
            type="checkbox"
            className="switch"
            checked={closeToTray}
            disabled={!tray}
            onChange={(e) => api.SetCloseToTray(e.target.checked)}
          />
        </label>
      </li>
      <li className="note">
        {tray === false
          ? 'Ícone na bandeja indisponível neste sistema.'
          : 'Clique no ícone da bandeja para reabrir. Para sair de vez, use “Sair” no menu do ícone.'}
      </li>
      <li className="note">Modo compacto: botão ao lado de minimizar, na barra de título.</li>
    </ul>
  )
}

function Diagnostics() {
  const { log, clearLog, info, host, tvs, status, tvState, notify } = useRemote()
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' })
  }, [log.length])

  const copy = async () => {
    const text = log.map((l) => `${l.time} ${l.dir} ${l.text}`).join('\n')
    try {
      await navigator.clipboard.writeText(text)
      notify('ok', 'Logs copiados')
    } catch {
      notify('err', 'Não foi possível copiar')
    }
  }

  const copyReport = async () => {
    const text = buildReport({ version: __APP_VERSION__, userAgent: navigator.userAgent, tvs, info, host, log })
    try {
      await navigator.clipboard.writeText(text)
      notify('ok', 'Relatório copiado')
    } catch {
      notify('err', 'Não foi possível copiar')
    }
  }

  const openIssue = (e: React.MouseEvent) => {
    e.preventDefault()
    api.BrowserOpenURL(ISSUE_URL)
  }

  const rows: [string, string][] = [
    ['Status', status],
    ['IP', host || '—'],
    ['Fabricante', info?.vendor || '—'],
    ['Modelo', info?.model || '—'],
    ['Sistema', info ? platformOSLabel(info.platform, info.version) : '—'],
    ['Serviço remoto', info?.appVersion || '—'],
    ['Volume', tvState?.volumeMax ? `${tvState.volume} / ${tvState.volumeMax}` : '—'],
  ]

  return (
    <div className="diag">
      <dl className="diag-grid">
        {rows.map(([k, v]) => (
          <div key={k}>
            <dt>{k}</dt>
            <dd>{v}</dd>
          </div>
        ))}
      </dl>
      <div className="label-row">
        <span className="label">Logs do protocolo</span>
        <span className="log-actions">
          <button className="link" onClick={copy}>
            <Copy size={12} /> copiar
          </button>
          <button className="link" onClick={clearLog}>
            <Eraser size={12} /> limpar
          </button>
        </span>
      </div>
      <div className="log">
        {log.length === 0 && <div className="line dsys">Sem eventos ainda.</div>}
        {log.map((l) => (
          <div key={l.id} className={`line d${l.dir === '>' ? 'out' : l.dir === '<' ? 'in' : l.dir === '!' ? 'err' : 'sys'}`}>
            <span className="t">{l.time}</span> {l.dir} {l.text}
          </div>
        ))}
        <div ref={endRef} />
      </div>
      <div className="label-row">
        <span className="label">Testou em outra marca de TV?</span>
        <span className="log-actions">
          <button className="link" onClick={copyReport}>
            <ClipboardList size={12} /> Copiar relatório
          </button>
          <a className="link" href={ISSUE_URL} onClick={openIssue}>
            <ExternalLink size={12} /> Abrir issue de teste
          </a>
        </span>
      </div>
      <p className="note">“&gt;” enviado pelo OPTIC · “&lt;” recebido da TV · protobuf decodificado sem schema.</p>
    </div>
  )
}

function UpdatesTab({ updates }: { updates: Updates }) {
  const { release, available, checking, auto, lastChecked, checkNow, setAuto } = updates
  const when = lastChecked ? new Date(lastChecked).toLocaleString('pt-BR') : 'nunca'
  return (
    <ul className="shortcut-list">
      <li>
        <span>Versão instalada</span>
        <span>{__APP_VERSION__}</span>
      </li>
      <li>
        <span>Última versão publicada</span>
        <span>{release ? (available ? `${release.tag} (nova)` : `${release.tag} (em dia)`) : '—'}</span>
      </li>
      <li>
        <label className="pref" htmlFor="auto-update">
          <span>Verificar automaticamente</span>
          <input id="auto-update" type="checkbox" className="switch" checked={auto} onChange={(e) => setAuto(e.target.checked)} />
        </label>
      </li>
      <li>
        <span>Última verificação: {when}</span>
        <span className="log-actions">
          {available && release && (
            <button className="link" onClick={() => api.BrowserOpenURL(safeReleaseURL(release.url))}>
              <ExternalLink size={12} /> Ver novidades
            </button>
          )}
          <button className="link" onClick={checkNow} disabled={checking}>
            <RefreshCw size={12} /> {checking ? 'Verificando…' : 'Verificar agora'}
          </button>
        </span>
      </li>
      <li className="note">O OPTIC só consulta a página de versões no GitHub; nada é instalado sem você baixar.</li>
    </ul>
  )
}

function About() {
  return (
    <div className="about">
      <p>
        <b>OPTIC</b> — Open PC-to-TV Interface Controller. Controla Android TV / Google TV pela rede local usando o Android TV
        Remote protocol v2.
      </p>
      <p className="note">Projeto open source. Sem nuvem: tudo acontece entre o seu PC e a TV.</p>
    </div>
  )
}
