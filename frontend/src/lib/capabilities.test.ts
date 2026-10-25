// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { PLATFORM_LABEL, can, canConnect, isSupported } from './capabilities.ts'
import type { DeviceInfo, TV } from './types.ts'

test('isSupported', () => {
  assert.equal(isSupported('androidtv'), true)
  assert.equal(isSupported(''), true)
  assert.equal(isSupported('roku'), true)
  assert.equal(isSupported('webos'), false)
})

test('canConnect', () => {
  const tv = (platform: TV['platform'], remotev2: boolean): TV => ({ name: 'TV', model: '', ip: '10.0.0.2', remotev2, platform })
  assert.equal(canConnect(tv('roku', false)), true)
  assert.equal(canConnect(tv('androidtv', false)), false)
  assert.equal(canConnect(tv('androidtv', true)), true)
  assert.equal(canConnect(tv('webos', false)), false)
})

test('can', () => {
  assert.equal(can(null, 'text'), false)
  assert.equal(can({ capabilities: { text: true } } as DeviceInfo, 'text'), true)
  assert.equal(can({ capabilities: { text: true } } as DeviceInfo, 'appList'), false)
})

test('platform labels mirror the backend', () => {
  assert.equal(PLATFORM_LABEL.androidtv, 'Android TV')
  assert.equal(PLATFORM_LABEL.tizen, 'Samsung')
  assert.equal(PLATFORM_LABEL.webos, 'LG webOS')
  assert.equal(PLATFORM_LABEL.roku, 'Roku')
  assert.equal(PLATFORM_LABEL[''], 'Desconhecida')
})
