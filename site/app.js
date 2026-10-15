// Picks the visitor's OS: selects its install tab and points the main
// download button at the right file.
const REL = 'https://github.com/andredeomondes/optic/releases/latest/download/'
const DOWNLOADS = {
  windows: { file: 'OPTIC-windows-amd64-setup.exe', label: 'Baixar para Windows' },
  macos: { file: 'OPTIC-macos-universal.dmg', label: 'Baixar para macOS' },
  linux: { file: 'OPTIC-linux-amd64.tar.gz', label: 'Baixar para Linux' },
}

function detectOS() {
  const p = (navigator.userAgentData?.platform || navigator.platform || navigator.userAgent).toLowerCase()
  if (p.includes('mac')) return 'macos'
  if (p.includes('linux') || p.includes('x11')) return 'linux'
  if (p.includes('win')) return 'windows'
  return null
}

function selectTab(os) {
  document.querySelectorAll('.tabs [role=tab]').forEach((b) => b.setAttribute('aria-selected', String(b.dataset.os === os)))
  document.querySelectorAll('[data-panel]').forEach((p) => (p.hidden = p.dataset.panel !== os))
}

document.querySelectorAll('.tabs [role=tab]').forEach((b) => b.addEventListener('click', () => selectTab(b.dataset.os)))

const os = detectOS()
if (os) {
  selectTab(os)
  const btn = document.getElementById('dl-primary')
  btn.href = REL + DOWNLOADS[os].file
  btn.textContent = DOWNLOADS[os].label
  document.getElementById('dl-hint').innerHTML =
    'Grátis e open source · <a href="https://github.com/andredeomondes/optic/releases/latest">outros sistemas</a>'
}
