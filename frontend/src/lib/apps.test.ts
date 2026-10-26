import { test } from 'node:test'
import assert from 'node:assert/strict'
import { CATALOG, launchLink, makeTVApp } from './apps.ts'

test('launchLink keeps the market link for Android packages', () => {
  assert.equal(launchLink(CATALOG[0]), 'market://launch?id=com.google.android.youtube.tv')
})

test('launchLink returns the bare id for TV apps', () => {
  const app = makeTVApp({ id: '837', name: 'YouTube' })
  assert.equal(launchLink(app), '837')
  assert.equal(launchLink({ ...app, pkg: 'netflix' }), 'netflix')
})

test('makeTVApp builds a custom favorite keyed by tv:<id>', () => {
  assert.deepEqual(makeTVApp({ id: '12', name: 'Netflix' }), {
    id: 'tv:12',
    name: 'Netflix',
    pkg: '12',
    brand: '#8B5CF6',
    custom: true,
  })
})
