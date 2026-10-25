// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { isNewer, isReleaseURL, parseVersion, RELEASES_URL, safeReleaseURL } from './version.ts'

test('parseVersion', () => {
  assert.deepEqual(parseVersion('v1.2.3'), { major: 1, minor: 2, patch: 3, pre: '' })
  assert.deepEqual(parseVersion('1.2.3-beta.1'), { major: 1, minor: 2, patch: 3, pre: 'beta.1' })
  assert.equal(parseVersion(''), null)
  assert.equal(parseVersion('dev'), null)
})

test('isNewer', () => {
  assert.equal(isNewer('1.0.2', 'v1.0.3'), true)
  assert.equal(isNewer('1.0.3', 'v1.0.3'), false)
  assert.equal(isNewer('1.0.10', '1.0.9'), false)
  assert.equal(isNewer('1.0.9', '1.0.10'), true)
  assert.equal(isNewer('1.0.2', 'v1.1.0-beta.1'), false)
  assert.equal(isNewer('dev', 'v1.0.3'), false)
  assert.equal(isNewer('1.0.2', 'garbage'), false)
  assert.equal(isNewer('1.0.2', ''), false)
})

test('isReleaseURL only accepts https github links', () => {
  assert.equal(isReleaseURL('https://github.com/andredeomondes/optic/releases/tag/v1.0.4'), true)
  assert.equal(isReleaseURL('file:///C:/x'), false)
  assert.equal(isReleaseURL('http://github.com/a/b'), false)
  assert.equal(isReleaseURL('https://github.com.evil.com/a'), false)
  assert.equal(isReleaseURL(''), false)
})

test('safeReleaseURL falls back to the releases page', () => {
  assert.equal(safeReleaseURL('https://github.com/a/b'), 'https://github.com/a/b')
  assert.equal(safeReleaseURL('https://evil.com/'), RELEASES_URL)
})
