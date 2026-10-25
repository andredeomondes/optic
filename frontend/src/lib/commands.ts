// Quick Command parser. No AI: a list of static commands plus small parsers
// for commands that take an argument. Add a command by appending to STATIC
// or a parser to PARSERS.
import type { AppDef } from './apps'

export type Action =
  | { type: 'key'; key: string }
  | { type: 'launch'; app: AppDef }
  | { type: 'volume'; level: number }
  | { type: 'text'; text: string }
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

const key = (id: string, title: string, icon: IconName, k: string, keywords: string[]): Command => ({
  id,
  title,
  icon,
  keywords,
  action: { type: 'key', key: k },
})

export const STATIC: Command[] = [
  key('vol-up', 'Aumentar volume', 'volume-up', 'VOLUME_UP', ['subir', 'volume +', 'mais alto', 'vol up']),
  key('vol-down', 'Diminuir volume', 'volume-down', 'VOLUME_DOWN', ['baixar', 'volume -', 'mais baixo', 'vol down']),
  key('mute', 'Mudo', 'mute', 'MUTE', ['mute', 'silenciar', 'sem som']),
  key('power', 'Desligar / ligar TV', 'power', 'POWER', ['desligar tv', 'ligar tv', 'power', 'energia']),
  key('home', 'Início', 'home', 'HOME', ['home', 'inicio', 'tela inicial']),
  key('back', 'Voltar', 'back', 'BACK', ['back', 'retornar']),
  key('ok', 'OK / selecionar', 'ok', 'DPAD_CENTER', ['enter', 'confirmar', 'selecionar']),
  key('play', 'Play / pause', 'play', 'MEDIA_PLAY_PAUSE', ['pausar', 'tocar', 'continuar', 'pause']),
  key('settings-tv', 'Ajustes da TV', 'settings', 'SETTINGS', ['configuracoes da tv', 'ajustes']),
  key('ch-up', 'Próximo canal', 'channel', 'CHANNEL_UP', ['canal +', 'canal acima']),
  key('ch-down', 'Canal anterior', 'channel', 'CHANNEL_DOWN', ['canal -', 'canal abaixo']),
  key('input', 'Entrada / source', 'channel', 'TV_INPUT', ['hdmi', 'source', 'fonte']),
  { id: 'type', title: 'Digitar na TV', icon: 'text', keywords: ['escrever', 'teclado'], action: { type: 'ui', target: 'focus-text' } },
  { id: 'paste', title: 'Colar na TV', icon: 'text', keywords: ['colar', 'clipboard', 'área de transferência'], action: { type: 'ui', target: 'paste' } },
  { id: 'sleep-cancel', title: 'Cancelar timer', icon: 'moon', keywords: ['cancelar timer', 'parar timer', 'sem timer'], action: { type: 'sleep', minutes: 0 } },
  { id: 'scan', title: 'Buscar TVs na rede', icon: 'search', keywords: ['procurar', 'descobrir'], action: { type: 'ui', target: 'scan' } },
  { id: 'settings', title: 'Configurações do OPTIC', icon: 'settings', keywords: ['logs', 'diagnostico', 'preferencias'], action: { type: 'ui', target: 'settings' } },
]

/** Lowercase, strip accents and extra spaces so "Início" matches "inicio". */
export const normalize = (s: string) =>
  s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase().replace(/\s+/g, ' ').trim()

/** input is normalized; raw keeps the user's casing and accents. */
type Parser = (input: string, apps: AppDef[], raw: string) => Command[]

const PARSERS: Parser[] = [
  // "volume 30", "vol 30", "volume em 30"
  (input) => {
    const m = input.match(/^vol(?:ume)?\s+(?:em\s+|para\s+)?(\d{1,3})$/)
    if (!m) return []
    const level = Math.min(100, Number(m[1]))
    return [{ id: `vol-${level}`, title: `Volume ${level}`, icon: 'volume', action: { type: 'volume', level } }]
  },
  // "desligar em 30 min", "desligar em 1 minuto", "timer 45" (1-240; anything else is not suggested)
  (input) => {
    const m = input.match(/^(?:desligar\s+em|timer)\s+(\d{1,4})(?:\s*min(?:utos?)?)?$/)
    if (!m) return []
    const minutes = Number(m[1])
    if (minutes < 1 || minutes > 240) return []
    return [{ id: `sleep-${minutes}`, title: `Desligar a TV em ${minutes} min`, icon: 'moon', action: { type: 'sleep', minutes } }]
  },
  // "abrir youtube", "abre net", "open spotify"
  (input, apps) => {
    const m = input.match(/^(?:abrir|abre|open|iniciar|app)\s*(.*)$/)
    if (!m) return []
    const q = m[1]
    return apps
      .filter((a) => normalize(a.name).includes(q) || a.id.includes(q))
      .map((a) => ({ id: `open-${a.id}`, title: `Abrir ${a.name}`, icon: 'app' as const, app: a, action: { type: 'launch' as const, app: a } }))
  },
  // "pesquisar cimatec no youtube" (deep link; depends on the TV's YouTube app)
  (_input, _apps, raw) => {
    const m = raw.trim().match(/^pesquisar\s+(.+?)(?:\s+no\s+youtube)?$/i)
    if (!m) return []
    const q = m[1].replace(/^["“]|["”]$/g, '')
    const url = `https://www.youtube.com/results?search_query=${encodeURIComponent(q)}`
    return [{ id: 'yt-search', title: `Pesquisar “${q}” no YouTube`, icon: 'search', action: { type: 'link', url, label: 'YouTube' } }]
  },
  // "digitar olá mundo", "escrever ..."
  (_input, _apps, raw) => {
    const m = raw.trim().match(/^(?:digitar|escrever|type)\s+(.+)$/i)
    if (!m) return []
    return [{ id: 'type-text', title: `Digitar “${m[1]}”`, hint: 'envia para o campo de texto aberto', icon: 'text', action: { type: 'text', text: m[1] } }]
  },
]

/** Returns ranked suggestions for the palette. */
export function suggest(raw: string, apps: AppDef[]): Command[] {
  const input = normalize(raw)
  const dynamic = input ? PARSERS.flatMap((p) => p(input, apps, raw)) : []

  const appCommands: Command[] = apps.map((a) => ({
    id: `open-${a.id}`,
    title: `Abrir ${a.name}`,
    icon: 'app',
    app: a,
    keywords: [a.name, a.id],
    action: { type: 'launch', app: a },
  }))

  const pool = [...STATIC, ...appCommands]
  const matched = input
    ? pool.filter((c) => [c.title, ...(c.keywords ?? [])].some((k) => normalize(k).includes(input)))
    : pool

  const seen = new Set<string>()
  return [...dynamic, ...matched].filter((c) => !seen.has(c.id) && seen.add(c.id)).slice(0, 8)
}
