import { test } from 'node:test'
import assert from 'node:assert/strict'
import { FOOTER_HINTS, SHORTCUTS } from './shortcuts.ts'

test('shortcut ids are unique', () => {
  const ids = SHORTCUTS.map((s) => s.id)
  assert.equal(new Set(ids).size, ids.length)
})

test('every footer hint id exists in SHORTCUTS', () => {
  for (const id of FOOTER_HINTS) assert.ok(SHORTCUTS.some((s) => s.id === id), `missing ${id}`)
})
