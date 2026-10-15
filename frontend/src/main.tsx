import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
// Bundled fonts (latin subset, woff2): offline and no layout shift.
import '@fontsource/inter/latin-400.css'
import '@fontsource/inter/latin-500.css'
import '@fontsource/inter/latin-600.css'
import '@fontsource/space-grotesk/latin-600.css'
import './styles.css'

// In a plain browser (`npm run dev`), fake the Go backend.
if (import.meta.env.DEV && !(window as any).go) {
  await import('./devStub')
}

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
