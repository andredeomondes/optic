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

/** Hides the device-specific half of a MAC address: aa:bb:cc:dd:ee:ff -> aa:bb:cc:**:**:**. */
function maskMac(mac: string): string {
  return mac.replace(/^((?:[0-9a-f]{2}[:-]){3})(?:[0-9a-f]{2}[:-]){2}[0-9a-f]{2}$/i, '$1**:**:**')
}

/** Also masks MACs that show up inside free-form log text. */
function maskMacsIn(text: string): string {
  return text.replace(/\b((?:[0-9a-f]{2}[:-]){3})(?:[0-9a-f]{2}[:-]){2}[0-9a-f]{2}\b/gi, '$1**:**:**')
}

/** Markdown diagnostic report, ready to paste into a GitHub issue. */
export function buildReport({ version, userAgent, tvs, info, host, log }: ReportInput): string {
  const out: string[] = ['### OPTIC', '', `- Versão: ${version}`, `- Ambiente: ${userAgent}`, '']

  out.push('### TVs encontradas', '')
  if (tvs.length === 0) {
    out.push('Nenhuma TV encontrada.')
  } else {
    out.push('| Nome | IP | Plataforma |', '| --- | --- | --- |')
    for (const t of tvs) {
      out.push(`| ${t.name} | ${t.ip} | ${PLATFORM_LABEL[t.platform] ?? t.platform} |`)
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
  out.push(...log.slice(-MAX_LOG_LINES).map((l) => maskMacsIn(`${l.time} ${l.dir} ${l.text}`)))
  out.push('```', '')
  return out.join('\n')
}
