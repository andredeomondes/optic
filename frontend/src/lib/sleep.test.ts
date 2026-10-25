// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { decideAtDeadline, remaining, SLEEP_PRESETS } from './sleep.ts'

test('remaining rounds up', () => {
  assert.equal(remaining(10 * 60_000 + 1, 0), 11)
  assert.equal(remaining(10 * 60_000, 0), 10)
  assert.equal(remaining(0, 5), 0)
})

const H = { armedHost: '10.0.0.2', host: '10.0.0.2' }

test('decideAtDeadline never powers a tv that is off', () => {
  assert.equal(decideAtDeadline({ ...H, connected: true, tvOn: true }), 'power')
  assert.equal(decideAtDeadline({ ...H, connected: true, tvOn: false }), 'skip')
  assert.equal(decideAtDeadline({ ...H, connected: false, tvOn: true }), 'skip')
})

test('decideAtDeadline never powers a different tv', () => {
  assert.equal(decideAtDeadline({ connected: true, tvOn: true, armedHost: '10.0.0.2', host: '10.0.0.3' }), 'changed')
})

test('presets are ascending minutes', () => {
  assert.deepEqual([...SLEEP_PRESETS], [15, 30, 60, 90])
})
