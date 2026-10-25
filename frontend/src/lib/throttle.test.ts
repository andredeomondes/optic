// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createThrottle, wheelStep } from './throttle.ts'

test('wheelStep', () => {
  assert.equal(wheelStep(-100), 'VOLUME_UP')
  assert.equal(wheelStep(53), 'VOLUME_DOWN')
  assert.equal(wheelStep(0.4), null)
  assert.equal(wheelStep(-0.9), null)
  assert.equal(wheelStep(0), null)
})

test('createThrottle allows one call per window', () => {
  let t = 0
  const ok = createThrottle(120, () => t)
  assert.equal(ok(), true)
  t = 50
  assert.equal(ok(), false)
  t = 121
  assert.equal(ok(), true)
})
