import { test } from 'node:test'
import assert from 'node:assert/strict'
import { filterApps, normalize, sortApps } from './tvapps.ts'

const apps = [
  { id: '837', name: 'YouTube' },
  { id: '12', name: 'Netflix' },
  { id: 'g', name: 'Globoplay' },
  { id: 'a', name: 'Álbuns' },
]

test('normalize strips accents, case and spaces', () => {
  assert.equal(normalize('  ÁLBUM São '), 'album sao')
})

test('filterApps is accent-insensitive', () => {
  assert.deepEqual(filterApps(apps, 'GLOBO').map((a) => a.id), ['g'])
  assert.deepEqual(filterApps(apps, 'album').map((a) => a.id), ['a'])
})

test('filterApps with empty query returns all sorted', () => {
  assert.deepEqual(filterApps(apps, '  ').map((a) => a.name), ['Álbuns', 'Globoplay', 'Netflix', 'YouTube'])
})

test('sortApps does not mutate the input', () => {
  const copy = [...apps]
  sortApps(apps)
  assert.deepEqual(apps, copy)
})
