import { PLATFORM_LABEL } from './capabilities.ts'
import type { DeviceInfo, LogLine, TV } from './types.ts'

const MAX_LOG_LINES = 200

export type ReportInput = {
  version: string
  userAgent: string
  tvs: TV[]
  info: DeviceInfo | null
  host: string
  log: LogLine[]
}

const MASKED_MAC = '**:**:**:**:**:**'

/** First three octets (vendor prefix) of 6 hex digits, followed by the masked device half. */
function ouiOnly(hex6: string): string {
  const h = hex6.toLowerCase()
  return `${h.slice(0, 2)}:${h.slice(2, 4)}:${h.slice(4, 6)}:**:**:**`
}

/** Fails closed: any value that is not a clean 12-digit MAC is fully masked. */
function maskMac(mac: string): string {
  const hex = mac.replace(/[^0-9a-f]/gi, '')
  return hex.length === 12 && /^[0-9a-f:.\-\s]+$/i.test(mac) ? ouiOnly(hex.slice(0, 6)) : MASKED_MAC
}

/**
 * Masks MACs anywhere in free text: separated (colon/dash), dotted (aabb.ccdd.eeff)
 * and unseparated. Hex lookarounds (not \b, which misses "TV_AABBCCDDEEFF") mask any
 * 12-digit run that is not part of a longer hex run, so long protocol tokens survive.
 */
function maskMacsIn(text: string): string {
  return text
    .replace(/(?<![0-9a-f])((?:[0-9a-f]{2}[:-]){3})(?:[0-9a-f]{2}[:-]){2}[0-9a-f]{2}(?![0-9a-f])/gi, (_m, oui: string) => ouiOnly(oui.replace(/[:-]/g, '')))
    .replace(/(?<![0-9a-f])([0-9a-f]{4})\.([0-9a-f]{4})\.[0-9a-f]{4}(?![0-9a-f])/gi, (_m, a: string, b: string) => ouiOnly(a + b.slice(0, 2)))
    .replace(/(?<![0-9a-f])([0-9a-f]{6})[0-9a-f]{6}(?![0-9a-f])/gi, (_m, oui: string) => ouiOnly(oui))
}

/** Table cells: keep the row intact if a value contains a pipe. */
const cell = (v: string) => v.replace(/\|/g, '\\|')

/** Markdown diagnostic report, ready to paste into a GitHub issue. */
export function buildReport({ version, userAgent, tvs, info, host, log }: ReportInput): string {
  const out: string[] = ['### OPTIC', '', `- Versão: ${version}`, `- Ambiente: ${userAgent}`, '']

  out.push('### TVs encontradas', '')
  if (tvs.length === 0) {
    out.push('Nenhuma TV encontrada.')
  } else {
    out.push('| Nome | IP | Plataforma |', '| --- | --- | --- |')
    for (const t of tvs) {
      out.push(`| ${cell(t.name)} | ${cell(t.ip)} | ${PLATFORM_LABEL[t.platform] ?? t.platform} |`)
    }
    const macs = tvs.filter((t) => t.mac).map((t) => `${t.name}: ${maskMac(t.mac!)}`)
    if (macs.length) out.push('', ...macs.map((m) => `- MAC ${m}`))
  }
  out.push('')

  out.push('### TV conectada', '')
  if (!info) {
    out.push('Nenhuma TV conectada.')
  } else {
    out.push(
      `- IP: ${host || '—'}`,
      `- Plataforma: ${PLATFORM_LABEL[info.platform] ?? info.platform}`,
      `- Fabricante: ${info.vendor || '—'}`,
      `- Modelo: ${info.model || '—'}`,
      `- Sistema: ${info.version || '—'}`,
      `- Serviço remoto: ${info.appVersion || '—'}`,
      `- Recursos: ${Object.entries(info.capabilities ?? {}).map(([k, v]) => `${k}=${v ? 'sim' : 'não'}`).join(', ') || '—'}`,
    )
  }
  out.push('')

  out.push('### Log', '', '```')
  out.push(...log.slice(-MAX_LOG_LINES).map((l) => `${l.time} ${l.dir} ${l.text}`))
  out.push('```', '')
  // One pass over everything: names, models, user agent and log may all embed a MAC.
  return maskMacsIn(out.join('\n'))
}
