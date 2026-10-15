// Small progressive enhancements: OS-aware download, install tabs with a
// sliding indicator, mobile menu, sticky-nav state and scroll reveals.
// The page works fully without this file.

const REL = 'https://github.com/andredeomondes/optic/releases/latest/download/'
const DOWNLOADS = {
  windows: { file: 'OPTIC-windows-amd64-setup.exe', label: 'Baixar para Windows' },
  macos: { file: 'OPTIC-macos-universal.dmg', label: 'Baixar para macOS' },
  linux: { file: 'OPTIC-linux-amd64.tar.gz', label: 'Baixar para Linux' },
}
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches

function detectOS() {
  const p = (navigator.userAgentData?.platform || navigator.platform || navigator.userAgent).toLowerCase()
  if (p.includes('mac')) return 'macos'
  if (p.includes('linux') || p.includes('x11')) return 'linux'
  if (p.includes('win')) return 'windows'
  return null
}

// ── install tabs ─────────────────────────────────────
const tabs = document.querySelector('.tabs')
const indicator = Object.assign(document.createElement('span'), { className: 'ind' })
tabs.prepend(indicator)

function moveIndicator() {
  const active = tabs.querySelector('[aria-selected="true"]')
  indicator.style.width = active.offsetWidth + 'px'
  indicator.style.transform = `translateX(${active.offsetLeft}px)`
}

function selectTab(os) {
  tabs.querySelectorAll('[role=tab]').forEach((b) => b.setAttribute('aria-selected', String(b.dataset.os === os)))
  document.querySelectorAll('[data-panel]').forEach((p) => (p.hidden = p.dataset.panel !== os))
  moveIndicator()
}

tabs.querySelectorAll('[role=tab]').forEach((b) => b.addEventListener('click', () => selectTab(b.dataset.os)))
addEventListener('resize', moveIndicator)
document.fonts?.ready.then(moveIndicator)

const os = detectOS()
if (os) {
  const btn = document.getElementById('dl-primary')
  btn.href = REL + DOWNLOADS[os].file
  btn.textContent = DOWNLOADS[os].label
  document.getElementById('dl-hint').innerHTML =
    'Grátis e open source · <a href="https://github.com/andredeomondes/optic/releases/latest">outros sistemas</a>'
}
selectTab(os || 'windows')

// ── nav: mobile menu, scrolled state, current section ─
const nav = document.querySelector('.nav')
const menu = nav.querySelector('.menu')
const setMenu = (open) => {
  nav.classList.toggle('open', open)
  menu.setAttribute('aria-expanded', String(open))
  menu.setAttribute('aria-label', open ? 'Fechar menu' : 'Abrir menu')
}
menu.addEventListener('click', () => setMenu(!nav.classList.contains('open')))
nav.querySelectorAll('nav a').forEach((a) => a.addEventListener('click', () => setMenu(false)))
addEventListener('keydown', (e) => e.key === 'Escape' && setMenu(false))

const onScroll = () => nav.classList.toggle('scrolled', scrollY > 8)
addEventListener('scroll', onScroll, { passive: true })
onScroll()

const links = new Map([...nav.querySelectorAll('nav a[href^="#"]')].map((a) => [a.getAttribute('href').slice(1), a]))
const sectionObserver = new IntersectionObserver(
  (entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue
      links.forEach((a) => a.classList.remove('current'))
      links.get(e.target.id)?.classList.add('current')
    }
  },
  { rootMargin: '-45% 0px -50% 0px' },
)
links.forEach((_, id) => {
  const el = document.getElementById(id)
  if (el) sectionObserver.observe(el)
})

// ── scroll reveal with per-group stagger ──────────────
const groups = ['.features article', '.block > h2, .block > .sub', '.steps li', '.duo figure', '.keys div', '.faq details', '.shot-sm', '.bullets li', '.note']
if (!reducedMotion && 'IntersectionObserver' in window) {
  const io = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) {
          e.target.classList.add('in')
          io.unobserve(e.target)
        }
      }
    },
    { rootMargin: '0px 0px -8% 0px', threshold: 0.08 },
  )
  // Safety net: anything already scrolled past (fast scrolls, anchor jumps,
  // printing) is shown even if the observer never fired for it.
  let ticking = false
  const sweep = () => {
    ticking = false
    for (const el of document.querySelectorAll('.reveal:not(.in)')) {
      if (el.getBoundingClientRect().top < innerHeight) el.classList.add('in')
    }
  }
  addEventListener('scroll', () => { if (!ticking) { ticking = true; requestAnimationFrame(sweep) } }, { passive: true })
  addEventListener('beforeprint', () => document.querySelectorAll('.reveal').forEach((el) => el.classList.add('in')))
  for (const sel of groups) {
    document.querySelectorAll(sel).forEach((el, i) => {
      el.classList.add('reveal')
      el.style.setProperty('--d', `${Math.min(i % 6, 5) * 70}ms`)
      io.observe(el)
    })
  }
}
