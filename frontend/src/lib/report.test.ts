// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { buildReport } from './report.ts'

test('buildReport masks MAC and caps log', () => {
  const log = Array.from({ length: 250 }, (_, i) => ({ id: i, dir: '<', text: `line ${i}`, time: '00:00' }))
  const r = buildReport({
    version: '1.0.2',
    userAgent: 'x',
    host: '192.168.0.18',
    info: null,
    tvs: [{ name: 'UnionTV', model: '', ip: '192.168.0.18', remotev2: true, platform: 'androidtv', mac: 'aa:bb:cc:dd:ee:ff' }],
    log,
  })
  assert.ok(r.includes('aa:bb:cc:**:**:**'))
  assert.ok(!r.includes('dd:ee:ff'))
  assert.ok(r.includes('line 249') && !r.includes('line 49\n'))
  assert.ok(r.includes('| UnionTV | 192.168.0.18 | Android TV |'))
})

const base = { version: '1.0.2', userAgent: 'x', host: '', info: null, tvs: [], log: [] }
const tv = (over: object) => ({ name: 'TV', model: '', ip: '10.0.0.2', remotev2: true, platform: 'androidtv' as const, ...over })

test('masks MAC inside a log line', () => {
  const r = buildReport({ ...base, log: [{ id: 1, dir: '<', text: 'arp aa:bb:cc:dd:ee:ff seen', time: '00:00' }] })
  assert.ok(r.includes('aa:bb:cc:**:**:**') && !r.includes('dd:ee:ff'))
})

test('masks uppercase and dash-separated MACs', () => {
  const r = buildReport({ ...base, tvs: [tv({ mac: 'AA-BB-CC-DD-EE-FF' }), tv({ name: 'B', mac: 'AA:BB:CC:11:22:33' })] })
  assert.ok(!/DD-EE-FF|11:22:33/i.test(r))
})

test('malformed or unseparated mac never prints in full', () => {
  for (const mac of ['aabbccddeeff', 'aabb.ccdd.eeff', ' aa:bb:cc:dd:ee:ff ', 'aa:bb:cc:dd:ee', 'garbage']) {
    const r = buildReport({ ...base, tvs: [tv({ mac })] })
    assert.ok(!/ddeeff|dd:ee|ccdd\.eeff|eeff/i.test(r), mac)
  }
  assert.ok(buildReport({ ...base, tvs: [tv({ mac: 'garbage' })] }).includes('**:**:**:**:**:**'))
})

test('masks MAC embedded in TV name and info.model, and unseparated token in log', () => {
  const info = { model: 'X aa:bb:cc:dd:ee:ff', vendor: 'V', version: '12', appVersion: '1', platform: 'androidtv' as const,
    capabilities: { text: true, appList: false, absoluteVolume: true, powerOn: false, inputs: false, pointer: false, toast: false } }
  const r = buildReport({ ...base, host: '10.0.0.2', info, tvs: [tv({ name: 'TV aabbccddeeff' })],
    log: [{ id: 1, dir: '<', text: 'hex 0123456789abcdef0123', time: '00:00' }] })
  assert.ok(!r.includes('ddeeff') && !r.includes('dd:ee:ff'))
  assert.ok(r.includes('0123456789abcdef0123'))
})

test('connected section renders vendor, model and capabilities', () => {
  const info = { model: 'M1', vendor: 'Acme', version: '12', appVersion: '1.2', platform: 'androidtv' as const,
    capabilities: { text: true, appList: false, absoluteVolume: true, powerOn: false, inputs: false, pointer: false, toast: false } }
  const r = buildReport({ ...base, host: '10.0.0.2', info })
  assert.ok(r.includes('- Fabricante: Acme') && r.includes('- Modelo: M1') && r.includes('text=sim') && r.includes('appList=não'))
})

test('empty states and pipe escaping', () => {
  const r = buildReport(base)
  assert.ok(r.includes('Nenhuma TV encontrada.') && r.includes('Nenhuma TV conectada.'))
  assert.ok(buildReport({ ...base, tvs: [tv({ name: 'a|b' })] }).includes('| a\\|b |'))
})
