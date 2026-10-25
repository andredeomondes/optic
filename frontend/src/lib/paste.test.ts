// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { cleanPaste, shouldPasteToTV } from './paste.ts'

test('shouldPasteToTV', () => {
  const v = { ctrlKey: true, metaKey: false, key: 'v' }
  assert.equal(shouldPasteToTV(v, { tagName: 'BUTTON' }, true), true)
  assert.equal(shouldPasteToTV({ ctrlKey: false, metaKey: true, key: 'V' }, null, true), true)
  assert.equal(shouldPasteToTV(v, { tagName: 'INPUT' }, true), false)
  assert.equal(shouldPasteToTV(v, { tagName: 'DIV', isContentEditable: true }, true), false)
  assert.equal(shouldPasteToTV(v, { tagName: 'BUTTON' }, false), false)
  assert.equal(shouldPasteToTV({ ...v, ctrlKey: false }, null, true), false)
  assert.equal(shouldPasteToTV({ ...v, shiftKey: true }, null, true), false)
  assert.equal(shouldPasteToTV({ ...v, altKey: true }, null, true), false)
  assert.equal(shouldPasteToTV({ ...v, repeat: true }, null, true), false)
})

test('cleanPaste', () => {
  assert.equal(cleanPaste('  senha\r\n123 '), 'senha 123')
  assert.equal(cleanPaste('a\rb  \t c'), 'a b c')
  assert.equal(cleanPaste('x'.repeat(600)).length, 500)
})
