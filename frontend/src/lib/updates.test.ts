import { test } from 'node:test'
import assert from 'node:assert/strict'
import { en, ptBR, type Lang } from './i18n.ts'
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

test('lastResultNotice: ok and nothing', () => {
  assert.equal(lastResultNotice(null, 'pt-BR'), null)
  assert.deepEqual(lastResultNotice({ version: 'v1.1.0', ok: true, error: '' }, 'pt-BR'), { kind: 'ok', text: 'Atualizado para 1.1.0' })
  assert.deepEqual(lastResultNotice({ version: '1.1.0', ok: true }, 'en'), { kind: 'ok', text: 'Updated to 1.1.0' })
})

test('lastResultNotice: empty reason (version did not change) makes no Smart App Control claim', () => {
  const pt = lastResultNotice({ version: '1.1.0', ok: false, error: '' }, 'pt-BR')
  assert.deepEqual(pt, { kind: 'err', text: 'A atualização não foi instalada. Baixe pela página de releases.' })
  assert.deepEqual(lastResultNotice({ version: '1.1.0', ok: false }, 'en'), {
    kind: 'err',
    text: "The update wasn't installed. Download it from the releases page.",
  })
})

test('lastResultNotice: app did not close in time', () => {
  const r = { version: '1.1.0', ok: false, error: 'O OPTIC não fechou a tempo.' }
  assert.deepEqual(lastResultNotice(r, 'pt-BR'), { kind: 'err', text: 'A atualização não foi feita: o OPTIC não fechou a tempo. Tente de novo.' })
  assert.deepEqual(lastResultNotice(r, 'en'), { kind: 'err', text: "The update wasn't applied: OPTIC didn't close in time. Try again." })
})

test('lastResultNotice: checksum re-check failed in the script', () => {
  const r = { version: '1.1.0', ok: false, error: 'checksum' }
  assert.deepEqual(lastResultNotice(r, 'pt-BR'), { kind: 'err', text: 'O download não confere (checksum). Tente de novo.' })
  assert.deepEqual(lastResultNotice(r, 'en'), { kind: 'err', text: "The download doesn't match its checksum. Try again." })
})

test('lastResultNotice: other reasons keep the Smart App Control hint and show the reason', () => {
  for (const lang of ['pt-BR', 'en'] as Lang[]) {
    const reason = 'A operação foi cancelada pelo usuário'
    const n = lastResultNotice({ version: '1.1.0', ok: false, error: reason }, lang)
    assert.equal(n?.kind, 'err')
    assert.equal(n?.text, (lang === 'en' ? en : ptBR)['update.notInstalled'])
    assert.equal(n?.detail, reason)
  }
  assert.equal(
    ptBR['update.notInstalled'],
    'A atualização não foi instalada. O Windows pode ter bloqueado o instalador (Controle Inteligente de Aplicativos). Baixe pela página de releases.',
  )
  const long = lastResultNotice({ version: '1.1.0', ok: false, error: 'x'.repeat(500) }, 'en')
  assert.equal(long?.detail?.length, 160)
})

test('ready texts say what to do next', () => {
  assert.equal(ptBR['update.state.readyInstaller'], 'Abrimos o DMG: feche o OPTIC e arraste-o para Aplicativos.')
  assert.equal(en['update.state.readyInstaller'], 'We opened the DMG: quit OPTIC and drag it to Applications.')
  assert.equal(ptBR['update.state.readyFolder'], 'Abrimos a pasta da nova versão: rode ./install.sh e abra o OPTIC de novo.')
  assert.equal(en['update.state.readyFolder'], "We opened the new version's folder: run ./install.sh and open OPTIC again.")
})
