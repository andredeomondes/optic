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
