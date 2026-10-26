// English versions of the pt-BR messages the Go backend returns. Go keeps
// pt-BR as the source text; the UI translates at render time. Unknown
// messages (and the NOT_PAIRED sentinel) pass through unchanged.
import type { Lang } from './i18n.ts'

const exact: Record<string, string> = {
  // Apps and links
  'a TV não abre este app pelo controle remoto; abra pela tela inicial':
    "The TV can't open this app from the remote; open it from the home screen.",
  'A Roku não abre links; abra o app pela lista.': "Roku can't open links; open the app from the list.",
  'A TV Samsung não enviou a lista de apps. Tente de novo.': "The Samsung TV didn't send its app list. Try again.",
  // Pairing and connection
  'TVs Roku não precisam de pareamento': "Roku TVs don't need pairing.",
  'A TV recusou a conexão. Tente de novo e escolha Permitir.': 'The TV refused the connection. Try again and choose Allow.',
  "A TV não respondeu ao pedido: confirme 'Permitir' na tela.": "The TV didn't answer the request: confirm 'Allow' on the screen.",
  'Conexão com a TV perdida durante o pareamento.': 'Lost the connection to the TV during pairing.',
  "A TV Samsung não respondeu. Verifique se ela está ligada e tente de novo. Se aparecer 'Permitir' na TV, aceite e tente de novo.":
    "The Samsung TV didn't respond. Check that it's on and try again. If 'Allow' appears on the TV, accept it and try again.",
  'A TV LG não respondeu. Verifique se ela está ligada e tente de novo.':
    "The LG TV didn't respond. Check that it's on and try again.",
  'Abra um campo de texto na TV antes de enviar.': 'Open a text field on the TV before sending.',
  'não conectado': 'Not connected.',
  'nenhum pairing em andamento': 'No pairing in progress.',
  'código não confere (erro de digitação?)': "The code doesn't match (typo?).",
  'TV recusou a conexão: rode o pairing primeiro': 'The TV refused the connection: pair it first.',
  'TV não apresentou certificado': "The TV didn't present a certificate.",
  'código deve ser hexadecimal (0-9, A-F)': 'The code must be hexadecimal (0-9, A-F).',
  'certificados precisam ser RSA': 'Certificates must be RSA.',
  'TV não ativou a sessão de controle': "The TV didn't start the control session.",
  'sessão de controle encerrada': 'Control session closed.',
  'TV reportou erro na sessão': 'The TV reported a session error.',
  'a busca da TV abriu sem teclado (talvez por voz)': 'The TV search opened without a keyboard (maybe voice).',
  // Volume, power, inputs, toast
  'volume atual desconhecido: mude o volume uma vez e tente de novo': 'Current volume unknown: change the volume once and try again.',
  'Esta TV não informa o volume: use Volume + e −.': "This TV doesn't report its volume: use Volume + and −.",
  'MAC da TV desconhecido: conecte uma vez com ela ligada': 'TV MAC address unknown: connect once while it is on.',
  'Entrada desconhecida': 'Unknown input.',
  'Mensagem vazia': 'Empty message.',
  'checagem de versão indisponível': 'Version check unavailable.',
  // Roku
  'A Roku bloqueou o controle pela rede. Na TV: Configurações › Sistema › Configurações avançadas do sistema › Controle por apps móveis › Acesso à rede: Padrão.':
    'Roku blocked network control. On the TV: Settings › System › Advanced system settings › Control by mobile apps › Network access: Default.',
  // Shared TV adapters (tizen/webos clients)
  'tizen: conexão não autorizada pela TV': 'Connection not authorized by the TV.',
  'tizen: a TV não respondeu': "The TV didn't respond.",
  'tizen: conexão encerrada': 'Connection closed.',
  'tizen: endereço inválido': 'Invalid address.',
  'tizen: resposta da TV grande demais': 'The TV response is too large.',
  'tizen: resposta da TV inválida': 'The TV response is invalid.',
  'webos: registro recusado pela TV': 'Registration refused by the TV.',
  'webos: a TV pediu confirmação': 'The TV asked for confirmation.',
  'webos: conexão encerrada': 'Connection closed.',
  'webos: resposta de registro sem chave': 'The TV registration response had no key.',
  'webos: TV não informou o socket de ponteiro': "The TV didn't provide the pointer socket.",
  'webos: socket de ponteiro inválido': 'Invalid pointer socket.',
  'webos: não foi possível conectar ao socket de ponteiro': 'Could not connect to the pointer socket.',
  // Global shortcuts
  'Use Ctrl ou Alt junto com a tecla': 'Use Ctrl or Alt together with the key.',
  'A tecla Windows é reservada pelo sistema': 'The Windows key is reserved by the system.',
  'Atalho reservado pelo sistema': 'Shortcut reserved by the system.',
  'Em uso por outro programa': 'In use by another program.',
  'Atalhos globais indisponíveis neste sistema': 'Global shortcuts are unavailable on this system.',
  'Atalhos globais desligados': 'Global shortcuts are off.',
  'Ação repetida': 'Repeated action.',
  'Atalho já usado em outra ação': 'Shortcut already used by another action.',
  'Conecte uma TV primeiro': 'Connect a TV first.',
}

// Hotkey action labels (hotkeys.go) as they appear before ": <reason>".
const actionLabels: Record<string, string> = {
  'Volume +': 'Volume +',
  'Volume −': 'Volume −',
  Mudo: 'Mute',
  Início: 'Home',
  'Comandos rápidos': 'Quick commands',
  'Mídia: play/pause': 'Media: play/pause',
  'Mídia: próxima': 'Media: next',
  'Mídia: anterior': 'Media: previous',
  'Mídia: parar': 'Media: stop',
}

const labelAlternation = Object.keys(actionLabels)
  .map((l) => l.replace(/[+]/g, '\\$&'))
  .join('|')

// Ordered specific → general; the first match wins.
const patterns: Array<[RegExp, (m: RegExpMatchArray) => string]> = [
  [/^Este app não está instalado na TV (.+)\.$/, (m) => `This app isn't installed on the ${m[1]} TV.`],
  [/^Tecla não suportada pela (.+): (.+)$/, (m) => `Key not supported by ${m[1]}: ${m[2]}.`],
  [/^[Tt]ecla desconhecida: (.+)$/, (m) => `Unknown key: ${m[1]}.`],
  [/^Atalho repetido: (.+)$/, (m) => `Duplicate shortcut: ${m[1]}.`],
  [/^código deve ter (\d+) caracteres$/, (m) => `The code must have ${m[1]} characters.`],
  [/^TV respondeu status (\d+) \((.*)\)$/, (m) => `The TV answered with status ${m[1]} (${m[2]}).`],
  [/^TV respondeu mensagem inesperada \(esperava campo (\d+)\)$/, (m) => `The TV sent an unexpected message (expected field ${m[1]}).`],
  [/^lendo resposta da TV: (.+)$/, (m) => `Reading the TV's response: ${m[1]}`],
  [/^MAC inválido (".*")$/, (m) => `Invalid MAC address ${m[1]}.`],
  [/^descrição UPnP: status (\d+)$/, (m) => `UPnP description: status ${m[1]}.`],
  [/^descrição UPnP grande demais$/, () => 'UPnP description too large.'],
  [/^(?:tizen|webos): não foi possível conectar(?:: (.+))?$/, (m) => (m[1] ? `Could not connect: ${m[1]}` : 'Could not connect.')],
  [/^tizen: a TV respondeu (\d+)$/, (m) => `The TV answered with status ${m[1]}.`],
  [/^tizen: consultar a TV: (.+)$/, (m) => `Querying the TV: ${m[1]}`],
  [/^tizen: ler resposta da TV: (.+)$/, (m) => `Reading the TV response: ${m[1]}`],
  [/^tizen: enviar: (.+)$/, (m) => `Sending: ${m[1]}`],
  [/^webos: lista de apps inválida: (.+)$/, (m) => `Invalid app list: ${m[1]}`],
  [/^webos: lista de entradas inválida: (.+)$/, (m) => `Invalid input list: ${m[1]}`],
  [/^webos: enviar registro: (.+)$/, (m) => `Sending registration: ${m[1]}`],
  [/^webos: enviar (.+?): (.+)$/, (m) => `Sending ${m[1]}: ${m[2]}`],
  [/^webos: assinar (.+?): (.+)$/, (m) => `Subscribing to ${m[1]}: ${m[2]}`],
  [/^roku: (.+): requisição inválida$/, (m) => `Invalid request (${m[1]}).`],
  [/^roku: HTTP (\d+)$/, (m) => `Roku answered with HTTP ${m[1]}.`],
  // "<ação>: <motivo>" from SetHotkeys: only known action labels, reason translated too.
  [new RegExp(`^(${labelAlternation}): (.+)$`), (m) => `${actionLabels[m[1]]}: ${tr(m[2])}`],
]

function tr(msg: string): string {
  const hit = exact[msg]
  if (hit !== undefined) return hit
  for (const [re, fn] of patterns) {
    const m = msg.match(re)
    if (m) return fn(m)
  }
  return msg
}

/** Translates a backend error message; pt-BR and unknown messages are returned as they came. */
export function translateError(msg: string, lang: Lang): string {
  return lang === 'en' ? tr(msg) : msg
}
