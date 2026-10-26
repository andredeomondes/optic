// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { normalize, suggest } from './commands.ts'
import type { AppDef } from './apps.ts'

const CATALOG: AppDef[] = [
  { id: 'youtube', name: 'YouTube', pkg: 'com.google.android.youtube.tv', brand: '#f00' },
  { id: 'netflix', name: 'Netflix', pkg: 'com.netflix.ninja', brand: '#e50914' },
]

test('toast command', () => {
  const ctx = { toast: true }
  assert.deepEqual(suggest('avisar na tv oi', CATALOG, ctx)[0].action, { type: 'toast', text: 'oi' })
  assert.deepEqual(suggest('Mensagem  A Janta Saiu ', CATALOG, ctx)[0].action, { type: 'toast', text: 'A Janta Saiu' })
  assert.ok(!suggest('mensagem   ', CATALOG, ctx).some((c) => c.action.type === 'toast'))
  assert.ok(!suggest('avisar na tv', CATALOG, ctx).some((c) => c.action.type === 'toast'))
  assert.ok(!suggest('avisar na tv oi', CATALOG).some((c) => c.action.type === 'toast'))
})

test('normalize strips accents and case', () => {
  assert.equal(normalize('  Início   DA TV '), 'inicio da tv')
})

test('volume N sets absolute volume', () => {
  const [first] = suggest('volume 30', CATALOG)
  assert.deepEqual(first.action, { type: 'volume', level: 30 })
  assert.deepEqual(suggest('vol 250', CATALOG)[0].action, { type: 'volume', level: 100 })
})

test('abrir <app> launches matching app', () => {
  const [first] = suggest('abrir youtube', CATALOG)
  assert.equal(first.action.type, 'launch')
  assert.equal(first.app?.id, 'youtube')
  assert.equal(suggest('abre net', CATALOG)[0].app?.id, 'netflix')
})

test('static commands match keywords and accents', () => {
  const ids = (q: string) => suggest(q, CATALOG).map((c) => c.id)
  assert.ok(ids('mudo').includes('mute'))
  assert.ok(ids('desligar tv').includes('power'))
  assert.ok(ids('inicio').includes('home'))
  assert.ok(ids('aumentar volume').includes('vol-up'))
  assert.ok(ids('diminuir volume').includes('vol-down'))
  assert.ok(ids('voltar').includes('back'))
})

test('digitar <texto> sends text', () => {
  assert.deepEqual(suggest('digitar Olá mundo', CATALOG)[0].action, { type: 'text', text: 'Olá mundo' })
})

test('empty input lists commands, capped', () => {
  assert.equal(suggest('', CATALOG).length, 8)
})

test('pesquisar <q> no youtube builds a search link', () => {
  const [first] = suggest('pesquisar "Cimatec" no YouTube', CATALOG)
  assert.equal(first.action.type, 'link')
  assert.ok(first.action.type === 'link' && first.action.url.endsWith('search_query=Cimatec'))
})

test('colar finds the paste command', () => {
  assert.deepEqual(suggest('colar', CATALOG).find((c) => c.id === 'paste')?.action, { type: 'ui', target: 'paste' })
})

test('desligar em N min', () => {
  assert.deepEqual(suggest('desligar em 30 min', CATALOG)[0].action, { type: 'sleep', minutes: 30 })
  assert.deepEqual(suggest('desligar em 1 minuto', CATALOG)[0].action, { type: 'sleep', minutes: 1 })
  assert.deepEqual(suggest('timer 45', CATALOG)[0].action, { type: 'sleep', minutes: 45 })
  assert.ok(suggest('cancelar timer', CATALOG).some((c) => c.action.type === 'sleep' && c.action.minutes === 0))
})

test('sleep timer rejects minutes outside 1-240', () => {
  for (const q of ['timer 0', 'timer 241', 'desligar em 500 min']) {
    assert.ok(!suggest(q, CATALOG).some((c) => c.action.type === 'sleep' && c.action.minutes > 0), q)
  }
})

test('input key command shows only on android tv, samsung or unknown platform', () => {
  const has = (platform?: 'androidtv' | 'webos' | 'roku' | 'tizen' | '') => suggest('hdmi', CATALOG, { platform }).some((c) => c.id === 'input')
  assert.ok(has())
  assert.ok(has(''))
  assert.ok(has('androidtv'))
  assert.ok(has('tizen'))
  assert.ok(!has('roku'))
  assert.ok(!has('webos'))
})

test('input command parses names and hdmi numbers when the TV lists inputs', () => {
  const ctx = { inputs: true }
  assert.deepEqual(suggest('hdmi 2', CATALOG, ctx)[0].action, { type: 'input', query: 'hdmi 2' })
  assert.deepEqual(suggest('hdmi2', CATALOG, ctx)[0].action, { type: 'input', query: 'hdmi2' })
  assert.deepEqual(suggest('entrada tv', CATALOG, ctx)[0].action, { type: 'input', query: 'tv' })
  assert.ok(!suggest('entrada', CATALOG, ctx).some((c) => c.action.type === 'input'))
})

test('input command is hidden without inputs', () => {
  assert.ok(!suggest('hdmi 2', CATALOG).some((c) => c.action.type === 'input'))
  assert.ok(!suggest('entrada tv', CATALOG, { inputs: false }).some((c) => c.action.type === 'input'))
})

// English and Portuguese keywords work in both UI languages; only titles follow the UI.
for (const lang of ['pt-BR', 'en'] as const) {
  const ctx = { lang, toast: true, inputs: true }
  const first = (q: string) => suggest(q, CATALOG, ctx)[0]
  const ids = (q: string) => suggest(q, CATALOG, ctx).map((c) => c.id)

  test(`bilingual keywords (UI ${lang})`, () => {
    assert.deepEqual(first('volume 20').action, { type: 'volume', level: 20 })
    assert.deepEqual(first('type hello').action, { type: 'text', text: 'hello' })
    assert.deepEqual(first('digitar ola').action, { type: 'text', text: 'ola' })
    assert.deepEqual(first('turn off in 30 min').action, { type: 'sleep', minutes: 30 })
    assert.deepEqual(first('desligar em 30 min').action, { type: 'sleep', minutes: 30 })
    assert.deepEqual(first('timer 45').action, { type: 'sleep', minutes: 45 })
    assert.ok(suggest('cancel timer', CATALOG, ctx).some((c) => c.action.type === 'sleep' && c.action.minutes === 0))
    assert.ok(suggest('cancelar timer', CATALOG, ctx).some((c) => c.action.type === 'sleep' && c.action.minutes === 0))
    assert.deepEqual(first('hdmi 2').action, { type: 'input', query: 'hdmi 2' })
    assert.deepEqual(first('input tv').action, { type: 'input', query: 'tv' })
    assert.deepEqual(first('entrada tv').action, { type: 'input', query: 'tv' })
    assert.deepEqual(first('notify tv hi').action, { type: 'toast', text: 'hi' })
    assert.deepEqual(first('avisar na tv oi').action, { type: 'toast', text: 'oi' })
    assert.equal(first('open netflix').app?.id, 'netflix')
    assert.equal(first('abrir netflix').app?.id, 'netflix')
    for (const q of ['mute', 'mudo']) assert.ok(ids(q).includes('mute'), q)
    for (const q of ['home', 'início']) assert.ok(ids(q).includes('home'), q)
    for (const q of ['back', 'voltar']) assert.ok(ids(q).includes('back'), q)
  })
}

test('titles follow the UI language', () => {
  const title = (q: string, lang: 'pt-BR' | 'en') => suggest(q, CATALOG, { lang, toast: true, inputs: true })[0].title
  assert.equal(title('mute', 'en'), 'Mute')
  assert.equal(title('mute', 'pt-BR'), 'Mudo')
  assert.equal(title('open netflix', 'en'), 'Open Netflix')
  assert.equal(title('open netflix', 'pt-BR'), 'Abrir Netflix')
  assert.equal(title('turn off in 30 min', 'en'), 'Turn the TV off in 30 min')
  assert.equal(title('timer 30', 'pt-BR'), 'Desligar a TV em 30 min')
  assert.equal(title('type hello', 'en'), 'Type “hello”')
  assert.equal(title('digitar oi', 'pt-BR'), 'Digitar “oi”')
  assert.equal(title('notify tv hi', 'en'), 'Show on the TV: “hi”')
  assert.equal(title('input tv', 'en'), 'Switch to input tv')
  assert.equal(title('volume 20', 'en'), 'Volume 20')
})

test('english search link', () => {
  const c = suggest('search cats on youtube', CATALOG, { lang: 'en' })[0]
  assert.ok(c.action.type === 'link' && c.action.url.endsWith('search_query=cats'))
})

test('sleep timer phrasings', () => {
  for (const q of ['turn off tv in 30', 'turn the tv off in 30', 'turn tv off in 30', 'turn off in 30', 'sleep 30', 'sleep in 30', 'desligar em 30']) {
    assert.deepEqual(suggest(q, CATALOG)[0].action, { type: 'sleep', minutes: 30 }, q)
  }
})

test('"type" and "input" alone suggest the static commands, not a parsed action', () => {
  const ctx = { inputs: true, toast: true }
  const t = suggest('type', CATALOG, ctx)
  assert.ok(!t.some((c) => c.action.type === 'text'))
  assert.deepEqual(t.find((c) => c.id === 'type')?.action, { type: 'ui', target: 'focus-text' })
  const i = suggest('input', CATALOG, ctx)
  assert.ok(!i.some((c) => c.action.type === 'input'))
  assert.deepEqual(i.find((c) => c.id === 'input')?.action, { type: 'key', key: 'TV_INPUT' })
})

for (const lang of ['pt-BR', 'en'] as const) {
  test(`"settings" ranks OPTIC settings first (UI ${lang})`, () => {
    assert.equal(suggest('settings', CATALOG, { lang })[0].id, 'settings')
  })
}
