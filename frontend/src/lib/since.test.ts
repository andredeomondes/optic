import { test } from 'node:test'
import assert from 'node:assert/strict'
import { since } from '../hooks/useNow.ts'
import { translate } from './i18n.ts'

const MIN = 60_000

for (const [lang, now, minAgo, hourMin] of [
  ['en', 'just now', '5 min ago', '2 h 5 min ago'],
  ['pt-BR', 'agora', 'há 5 min', 'há 2 h 5 min'],
] as const) {
  test(`since() interpolates (${lang})`, () => {
    const t = (k: Parameters<typeof translate>[1], v?: Record<string, string | number>) => translate(lang, k, v)
    assert.equal(since(0, 30_000, t), now)
    assert.equal(since(0, 5 * MIN, t), minAgo)
    assert.equal(since(0, 125 * MIN, t), hourMin)
  })
}
