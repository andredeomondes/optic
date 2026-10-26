// Run with: npm test (Node's built-in test runner + type stripping).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { detectLang, en, placeholders, ptBR, resolveLang, translate } from './i18n.ts'

test('detectLang', () => {
  assert.equal(detectLang(['pt-PT']), 'pt-BR')
  assert.equal(detectLang(['pt-BR']), 'pt-BR')
  assert.equal(detectLang(['en-US', 'pt-BR']), 'en')
  assert.equal(detectLang([]), 'en')
  assert.equal(detectLang(['es']), 'en')
})

test('resolveLang', () => {
  assert.equal(resolveLang('en', ['pt-BR']), 'en')
  assert.equal(resolveLang('pt-BR', ['en-US']), 'pt-BR')
  assert.equal(resolveLang('auto', ['pt-BR']), 'pt-BR')
  assert.equal(resolveLang('auto', ['fr']), 'en')
})

test('translate', () => {
  assert.equal(translate('pt-BR', 'settings.language'), 'Idioma')
  assert.equal(translate('en', 'settings.language'), 'Language')
  const dict = ptBR as Record<string, string>
  dict['test.vars'] = 'Olá {name}, {n} e {x}'
  ;(en as Record<string, string>)['test.vars'] = 'Hi {name}, {n} and {x}'
  try {
    assert.equal(translate('en', 'test.vars' as keyof typeof ptBR, { name: 'Ana', n: 2 }), 'Hi Ana, 2 and {x}')
  } finally {
    delete dict['test.vars']
    delete (en as Record<string, string>)['test.vars']
  }
})

test('placeholders', () => {
  assert.deepEqual(placeholders('{y} a {x} {y}'), ['x', 'y'])
  assert.deepEqual(placeholders('none'), [])
})

test('dictionaries are complete and consistent', () => {
  for (const key of Object.keys(ptBR) as (keyof typeof ptBR)[]) {
    assert.ok(en[key] && en[key].trim(), `en.${key} is empty`)
    assert.deepEqual(placeholders(en[key]), placeholders(ptBR[key]), `placeholders differ for ${key}`)
  }
  assert.deepEqual(Object.keys(en).sort(), Object.keys(ptBR).sort())
})
