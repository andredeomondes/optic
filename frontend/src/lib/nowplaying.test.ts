import { test } from 'node:test'
import assert from 'node:assert/strict'
import { appLabel, nowPlayingText, playingKey } from './nowplaying.ts'

const catalog = [{ pkg: 'com.netflix.ninja', name: 'Netflix' }]
const installed = [
  { id: '837', name: 'YouTube' },
  { id: '2285', name: 'Globoplay' },
  { id: '9', name: 'MLB.TV' },
  { id: 'netflix', name: 'Netflix' },
]
const ctx = { installed, catalog }

test('appLabel: installed id gives its name', () => {
  assert.equal(appLabel('837', ctx), 'YouTube')
  assert.equal(appLabel('netflix', ctx), 'Netflix')
})

test('appLabel: Roku sends the channel name while ids are numeric', () => {
  assert.equal(appLabel('Globoplay', ctx), 'Globoplay')
  assert.equal(appLabel('MLB.TV', ctx), 'MLB.TV')
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

test('playingKey', () => {
  assert.equal(playingKey('playing'), 'now.playing')
  assert.equal(playingKey('paused'), 'now.paused')
  assert.equal(playingKey(''), '')
  assert.equal(playingKey('x'), '')
})

test('nowPlayingText: roku and webos use the app label', () => {
  assert.equal(nowPlayingText('YouTube', 'Tocando', 'roku', { installed: [{ id: '837', name: 'YouTube' }], catalog }), 'YouTube · Tocando')
  assert.equal(nowPlayingText('com.webos.app.livetv', '', 'webos', { installed: [], catalog }), 'Livetv')
})

test('nowPlayingText: androidtv only for catalog apps', () => {
  assert.equal(nowPlayingText('com.netflix.ninja', 'Pausado', 'androidtv', { installed: [], catalog }), 'Netflix · Pausado')
  assert.equal(nowPlayingText('com.google.android.tvlauncher', '', 'androidtv', { installed: [], catalog }), '')
})

test('nowPlayingText: empty id', () => {
  assert.equal(nowPlayingText('', 'Tocando', 'roku', ctx), '')
})
