// Fake Go backend for `npm run dev` in a plain browser, so the UI can be
// designed without a TV. Never bundled into the Wails build (DEV only).

type Handler = (...args: any[]) => void
const handlers: Record<string, Handler[]> = {}
const emit = (name: string, ...args: any[]) => handlers[name]?.forEach((h) => h(...args))
const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))

let volume = 18
let closeToTray = true
let size = { w: 820, h: 880 }
let maximised = false
let muted = false
let webosPaired = false
let webosVolume = 12
let tizenPaired = false
let tizenKeyboard = false
const ROKU_HOST = '192.168.0.50'
const WEBOS_HOST = '192.168.0.40'
const TIZEN_HOST = '192.168.0.60'
let host = ''
let rokuPlaying: 'playing' | 'paused' = 'playing'
const pushState = () =>
  emit(
    'state',
    host === ROKU_HOST
      ? { on: true, volume: 0, volumeMax: 0, muted: false, app: 'YouTube', playing: rokuPlaying }
      : host === WEBOS_HOST
        ? { on: true, volume: webosVolume, volumeMax: 100, muted, app: 'netflix', playing: '' }
        : host === TIZEN_HOST
          ? { on: true, volume: 0, volumeMax: 0, muted: false, playing: '', keyboard: tizenKeyboard }
          : { on: true, volume, volumeMax: 100, muted, playing: '' },
  )

const w = window as any

w.runtime = {
  EventsOnMultiple(name: string, cb: Handler) {
    ;(handlers[name] ||= []).push(cb)
    return () => (handlers[name] = handlers[name].filter((h) => h !== cb))
  },
  EventsOff(name: string) {
    delete handlers[name]
  },
  WindowMinimise() {},
  WindowGetSize: async () => size,
  WindowSetAlwaysOnTop() {},
  WindowToggleMaximise() {
    maximised = !maximised
    window.dispatchEvent(new Event('resize'))
  },
  WindowIsMaximised: async () => maximised,
  WindowIsFullscreen: async () => Boolean(document.fullscreenElement),
  Quit() {},
  ClipboardGetText: async () => 'texto de teste',
  BrowserOpenURL(url: string) {
    window.open(url, '_blank')
  },
}

w.go = {
  main: {
    App: {
      async Discover() {
        await wait(900)
        return [
          { name: 'UnionTV', model: 'UnionTV', ip: '192.168.0.18', remotev2: true, platform: 'androidtv', mac: 'AA:BB:CC:DD:EE:FF' },
          { name: 'Quarto', model: 'Chromecast', ip: '192.168.0.71', remotev2: true, platform: 'androidtv' },
          { name: 'Sala Roku', model: 'TCL 43S5', ip: ROKU_HOST, remotev2: false, platform: 'roku' },
          { name: 'LG Sala', model: 'OLED55C1', ip: WEBOS_HOST, remotev2: false, platform: 'webos' },
          { name: 'Samsung Quarto', model: 'QN55Q60', ip: TIZEN_HOST, remotev2: false, platform: 'tizen' },
        ]
      },
      async LastHost() {
        return ''
      },
      async Connect(ip: string) {
        await wait(700)
        host = ip
        if (ip === ROKU_HOST) {
          emit('trace', '>', 'GET /query/device-info')
          setTimeout(pushState, 300)
          return {
            model: 'TCL 43S5', vendor: 'TCL', version: '12.5', appVersion: '', platform: 'roku',
            capabilities: { text: true, appList: true, absoluteVolume: false, powerOn: true, inputs: true, pointer: false, toast: false },
          }
        }
        if (ip === WEBOS_HOST) {
          if (!webosPaired) throw new Error('NOT_PAIRED')
          emit('trace', '>', 'register')
          setTimeout(pushState, 300)
          return {
            model: 'OLED55C1', vendor: 'LG', version: '6.0', appVersion: '', platform: 'webos',
            capabilities: { text: true, appList: true, absoluteVolume: true, powerOn: false, inputs: true, pointer: false, toast: true },
          }
        }
        if (ip === TIZEN_HOST) {
          if (!tizenPaired) throw new Error('NOT_PAIRED')
          emit('trace', '>', 'ms.channel.connect')
          setTimeout(pushState, 300)
          return {
            model: 'QN55Q60', vendor: 'Samsung', version: '6.5', appVersion: '', platform: 'tizen',
            capabilities: { text: true, appList: true, absoluteVolume: false, powerOn: false, inputs: false, pointer: false, toast: false },
          }
        }
        emit('trace', '<', `1 { 1: 639 2 { 1: "UnionTV" 2: "TCL" } }`)
        emit('trace', '>', `1 { 1: 622 2 { 1: "OPTIC" } }`)
        setTimeout(pushState, 300)
        return {
          model: 'UnionTV', vendor: 'TCL', version: '9', appVersion: '7.00', platform: 'androidtv',
          capabilities: { text: true, appList: false, absoluteVolume: false, powerOn: true, inputs: false, pointer: false, toast: false },
        }
      },
      async SendKey(key: string) {
        emit('trace', '>', `10 { 1: ${key} 2: 3 }`)
        // Stands in for a text field gaining focus on the Samsung.
        if (key === 'ENTER' && host === TIZEN_HOST) {
          tizenKeyboard = !tizenKeyboard
          setTimeout(pushState, 120)
        }
        if (key === 'VOLUME_UP') volume = Math.min(100, volume + 1)
        if (key === 'VOLUME_DOWN') volume = Math.max(0, volume - 1)
        if (key === 'MUTE') muted = !muted
        if (key === 'MEDIA_PLAY_PAUSE' && host === ROKU_HOST) {
          rokuPlaying = rokuPlaying === 'playing' ? 'paused' : 'playing'
          setTimeout(pushState, 120)
        }
        if (key.startsWith('VOLUME') || key === 'MUTE') setTimeout(pushState, 120)
      },
      async SendText(text: string) {
        await wait(250)
        emit('trace', '>', `21 { 1: 0 2: 0 3 { 1: 1 2 { 3: "${text}" } } }`)
      },
      async SetVolume(level: number) {
        await wait(300)
        if (host === WEBOS_HOST) webosVolume = level
        else volume = level
        pushState()
      },
      async LaunchApp(id: string) {
        emit('trace', '>', `90 { 1: "${id}" }`)
      },
      async InstalledApps() {
        await wait(400)
        if (host === ROKU_HOST) {
          return [
            { id: '12', name: 'Netflix' },
            { id: '837', name: 'YouTube' },
            { id: '2285', name: 'Globoplay' },
            { id: '13535', name: 'Plex' },
            { id: '291097', name: 'Disney+' },
            { id: '61322', name: 'Max' },
            { id: '74519', name: 'Pluto TV' },
            { id: '151908', name: 'The Roku Channel' },
          ]
        }
        if (host === WEBOS_HOST) {
          return [
            { id: 'netflix', name: 'Netflix' },
            { id: 'youtube.leanback.v4', name: 'YouTube' },
            { id: 'com.webos.app.hdmi1', name: 'HDMI 1' },
            { id: 'com.webos.app.livetv', name: 'Live TV' },
            { id: 'globoplay', name: 'Globoplay' },
          ]
        }
        if (host === TIZEN_HOST) {
          return [
            { id: '111299001912', name: 'YouTube' },
            { id: '3201907018807', name: 'Netflix' },
            { id: '3201908019041', name: 'Globoplay' },
          ]
        }
        throw new Error('Android TV ainda não é suportada')
      },
      async LatestRelease() {
        return {
          tag: 'v1.0.4',
          name: 'OPTIC 1.0.4',
          url: 'https://github.com/andredeomondes/optic/releases/tag/v1.0.4',
          notes: 'Correções e melhorias.',
          publishedAt: new Date().toISOString(),
        }
      },
      async PowerOn(ip: string) {
        await wait(300)
        emit('trace', '>', `wol ${ip}`)
      },
      async StartPairing(ip: string) {
        if (ip === TIZEN_HOST) return { name: 'Samsung Quarto', mode: 'prompt' }
        return ip === WEBOS_HOST ? { name: 'LG Sala', mode: 'prompt' } : { name: 'UnionTV', mode: 'code' }
      },
      async Inputs() {
        if (host === ROKU_HOST) {
          return [
            { id: 'InputTuner', name: 'TV' },
            { id: 'InputHDMI1', name: 'HDMI 1' },
            { id: 'InputHDMI2', name: 'HDMI 2' },
            { id: 'InputHDMI3', name: 'HDMI 3' },
            { id: 'InputHDMI4', name: 'HDMI 4' },
            { id: 'InputAV1', name: 'AV' },
          ]
        }
        return [
          { id: 'HDMI_1', name: 'HDMI 1' },
          { id: 'HDMI_2', name: 'HDMI 2' },
        ]
      },
      async SwitchInput(id: string) {
        emit('trace', '>', `input ${id}`)
      },
      async Toast(msg: string) {
        emit('trace', '>', `toast (${msg.length} caracteres)`)
      },
      async FinishPairing() {
        await wait(1500)
        if (host === TIZEN_HOST) tizenPaired = true
        else webosPaired = true
      },
      async CancelPairing() {},
      async TrayAvailable() {
        return true
      },
      async CloseToTray() {
        return closeToTray
      },
      async SetCloseToTray(on: boolean) {
        closeToTray = on
        emit('prefs:closeToTray', on)
      },
      async SetCompactMode() {},
      async ResizeWindow(_minW: number, _minH: number, w: number, h: number) {
        size = { w, h }
      },
      async ShowWindow() {},
      async QuitApp() {},
      async ToggleFullscreen() {
        if (document.fullscreenElement) await document.exitFullscreen().catch(() => {})
        else await document.documentElement.requestFullscreen?.().catch(() => {})
        return Boolean(document.fullscreenElement)
      },
      async Disconnect() {},
    },
  },
}

export {}
