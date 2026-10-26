// Quick Command parser. No AI: a list of static commands plus small parsers
// for commands that take an argument. Add a command by appending to STATIC
// or a parser to PARSERS. Keywords work in English and Portuguese whatever
// the UI language; only the titles shown follow it (ctx.lang).
import type { AppDef } from './apps.ts'
import { type Key, type Lang, translate } from './i18n.ts'
import type { Platform } from './types.ts'

export type Action =
  | { type: 'key'; key: string }
  | { type: 'launch'; app: AppDef }
  | { type: 'volume'; level: number }
  | { type: 'text'; text: string }
  | { type: 'toast'; text: string }
  /** Switch to the input whose name matches query. */
  | { type: 'input'; query: string }
  /** minutes 0 cancels the timer. */
  | { type: 'sleep'; minutes: number }
  | { type: 'link'; url: string; label: string }
  | { type: 'ui'; target: 'settings' | 'scan' | 'focus-text' | 'paste' }

export type IconName =
  | 'app' | 'volume' | 'volume-up' | 'volume-down' | 'mute' | 'power' | 'home' | 'back'
  | 'ok' | 'up' | 'down' | 'left' | 'right' | 'play' | 'settings' | 'channel' | 'text' | 'search' | 'moon'

export type Command = {
  id: string
  title: string
  hint?: string
  icon: IconName
  action: Action
  app?: AppDef
  keywords?: string[]
}

/** A built-in command; its title is translated when suggestions are built. */
export type StaticCommand = Omit<Command, 'title'> & { titleKey: Key }

const key = (id: string, titleKey: Key, icon: IconName, k: string, keywords: string[]): StaticCommand => ({
  id,
  titleKey,
  icon,
  keywords,
  action: { type: 'key', key: k },
})

export const STATIC: StaticCommand[] = [
  key('vol-up', 'cmd.volUp', 'volume-up', 'VOLUME_UP', ['subir', 'volume +', 'mais alto', 'vol up', 'volume up', 'louder', 'turn up']),
  key('vol-down', 'cmd.volDown', 'volume-down', 'VOLUME_DOWN', ['baixar', 'volume -', 'mais baixo', 'vol down', 'volume down', 'quieter', 'turn down']),
  key('mute', 'cmd.mute', 'mute', 'MUTE', ['mute', 'silenciar', 'sem som', 'silence', 'no sound']),
  key('power', 'cmd.power', 'power', 'POWER', ['desligar tv', 'ligar tv', 'power', 'energia', 'turn off tv', 'turn on tv', 'tv off', 'tv on']),
  key('home', 'cmd.home', 'home', 'HOME', ['home', 'inicio', 'tela inicial', 'home screen']),
  key('back', 'cmd.back', 'back', 'BACK', ['back', 'retornar', 'go back']),
  key('ok', 'cmd.ok', 'ok', 'DPAD_CENTER', ['enter', 'confirmar', 'selecionar', 'select', 'confirm']),
  key('play', 'cmd.play', 'play', 'MEDIA_PLAY_PAUSE', ['pausar', 'tocar', 'continuar', 'pause', 'play', 'resume']),
  { id: 'settings', titleKey: 'cmd.settings', icon: 'settings', keywords: ['logs', 'diagnostico', 'preferencias', 'settings', 'diagnostics', 'preferences'], action: { type: 'ui', target: 'settings' } },
  key('settings-tv', 'cmd.tvSettings', 'settings', 'SETTINGS', ['configuracoes da tv', 'ajustes', 'tv settings']),
  key('ch-up', 'cmd.chUp', 'channel', 'CHANNEL_UP', ['canal +', 'canal acima', 'channel up', 'next channel']),
  key('ch-down', 'cmd.chDown', 'channel', 'CHANNEL_DOWN', ['canal -', 'canal abaixo', 'channel down', 'previous channel']),
  key('input', 'cmd.input', 'channel', 'TV_INPUT', ['hdmi', 'source', 'fonte', 'input', 'entrada']),
  { id: 'type', titleKey: 'cmd.type', icon: 'text', keywords: ['escrever', 'teclado', 'write', 'keyboard'], action: { type: 'ui', target: 'focus-text' } },
  { id: 'paste', titleKey: 'cmd.paste', icon: 'text', keywords: ['colar', 'clipboard', 'área de transferência', 'paste'], action: { type: 'ui', target: 'paste' } },
  { id: 'sleep-cancel', titleKey: 'cmd.sleepCancel', icon: 'moon', keywords: ['cancelar timer', 'parar timer', 'sem timer', 'cancel timer', 'stop timer', 'no timer'], action: { type: 'sleep', minutes: 0 } },
  { id: 'scan', titleKey: 'cmd.scan', icon: 'search', keywords: ['procurar', 'descobrir', 'scan', 'find tvs', 'discover'], action: { type: 'ui', target: 'scan' } },
]

/** Lowercase, strip accents and extra spaces so "Início" matches "inicio". */
export const normalize = (s: string) =>
  s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase().replace(/\s+/g, ' ').trim()

/** input is normalized; raw keeps the user's casing and accents. */
type Parser = (input: string, apps: AppDef[], raw: string, ctx: Context) => Command[]

/** What the connected TV can do, plus the UI language for titles; parsers for optional features check it. */
export type Context = { toast?: boolean; inputs?: boolean; platform?: Platform; lang?: Lang }

const tr = (ctx: Context, k: Key, vars?: Record<string, string | number>) => translate(ctx.lang ?? 'pt-BR', k, vars)

const PARSERS: Parser[] = [
  // "volume 30", "vol 30", "volume em 30", "volume to 30"
  (input, _apps, _raw, ctx) => {
    const m = input.match(/^vol(?:ume)?\s+(?:em\s+|para\s+|to\s+|at\s+)?(\d{1,3})$/)
    if (!m) return []
    const level = Math.min(100, Number(m[1]))
    return [{ id: `vol-${level}`, title: tr(ctx, 'cmd.volume', { level }), icon: 'volume', action: { type: 'volume', level } }]
  },
  // "desligar em 30 min", "turn off in 30 min", "timer 45" (1-240; anything else is not suggested)
  (input, _apps, _raw, ctx) => {
    const m = input.match(/^(?:desligar\s+em|turn\s+(?:the\s+)?(?:tv\s+)?off(?:\s+(?:the\s+)?tv)?\s+in|sleep(?:\s+in)?|timer)\s+(\d{1,4})(?:\s*min(?:utos?|utes?|s)?)?$/)
    if (!m) return []
    const minutes = Number(m[1])
    if (minutes < 1 || minutes > 240) return []
    return [{ id: `sleep-${minutes}`, title: tr(ctx, 'cmd.sleep', { n: minutes }), icon: 'moon', action: { type: 'sleep', minutes } }]
  },
  // "abrir youtube", "abre net", "open spotify"
  (input, apps, _raw, ctx) => {
    const m = input.match(/^(?:abrir|abre|open|launch|iniciar|app)\s*(.*)$/)
    if (!m) return []
    const q = m[1]
    return apps
      .filter((a) => normalize(a.name).includes(q) || a.id.includes(q))
      .map((a) => ({ id: `open-${a.id}`, title: tr(ctx, 'cmd.open', { name: a.name }), icon: 'app' as const, app: a, action: { type: 'launch' as const, app: a } }))
  },
  // "pesquisar cimatec no youtube", "search cimatec on youtube" (deep link; depends on the TV's YouTube app)
  (_input, _apps, raw, ctx) => {
    const m = raw.trim().match(/^(?:pesquisar|search)\s+(.+?)(?:\s+(?:no|on)\s+youtube)?$/i)
    if (!m) return []
    const q = m[1].replace(/^["“]|["”]$/g, '')
    const url = `https://www.youtube.com/results?search_query=${encodeURIComponent(q)}`
    return [{ id: 'yt-search', title: tr(ctx, 'cmd.search', { q }), icon: 'search', action: { type: 'link', url, label: 'YouTube' } }]
  },
  // "digitar olá mundo", "escrever ...", "type hello"
  (_input, _apps, raw, ctx) => {
    const m = raw.trim().match(/^(?:digitar|escrever|type)\s+(.+)$/i)
    if (!m) return []
    return [{ id: 'type-text', title: tr(ctx, 'cmd.typeText', { text: m[1] }), hint: tr(ctx, 'cmd.typeTextHint'), icon: 'text', action: { type: 'text', text: m[1] } }]
  },
  // "entrada tv", "input hdmi 2", "hdmi 2" (only when the TV lists inputs)
  (_input, _apps, raw, ctx) => {
    if (!ctx.inputs) return []
    const t = raw.trim()
    const m = t.match(/^(?:entrada|input|fonte|source)\s+(.+)$/i) ?? t.match(/^(hdmi\s*\d+)$/i)
    const query = m?.[1].trim()
    if (!query) return []
    return [{ id: 'input-pick', title: tr(ctx, 'cmd.inputPick', { query }), icon: 'channel', action: { type: 'input', query } }]
  },
  // "avisar na tv a janta saiu", "mensagem oi", "notify tv hi" (only when the TV can show toasts)
  (_input, _apps, raw, ctx) => {
    if (!ctx.toast) return []
    const m = raw.trim().match(/^(?:avisar\s+na\s+tv|mensagem|notify\s+tv|message)\s+(.+)$/i)
    const text = m?.[1].trim()
    if (!text) return []
    return [{ id: 'toast', title: tr(ctx, 'cmd.toast', { text }), hint: tr(ctx, 'cmd.toastHint'), icon: 'text', action: { type: 'toast', text } }]
  },
]

const LANGS: Lang[] = ['pt-BR', 'en']

/** Returns ranked suggestions for the palette. */
export function suggest(raw: string, apps: AppDef[], ctx: Context = {}): Command[] {
  const input = normalize(raw)
  const dynamic = input ? PARSERS.flatMap((p) => p(input, apps, raw, ctx)) : []

  // The raw TV_INPUT key only exists on Android and Samsung (KEY_SOURCE); Roku and LG use the picker.
  const keyInput = ctx.platform === undefined || ctx.platform === '' || ctx.platform === 'androidtv' || ctx.platform === 'tizen'
  const statics = (keyInput ? STATIC : STATIC.filter((c) => c.id !== 'input')).map(({ titleKey, ...c }) => ({
    command: { ...c, title: tr(ctx, titleKey) } as Command,
    // Match against both languages, whatever the UI language is.
    names: LANGS.map((l) => translate(l, titleKey)),
  }))
  const appCommands = apps.map((a) => ({
    command: {
      id: `open-${a.id}`,
      title: tr(ctx, 'cmd.open', { name: a.name }),
      icon: 'app',
      app: a,
      keywords: [a.name, a.id],
      action: { type: 'launch', app: a },
    } as Command,
    names: LANGS.map((l) => translate(l, 'cmd.open', { name: a.name })),
  }))

  const pool = [...statics, ...appCommands]
  const matched = (input
    ? pool.filter((p) => [...p.names, ...(p.command.keywords ?? [])].some((k) => normalize(k).includes(input)))
    : pool
  ).map((p) => p.command)

  const seen = new Set<string>()
  return [...dynamic, ...matched].filter((c) => !seen.has(c.id) && seen.add(c.id)).slice(0, 8)
}
