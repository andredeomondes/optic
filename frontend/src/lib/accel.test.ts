import { test } from 'node:test'
import assert from 'node:assert/strict'
import { accelFromEvent, labelFor } from './accel.ts'

const ev = (key: string, code = '', mods: Partial<Record<'ctrlKey' | 'altKey' | 'shiftKey' | 'metaKey', boolean>> = {}) => ({
  key,
  code,
  ctrlKey: false,
  altKey: false,
  shiftKey: false,
  metaKey: false,
  ...mods,
})

test('accelFromEvent table', () => {
  const cases: [ReturnType<typeof ev>, string | null][] = [
    [ev('ArrowUp', 'ArrowUp', { ctrlKey: true, altKey: true }), 'Ctrl+Alt+Up'],
    [ev('ArrowLeft', 'ArrowLeft', { altKey: true }), 'Alt+Left'],
    [ev('k', 'KeyK', { ctrlKey: true, altKey: true }), 'Ctrl+Alt+K'],
    [ev('K', 'KeyK', { ctrlKey: true, shiftKey: true }), 'Ctrl+Shift+K'],
    [ev('¬', 'KeyL', { ctrlKey: true, altKey: true }), 'Ctrl+Alt+L'],
    [ev('5', 'Digit5', { ctrlKey: true }), 'Ctrl+5'],
    [ev('F5', 'F5'), 'F5'],
    [ev('F12', 'F12', { shiftKey: true }), 'Shift+F12'],
    [ev(' ', 'Space', { ctrlKey: true }), 'Ctrl+Space'],
    [ev('PageDown', 'PageDown', { ctrlKey: true }), 'Ctrl+PageDown'],
    [ev('MediaPlayPause', 'MediaPlayPause'), 'MediaPlayPause'],
    [ev('MediaTrackNext', 'MediaTrackNext'), 'MediaNext'],
    [ev('AudioVolumeMute', 'AudioVolumeMute'), 'VolumeMute'],
    [ev('Control', 'ControlLeft', { ctrlKey: true }), null],
    [ev('Shift', 'ShiftLeft', { shiftKey: true }), null],
    [ev('Alt', 'AltLeft', { ctrlKey: true, altKey: true }), null],
    [ev('Meta', 'MetaLeft', { metaKey: true }), null],
    [ev('a', 'KeyA', { metaKey: true }), null],
    [ev('Escape', 'Escape'), null],
    [ev('F13', 'F13'), null],
    [ev('z', 'KeyY', { ctrlKey: true, altKey: true }), 'Ctrl+Alt+Z'], // QWERTZ
    [ev('/', 'KeyQ', { ctrlKey: true, altKey: true }), 'Ctrl+Alt+Q'], // AltGr symbol: fall back to code
    [ev('!', 'Digit1', { shiftKey: true }), 'Shift+1'],
    [ev('1', 'Numpad1'), null],
    [ev('Enter', 'NumpadEnter'), null],
  ]
  for (const [e, want] of cases) assert.equal(accelFromEvent(e), want, `${e.key} ${e.code}`)
})

test('modifier order is Ctrl, Alt, Shift', () => {
  const e = ev('h', 'KeyH', { shiftKey: true, altKey: true, ctrlKey: true })
  assert.equal(accelFromEvent(e), 'Ctrl+Alt+Shift+H')
})

test('labelFor', () => {
  assert.equal(labelFor('Ctrl+Alt+Up'), 'Ctrl + Alt + ↑')
  assert.equal(labelFor('Ctrl+Alt+M'), 'Ctrl + Alt + M')
  assert.equal(labelFor('MediaPlayPause'), 'MediaPlayPause')
  assert.equal(labelFor(''), '')
})
