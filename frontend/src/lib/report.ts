import { PLATFORM_LABEL } from './capabilities.ts'
import { type Key, type Lang, translate } from './i18n.ts'
import type { DeviceInfo, LogLine, Platform, TV } from './types.ts'

const MAX_LOG_LINES = 200

export type ReportInput = {
  version: string
  userAgent: string
  tvs: TV[]
  info: DeviceInfo | null
  host: string
  log: LogLine[]
  /** Report language; defaults to pt-BR. */
  lang?: Lang
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

const PACKAGE_NAME = /^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$/

/**
 * Protocol traces carry typed text (IME frames) and the TV echoes field content, so
 * every quoted string is replaced, except Android package names, which help diagnosis.
 * Fails closed: an unterminated string is redacted up to the end of the line.
 */
function redactStrings(text: string, rawOmitted: string): string {
  // Undecodable frames are dumped as hex, which can spell out typed text.
  return text.replace(/<raw[^>]*>?/g, rawOmitted).replace(/"((?:[^"\\]|\\.|\\$)*)(?:"|$)/g, (_m, body: string) =>
    PACKAGE_NAME.test(body) && _m.endsWith('"') ? `"${body}"` : '"[redacted]"',
  )
}

/** Table cells: keep the row intact if a value contains a pipe. */
const cell = (v: string) => v.replace(/\|/g, '\\|')

/** Markdown diagnostic report, ready to paste into a GitHub issue. */
export function buildReport({ version, userAgent, tvs, info, host, log, lang = 'pt-BR' }: ReportInput): string {
  const t = (k: Key) => translate(lang, k)
  const platform = (p: Platform) => p === '' ? t('device.platformUnknown') : (PLATFORM_LABEL[p] ?? p)
  const out: string[] = ['### OPTIC', '', `- ${t('report.version')}: ${version}`, `- ${t('report.env')}: ${userAgent}`, '']

  out.push(`### ${t('report.tvsFound')}`, '')
  if (tvs.length === 0) {
    out.push(t('sidebar.empty'))
  } else {
    out.push(`| ${t('report.colName')} | IP | ${t('report.colPlatform')} |`, '| --- | --- | --- |')
    for (const tv of tvs) {
      out.push(`| ${cell(tv.name)} | ${cell(tv.ip)} | ${platform(tv.platform)} |`)
    }
    const macs = tvs.filter((v) => v.mac).map((v) => `${v.name}: ${maskMac(v.mac!)}`)
    if (macs.length) out.push('', ...macs.map((m) => `- MAC ${m}`))
  }
  out.push('')

  out.push(`### ${t('report.connected')}`, '')
  if (!info) {
    out.push(t('report.noneConnected'))
  } else {
    out.push(
      `- IP: ${host || '—'}`,
      `- ${t('report.colPlatform')}: ${platform(info.platform)}`,
      `- ${t('diag.vendor')}: ${info.vendor || '—'}`,
      `- ${t('diag.model')}: ${info.model || '—'}`,
      `- ${t('diag.system')}: ${info.version || '—'}`,
      `- ${t('diag.service')}: ${info.appVersion || '—'}`,
      `- ${t('report.features')}: ${Object.entries(info.capabilities ?? {}).map(([k, v]) => `${k}=${v ? t('report.yes') : t('report.no')}`).join(', ') || '—'}`,
    )
  }
  out.push('')

  out.push('### Log', '', '```')
  out.push(...log.slice(-MAX_LOG_LINES).map((l) => `${l.time} ${l.dir} ${redactStrings(l.text, t('report.rawOmitted'))}`))
  out.push('```', '')
  // One pass over everything: names, models, user agent and log may all embed a MAC.
  return maskMacsIn(out.join('\n'))
}
