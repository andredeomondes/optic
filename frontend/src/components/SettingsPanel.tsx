import { useEffect, useRef, useState } from 'react'
import { AppWindow, ClipboardList, Copy, ExternalLink, Eraser, Info, Keyboard, RefreshCw, ScrollText, Settings } from 'lucide-react'
import { useI18n } from '../hooks/useI18n'
import { useRemote } from '../hooks/useRemote'
import type { Updates } from '../hooks/useUpdates'
import * as api from '../lib/backend'
import { platformOSLabel } from '../lib/capabilities'
import type { LangPref } from '../lib/i18n'
import { buildReport } from '../lib/report'
import { NAMED_KEYS, SHORTCUTS } from '../lib/shortcuts'
import { safeReleaseURL } from '../lib/version'
import { HotkeySettings } from './HotkeySettings'
import { Modal } from './Modal'

// Wails on Windows opens target=_blank links in a bare in-app window, so go through the runtime.
const ISSUE_URL = 'https://github.com/andredeomondes/optic/issues/new?template=tv-report.yml'

type Tab = 'atalhos' | 'janela' | 'diagnostico' | 'atualizacoes' | 'sobre'

/** Configurações: shortcuts, diagnostics (device info + protocol logs), about. */
export function SettingsPanel({ updates, onClose, initialTab = 'atalhos' }: { updates: Updates; onClose: () => void; initialTab?: Tab }) {
  const [tab, setTab] = useState<Tab>(initialTab)
  const { t } = useI18n()

  return (
    <Modal title={t('settings.title')} icon={<Settings size={20} />} onClose={onClose} className="settings">
      <nav className="tabs" role="tablist">
        <TabBtn id="atalhos" tab={tab} setTab={setTab} icon={<Keyboard size={14} />} label={t('settings.tabs.shortcuts')} />
        <TabBtn id="janela" tab={tab} setTab={setTab} icon={<AppWindow size={14} />} label={t('settings.tabs.window')} />
        <TabBtn id="diagnostico" tab={tab} setTab={setTab} icon={<ScrollText size={14} />} label={t('settings.tabs.diagnostics')} />
        <TabBtn id="atualizacoes" tab={tab} setTab={setTab} icon={<RefreshCw size={14} />} label={t('settings.tabs.updates')} />
        <TabBtn id="sobre" tab={tab} setTab={setTab} icon={<Info size={14} />} label={t('settings.tabs.about')} />
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
    <div className="tab-scroll">
      <HotkeySettings />
      <ShortcutList />
    </div>
  )
}

function ShortcutList() {
  const { t } = useI18n()
  return (
    <ul className="shortcut-list">
      {SHORTCUTS.map((s) => (
        <li key={s.id}>
          <span>{t(s.labelKey)}</span>
          <span className="keys">
            {s.keys.map((k) => (
              <kbd key={k}>{k in NAMED_KEYS ? t(NAMED_KEYS[k]) : k}</kbd>
            ))}
          </span>
        </li>
      ))}
      <li className="note">{t('shortcut.note')}</li>
    </ul>
  )
}

/** Tray behaviour; the preference lives in Go because OnBeforeClose reads it. */
const LANG_OPTIONS: { pref: LangPref; label: 'lang.auto' | 'lang.pt' | 'lang.en' }[] = [
  { pref: 'auto', label: 'lang.auto' },
  { pref: 'pt-BR', label: 'lang.pt' },
  { pref: 'en', label: 'lang.en' },
]

function WindowPrefs() {
  const { t, pref, setPref } = useI18n()
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
        <label className="pref" htmlFor="language">
          <span>{t('settings.language')}</span>
          <select id="language" className="lang-select" value={pref} onChange={(e) => setPref(e.target.value as LangPref)}>
            {LANG_OPTIONS.map((o) => (
              <option key={o.pref} value={o.pref}>
                {t(o.label)}
              </option>
            ))}
          </select>
        </label>
      </li>
      <li>
        <label className="pref" htmlFor="close-to-tray">
          <span>{t('window.closeToTray')}</span>
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
          ? t('window.trayUnavailable')
          : t('window.trayHelp')}
      </li>
      <li className="note">{t('window.compactNote')}</li>
    </ul>
  )
}

function Diagnostics() {
  const { t, lang } = useI18n()
  const { log, clearLog, info, host, tvs, status, tvState, notify } = useRemote()
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' })
  }, [log.length])

  const copy = async () => {
    const text = log.map((l) => `${l.time} ${l.dir} ${l.text}`).join('\n')
    try {
      await navigator.clipboard.writeText(text)
      notify('ok', t('diag.logsCopied'))
    } catch {
      notify('err', t('diag.copyFailed'))
    }
  }

  const copyReport = async () => {
    const text = buildReport({ version: __APP_VERSION__, userAgent: navigator.userAgent, tvs, info, host, log, lang })
    try {
      await navigator.clipboard.writeText(text)
      notify('ok', t('diag.reportCopied'))
    } catch {
      notify('err', t('diag.copyFailed'))
    }
  }

  const openIssue = (e: React.MouseEvent) => {
    e.preventDefault()
    api.BrowserOpenURL(ISSUE_URL)
  }

  const rows: [string, string][] = [
    [t('diag.status'), status],
    [t('diag.ip'), host || '—'],
    [t('diag.vendor'), info?.vendor || '—'],
    [t('diag.model'), info?.model || '—'],
    [t('diag.system'), info ? platformOSLabel(info.platform, info.version) : '—'],
    [t('diag.service'), info?.appVersion || '—'],
    [t('diag.volume'), tvState?.volumeMax ? `${tvState.volume} / ${tvState.volumeMax}` : '—'],
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
        <span className="label">{t('diag.protocolLogs')}</span>
        <span className="log-actions">
          <button className="link" onClick={copy}>
            <Copy size={12} /> {t('diag.copy')}
          </button>
          <button className="link" onClick={clearLog}>
            <Eraser size={12} /> {t('diag.clear')}
          </button>
        </span>
      </div>
      <div className="log">
        {log.length === 0 && <div className="line dsys">{t('diag.noEvents')}</div>}
        {log.map((l) => (
          <div key={l.id} className={`line d${l.dir === '>' ? 'out' : l.dir === '<' ? 'in' : l.dir === '!' ? 'err' : 'sys'}`}>
            <span className="t">{l.time}</span> {l.dir} {l.text}
          </div>
        ))}
        <div ref={endRef} />
      </div>
      <div className="label-row">
        <span className="label">{t('diag.otherBrand')}</span>
        <span className="log-actions">
          <button className="link" onClick={copyReport}>
            <ClipboardList size={12} /> {t('diag.copyReport')}
          </button>
          <a className="link" href={ISSUE_URL} onClick={openIssue}>
            <ExternalLink size={12} /> {t('diag.openIssue')}
          </a>
        </span>
      </div>
      <p className="note">{t('diag.legend')}</p>
    </div>
  )
}

function UpdatesTab({ updates }: { updates: Updates }) {
  const { t, lang } = useI18n()
  const { release, available, checking, auto, lastChecked, checkNow, setAuto } = updates
  const when = lastChecked ? new Date(lastChecked).toLocaleString(lang === 'en' ? 'en-US' : 'pt-BR') : t('updates.never')
  return (
    <ul className="shortcut-list">
      <li>
        <span>{t('updates.installed')}</span>
        <span>{__APP_VERSION__}</span>
      </li>
      <li>
        <span>{t('updates.latest')}</span>
        <span>{release ? t(available ? 'updates.tagNew' : 'updates.tagCurrent', { tag: release.tag }) : '—'}</span>
      </li>
      <li>
        <label className="pref" htmlFor="auto-update">
          <span>{t('updates.auto')}</span>
          <input id="auto-update" type="checkbox" className="switch" checked={auto} onChange={(e) => setAuto(e.target.checked)} />
        </label>
      </li>
      <li>
        <span>{t('updates.lastChecked', { when })}</span>
        <span className="log-actions">
          {available && release && (
            <button className="link" onClick={() => api.BrowserOpenURL(safeReleaseURL(release.url))}>
              <ExternalLink size={12} /> {t('update.whatsNew')}
            </button>
          )}
          <button className="link" onClick={checkNow} disabled={checking}>
            <RefreshCw size={12} /> {checking ? t('updates.checking') : t('updates.checkNow')}
          </button>
        </span>
      </li>
      <li className="note">{t('updates.note')}</li>
    </ul>
  )
}

function About() {
  const { t } = useI18n()
  return (
    <div className="about">
      <p>
        <b>OPTIC</b> {t('about.desc')}
      </p>
      <p className="note">{t('about.note')}</p>
    </div>
  )
}
