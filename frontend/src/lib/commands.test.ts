// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { normalize, suggest } from './commands.ts'
import type { AppDef } from './apps.ts'

const CATALOG: AppDef[] = [
  { id: 'youtube', name: 'YouTube', pkg: 'com.google.android.youtube.tv', brand: '#f00' },
  { id: 'netflix', name: 'Netflix', pkg: 'com.netflix.ninja', brand: '#e50914' },
]

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
