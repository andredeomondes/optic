import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // WebView2 is evergreen Chromium; esnext allows top-level await in main.tsx.
  build: { target: 'esnext' },
})
