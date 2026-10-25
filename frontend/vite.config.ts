import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Single source of truth for the app version: wails.json (bumped on release).
const wails = JSON.parse(readFileSync(new URL('../wails.json', import.meta.url), 'utf-8'))

export default defineConfig({
  plugins: [react()],
  define: { __APP_VERSION__: JSON.stringify(wails.info.productVersion) },
  // WebView2 is evergreen Chromium; esnext allows top-level await in main.tsx.
  build: { target: 'esnext' },
})
