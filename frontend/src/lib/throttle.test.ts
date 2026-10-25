// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createThrottle, wheelStep, wheelVolumeStep } from './throttle.ts'

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

test('wheelVolumeStep', () => {
  assert.equal(wheelVolumeStep({ deltaX: 0, deltaY: -100, ctrlKey: false }), 'VOLUME_UP')
  assert.equal(wheelVolumeStep({ deltaX: 0, deltaY: 53, ctrlKey: false }), 'VOLUME_DOWN')
  // Trackpad pinch-zoom arrives as a wheel event with ctrlKey set.
  assert.equal(wheelVolumeStep({ deltaX: 0, deltaY: -100, ctrlKey: true }), null)
  // Horizontal swipe: the panel should scroll sideways, not change volume.
  assert.equal(wheelVolumeStep({ deltaX: 80, deltaY: 20, ctrlKey: false }), null)
  assert.equal(wheelVolumeStep({ deltaX: -80, deltaY: -20, ctrlKey: false }), null)
  // Diagonal with vertical dominant still counts as volume.
  assert.equal(wheelVolumeStep({ deltaX: 10, deltaY: -40, ctrlKey: false }), 'VOLUME_UP')
  assert.equal(wheelVolumeStep({ deltaX: 0, deltaY: 0.4, ctrlKey: false }), null)
})
