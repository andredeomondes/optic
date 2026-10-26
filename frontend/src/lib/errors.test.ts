import { test } from 'node:test'
import assert from 'node:assert/strict'
import { translateError } from './errors.ts'

test('pt-BR passes through', () => {
  assert.equal(translateError('Mensagem vazia', 'pt-BR'), 'Mensagem vazia')
  assert.equal(translateError('Tecla não suportada pela Roku: X', 'pt-BR'), 'Tecla não suportada pela Roku: X')
})

test('unknown and sentinel messages are unchanged', () => {
  assert.equal(translateError('algo novo', 'en'), 'algo novo')
  assert.equal(translateError('NOT_PAIRED', 'en'), 'NOT_PAIRED')
  assert.equal(translateError('webos: enviar x: y', 'en'), 'webos: enviar x: y')
  assert.equal(translateError('', 'en'), '')
})

test('one exact message per adapter', () => {
  assert.equal(translateError('Entrada desconhecida', 'en'), 'Unknown input')
  assert.equal(
    translateError('A TV LG não respondeu. Verifique se ela está ligada e tente de novo.', 'en'),
    "The LG TV didn't respond. Check that it's on and try again.",
  )
  assert.match(translateError("A TV Samsung não respondeu. Verifique se ela está ligada e tente de novo. Se aparecer 'Permitir' na TV, aceite e tente de novo.", 'en'), /^The Samsung TV/)
  assert.match(translateError('A Roku bloqueou o controle pela rede. Na TV: Configurações › Sistema › Configurações avançadas do sistema › Controle por apps móveis › Acesso à rede: Padrão.', 'en'), /^Roku blocked/)
  assert.equal(translateError('Esta TV não informa o volume: use Volume + e −.', 'en'), "This TV doesn't report its volume: use Volume + and −.")
  assert.equal(translateError('MAC da TV desconhecido: conecte uma vez com ela ligada', 'en'), 'TV MAC address unknown: connect once while it is on')
  assert.equal(translateError('Em uso por outro programa', 'en'), 'In use by another program')
  assert.equal(translateError('Conecte uma TV primeiro', 'en'), 'Connect a TV first')
})

test('pattern messages', () => {
  assert.equal(translateError('Este app não está instalado na TV Samsung.', 'en'), "This app isn't installed on the Samsung TV.")
  assert.equal(translateError('Este app não está instalado na TV LG.', 'en'), "This app isn't installed on the LG TV.")
  assert.equal(translateError('Tecla não suportada pela Roku: VOLUME_MUTE', 'en'), 'Key not supported by Roku: VOLUME_MUTE')
  assert.equal(translateError('Tecla desconhecida: Foo', 'en'), 'Unknown key: Foo')
  assert.equal(translateError('Atalho repetido: Ctrl+Alt+M', 'en'), 'Duplicate shortcut: Ctrl+Alt+M')
  assert.equal(translateError('código deve ter 6 caracteres', 'en'), 'The code must have 6 characters')
})

test('hotkey "<ação>: <motivo>" translates action and reason', () => {
  assert.equal(translateError('Mudo: Use Ctrl ou Alt junto com a tecla', 'en'), 'Mute: Use Ctrl or Alt together with the key')
  assert.equal(translateError('Volume +: Tecla desconhecida: Foo', 'en'), 'Volume +: Unknown key: Foo')
  assert.equal(translateError('Comandos rápidos: Atalho reservado pelo sistema', 'en'), 'Quick commands: Shortcut reserved by the system')
  assert.equal(translateError('Mídia: próxima: Em uso por outro programa', 'en'), 'Media: next: In use by another program')
})

test('generic "a: b" does not swallow other messages', () => {
  assert.equal(translateError('roku: HTTP 500', 'en'), 'roku: HTTP 500')
  assert.equal(translateError('Outra coisa: motivo', 'en'), 'Outra coisa: motivo')
})
