import { test } from 'node:test'
import assert from 'node:assert/strict'
import { en, ptBR } from './i18n.ts'
import { isUpdateBusy, lastResultNotice, updatePercent, updateStateKey } from './updates.ts'

test('updatePercent', () => {
  assert.equal(updatePercent(0, 100), 0)
  assert.equal(updatePercent(50, 200), 25)
  assert.equal(updatePercent(999, 1000), 99)
  assert.equal(updatePercent(1000, 1000), 100)
  assert.equal(updatePercent(1200, 1000), 100)
  assert.equal(updatePercent(10, 0), null)
  assert.equal(updatePercent(10, -1), null)
})

test('isUpdateBusy', () => {
  for (const s of ['downloading', 'verifying', 'installing'] as const) assert.ok(isUpdateBusy(s), s)
  for (const s of ['idle', 'ready', 'error'] as const) assert.ok(!isUpdateBusy(s), s)
})

test('updateStateKey per state and kind', () => {
  assert.equal(updateStateKey('idle', 'installer'), null)
  assert.equal(updateStateKey('downloading', 'portable'), 'update.state.downloading')
  assert.equal(updateStateKey('verifying', 'dmg'), 'update.state.verifying')
  assert.equal(updateStateKey('installing', 'installer'), 'update.state.installingWindows')
  assert.equal(updateStateKey('installing', 'portable'), 'update.state.installingWindows')
  assert.equal(updateStateKey('installing', 'dmg'), 'update.state.opening')
  assert.equal(updateStateKey('ready', 'dmg'), 'update.state.readyInstaller')
  assert.equal(updateStateKey('ready', 'tarball'), 'update.state.readyFolder')
  assert.equal(updateStateKey('error', ''), 'update.state.error')
})

test('every state key exists in both languages', () => {
  for (const s of ['downloading', 'verifying', 'installing', 'ready', 'error'] as const) {
    for (const k of ['installer', 'portable', 'dmg', 'tarball', ''] as const) {
      const key = updateStateKey(s, k)!
      assert.ok(ptBR[key] && en[key], key)
    }
  }
})

test('lastResultNotice', () => {
  assert.equal(lastResultNotice(null), null)
  assert.deepEqual(lastResultNotice({ version: 'v1.1.0', ok: true }), { kind: 'ok', key: 'update.done', vars: { version: '1.1.0' } })
  assert.deepEqual(lastResultNotice({ version: '1.1.0', ok: false }), { kind: 'err', key: 'update.notInstalled' })
  assert.equal(
    ptBR['update.notInstalled'],
    'A atualização não foi instalada. O Windows pode ter bloqueado o instalador (Controle Inteligente de Aplicativos). Baixe pela página de releases.',
  )
})
