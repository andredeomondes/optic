// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { PLATFORM_LABEL, can, platformOSLabel, canConnect, canPowerOn, isSupported } from './capabilities.ts'
import type { DeviceInfo, TV } from './types.ts'

test('isSupported', () => {
  assert.equal(isSupported('androidtv'), true)
  assert.equal(isSupported(''), true)
  assert.equal(isSupported('roku'), true)
  assert.equal(isSupported('webos'), true)
  assert.equal(isSupported('tizen'), true)
})

test('canConnect', () => {
  const tv = (platform: TV['platform'], remotev2: boolean): TV => ({ name: 'TV', model: '', ip: '10.0.0.2', remotev2, platform })
  assert.equal(canConnect(tv('roku', false)), true)
  assert.equal(canConnect(tv('androidtv', false)), false)
  assert.equal(canConnect(tv('androidtv', true)), true)
  assert.equal(canConnect(tv('webos', false)), true)
  assert.equal(canConnect(tv('tizen', false)), true)
})

test('canPowerOn', () => {
  const tv = (platform: TV['platform'], mac?: string): TV => ({ name: 'TV', model: '', ip: '10.0.0.2', remotev2: false, platform, mac })
  assert.equal(canPowerOn(tv('roku')), true)
  assert.equal(canPowerOn(tv('androidtv')), false)
  assert.equal(canPowerOn(tv('androidtv', 'aa:bb:cc:dd:ee:ff')), true)
  assert.equal(canPowerOn(tv('webos', 'aa:bb:cc:dd:ee:ff')), true)
  assert.equal(canPowerOn(tv('tizen', 'aa:bb:cc:dd:ee:ff')), true)
  assert.equal(canPowerOn(tv('tizen')), false)
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

test('platformOSLabel never leaves a trailing space', () => {
  assert.equal(platformOSLabel('tizen', '2018'), 'Tizen 2018')
  assert.equal(platformOSLabel('tizen', ''), 'Tizen')
  assert.equal(platformOSLabel('tizen'), 'Tizen')
  assert.equal(platformOSLabel('roku', '12.5'), 'Roku OS 12.5')
  assert.equal(platformOSLabel('webos', '6'), 'webOS 6')
  assert.equal(platformOSLabel('androidtv', '13'), 'Android 13')
  assert.equal(platformOSLabel(undefined, ''), 'Android')
})
