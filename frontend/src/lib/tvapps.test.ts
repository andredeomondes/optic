import { test } from 'node:test'
import assert from 'node:assert/strict'
import { filterApps, matchInput, normalize, sortApps } from './tvapps.ts'

const apps = [
  { id: '837', name: 'YouTube' },
  { id: '12', name: 'Netflix' },
  { id: 'g', name: 'Globoplay' },
  { id: 'a', name: 'Álbuns' },
]

test('normalize strips accents, case and spaces', () => {
  assert.equal(normalize('  ÁLBUM São '), 'album sao')
})

test('filterApps is accent-insensitive', () => {
  assert.deepEqual(filterApps(apps, 'GLOBO').map((a) => a.id), ['g'])
  assert.deepEqual(filterApps(apps, 'albun').map((a) => a.id), ['a'])
})

test('filterApps with empty query returns all sorted', () => {
  assert.deepEqual(filterApps(apps, '  ').map((a) => a.name), ['Álbuns', 'Globoplay', 'Netflix', 'YouTube'])
})

test('sortApps does not mutate the input', () => {
  const copy = [...apps]
  sortApps(apps)
  assert.deepEqual(apps, copy)
})

const inputs = [
  { id: 'tv', name: 'TV' },
  { id: 'h1', name: 'HDMI 1' },
  { id: 'h2', name: 'HDMI 2' },
  { id: 'av', name: 'AV' },
]

test('matchInput: exact, spacing-insensitive and prefix', () => {
  assert.equal(matchInput(inputs, 'tv')?.id, 'tv')
  assert.equal(matchInput(inputs, 'HDMI 2')?.id, 'h2')
  assert.equal(matchInput(inputs, 'hdmi2')?.id, 'h2')
  assert.equal(matchInput(inputs, '  hdmi   1 ')?.id, 'h1')
  assert.equal(matchInput(inputs, 'hd')?.id, 'h1')
  assert.equal(matchInput([{ id: 'h10', name: 'HDMI 10' }, ...inputs], 'hdmi 1')?.id, 'h1')
  assert.equal(matchInput([{ id: 'h10', name: 'HDMI 10' }], 'hdmi 1'), undefined)
})

test('matchInput: no match or empty query', () => {
  assert.equal(matchInput(inputs, 'vga'), undefined)
  assert.equal(matchInput(inputs, '  '), undefined)
  assert.equal(matchInput([], 'tv'), undefined)
})
