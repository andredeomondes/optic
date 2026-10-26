import { test } from 'node:test'
import assert from 'node:assert/strict'
import { appsForPlatform, CATALOG, launchLink, makeTVApp } from './apps.ts'

test('launchLink keeps the market link for Android packages', () => {
  assert.equal(launchLink(CATALOG[0]), 'market://launch?id=com.google.android.youtube.tv')
})

test('launchLink returns the bare id for TV apps', () => {
  const app = makeTVApp({ id: '837', name: 'YouTube' }, 'roku')
  assert.equal(launchLink(app), '837')
  assert.equal(launchLink({ ...app, pkg: 'netflix' }), 'netflix')
})

test('makeTVApp builds a custom favorite keyed by tv:<id>', () => {
  assert.deepEqual(makeTVApp({ id: '12', name: 'Netflix' }, 'roku'), {
    id: 'tv:12',
    name: 'Netflix',
    pkg: '12',
    brand: '#8B5CF6',
    custom: true,
    platform: 'roku',
  })
})

test('appsForPlatform keeps unscoped apps and the matching platform only', () => {
  const roku = makeTVApp({ id: '12', name: 'Netflix' }, 'roku')
  const lg = makeTVApp({ id: 'netflix', name: 'Netflix' }, 'webos')
  const custom = { id: 'custom:x.y', name: 'X', pkg: 'x.y', brand: '#000', custom: true }
  const all = [CATALOG[0], custom, roku, lg]
  assert.deepEqual(appsForPlatform(all, 'roku'), [CATALOG[0], custom, roku])
  assert.deepEqual(appsForPlatform(all, 'webos'), [CATALOG[0], custom, lg])
  assert.deepEqual(appsForPlatform(all, undefined), [CATALOG[0], custom])
})
