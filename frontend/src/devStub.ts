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
const pushState = () => emit('state', { on: true, volume, volumeMax: 100, muted })

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
          { name: 'LG Sala', model: 'OLED55C3', ip: '192.168.0.40', remotev2: false, platform: 'webos' },
        ]
      },
      async LastHost() {
        return ''
      },
      async Connect(ip: string) {
        await wait(700)
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
        if (key === 'VOLUME_UP') volume = Math.min(100, volume + 1)
        if (key === 'VOLUME_DOWN') volume = Math.max(0, volume - 1)
        if (key === 'MUTE') muted = !muted
        if (key.startsWith('VOLUME') || key === 'MUTE') setTimeout(pushState, 120)
      },
      async SendText(text: string) {
        await wait(250)
        emit('trace', '>', `21 { 1: 0 2: 0 3 { 1: 1 2 { 3: "${text}" } } }`)
      },
      async SetVolume(level: number) {
        await wait(300)
        volume = level
        pushState()
      },
      async LaunchApp(id: string) {
        emit('trace', '>', `90 { 1: "${id}" }`)
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
      async StartPairing() {
        return 'UnionTV'
      },
      async FinishPairing() {
        await wait(600)
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
