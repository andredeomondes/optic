import { test } from 'node:test'
import assert from 'node:assert/strict'
import { appLabel, playingLabel } from './nowplaying.ts'

const catalog = [{ pkg: 'com.netflix.ninja', name: 'Netflix' }]
const installed = [
  { id: '837', name: 'YouTube' },
  { id: 'Globoplay', name: 'Globoplay' },
  { id: 'netflix', name: 'Netflix' },
]
const ctx = { installed, catalog }

test('appLabel: installed id gives its name', () => {
  assert.equal(appLabel('837', ctx), 'YouTube')
  assert.equal(appLabel('netflix', ctx), 'Netflix')
})

test('appLabel: Roku-style name equal to id passes through', () => {
  assert.equal(appLabel('Globoplay', ctx), 'Globoplay')
})

test('appLabel: catalog package', () => {
  assert.equal(appLabel('com.netflix.ninja', { installed: [], catalog }), 'Netflix')
})

test('appLabel: last segment capitalized', () => {
  assert.equal(appLabel('com.webos.app.livetv', { installed: [], catalog }), 'Livetv')
})

test('appLabel: empty id', () => {
  assert.equal(appLabel('', ctx), '')
})

test('playingLabel', () => {
  assert.equal(playingLabel('playing'), 'Tocando')
  assert.equal(playingLabel('paused'), 'Pausado')
  assert.equal(playingLabel(''), '')
  assert.equal(playingLabel('x'), '')
})
