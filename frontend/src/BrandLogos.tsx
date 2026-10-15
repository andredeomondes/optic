import { SIMPLE_ICON_PATHS } from './logosData'

// Streaming app logos, drawn as inline SVG so the app works offline.
// YouTube and Netflix marks come from Simple Icons (CC0); Prime Video and
// Globoplay have no Simple Icons entry, so they're simplified recreations.
// All marks are trademarks of their owners, used only to label app shortcuts.

type LogoProps = { size?: number }

export function YouTubeLogo({ size = 22 }: LogoProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#FF0000"
        d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814z"
      />
      <path fill="#FFFFFF" d="M9.545 15.568V8.432L15.818 12l-6.273 3.568z" />
    </svg>
  )
}

export function NetflixLogo({ size = 22 }: LogoProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#E50914"
        d="m5.398 0 8.348 23.602c2.346.059 4.856.398 4.856.398L10.113 0H5.398zm8.489 0v9.172l4.715 13.33V0h-4.715zM5.398 1.5V24c1.873-.225 2.81-.312 4.715-.398V14.83L5.398 1.5z"
      />
    </svg>
  )
}

export function PrimeVideoLogo({ size = 22 }: LogoProps) {
  return (
    <svg width={size * 1.6} height={size} viewBox="0 0 38 24" aria-hidden="true">
      <text
        x="19"
        y="13"
        textAnchor="middle"
        fill="#FFFFFF"
        fontFamily="'Segoe UI', Arial, sans-serif"
        fontWeight="700"
        fontSize="11"
        letterSpacing="-0.3"
      >
        prime
      </text>
      <path d="M7 16.2c7 4.2 17 4.2 24 0" fill="none" stroke="#1A98FF" strokeWidth="1.8" strokeLinecap="round" />
      <path d="M28.6 14.6l3 1.4-1.6 2.8" fill="none" stroke="#1A98FF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function GloboplayLogo({ size = 22 }: LogoProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <defs>
        <linearGradient id="gp" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#FF2D6F" />
          <stop offset="1" stopColor="#FF6A1A" />
        </linearGradient>
      </defs>
      <circle cx="12" cy="12" r="11" fill="url(#gp)" />
      <rect x="5.5" y="7" width="13" height="10" rx="3.2" fill="#FFFFFF" />
      <path d="M10.4 9.6v4.8l4.1-2.4z" fill="#FF3D55" />
    </svg>
  )
}

// Disney+ isn't in Simple Icons (removed at the owner's request): simplified
// recreation of its app tile — wordmark under the signature arc.
export function DisneyPlusLogo({ size = 22 }: LogoProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 13.5C6 6.5 18 6.5 21 13.5" fill="none" stroke="#7FD3FF" strokeWidth="1.1" strokeLinecap="round" />
      <text
        x="12"
        y="17.2"
        textAnchor="middle"
        fill="#FFFFFF"
        fontFamily="'Segoe Script', 'Brush Script MT', cursive"
        fontWeight="700"
        fontSize="7.6"
      >
        Disney+
      </text>
    </svg>
  )
}

/**
 * How each app's tile looks: background plus either a drawn logo component
 * or a Simple Icons path in `fg`. Mirrors Android TV launcher tiles.
 */
const TILES: Record<string, { bg: string; fg?: string; logo?: (p: LogoProps) => JSX.Element; scale?: number }> = {
  youtube: { bg: '#FFFFFF', logo: YouTubeLogo },
  netflix: { bg: '#000000', logo: NetflixLogo, scale: 0.85 },
  prime: { bg: '#0F171E', logo: PrimeVideoLogo, scale: 0.62 },
  globoplay: { bg: 'linear-gradient(135deg, #3A0A1C, #1A0610)', logo: GloboplayLogo },
  disney: { bg: 'linear-gradient(160deg, #0B1442, #113CCF)', logo: DisneyPlusLogo, scale: 1.25 },
  max: { bg: '#FFFFFF', fg: '#002BE7', scale: 0.8 },
  spotify: { bg: '#000000', fg: '#1DB954' },
  twitch: { bg: '#9146FF', fg: '#FFFFFF', scale: 0.85 },
  crunchyroll: { bg: '#F47521', fg: '#FFFFFF' },
  plex: { bg: '#1F1F1F', fg: '#EBAF00', scale: 0.85 },
  kodi: { bg: '#17B2E7', fg: '#FFFFFF' },
  vlc: { bg: '#FFFFFF', fg: '#FF8800' },
  playstore: { bg: '#FFFFFF', fg: '#01875F', scale: 0.85 },
  paramount: { bg: '#0064FF', fg: '#FFFFFF' },
  appletv: { bg: '#000000', fg: '#FFFFFF' },
  deezer: { bg: '#A238FF', fg: '#FFFFFF', scale: 0.85 },
  ytmusic: { bg: '#FF0000', fg: '#FFFFFF' },
  ytkids: { bg: '#FFFFFF', fg: '#FF0000' },
  steamlink: { bg: '#171A21', fg: '#FFFFFF' },
}

/** App tile: brand logo when known, otherwise the app's initial on its brand color. */
export function AppIcon({ id, name, brand, size = 22 }: { id: string; name: string; brand: string; size?: number }) {
  const tile = TILES[id]
  const box = Math.round(size * 1.45)
  const style = { width: box, height: box, background: tile?.bg ?? brand, borderRadius: Math.round(box * 0.26) }
  const inner = Math.round(size * (tile?.scale ?? 0.9))

  if (tile?.logo) {
    const Logo = tile.logo
    return (
      <span className="app-icon" style={style}>
        <Logo size={inner} />
      </span>
    )
  }
  const path = SIMPLE_ICON_PATHS[id]
  if (path && tile?.fg) {
    return (
      <span className="app-icon" style={style}>
        <svg width={inner} height={inner} viewBox="0 0 24 24" aria-hidden="true">
          <path d={path} fill={tile.fg} />
        </svg>
      </span>
    )
  }
  const initials = name.replace(/[^\p{L}\p{N}+ ]/gu, '').slice(0, name.length > 3 ? 1 : 3)
  return (
    <span className="app-icon mono" style={{ ...style, fontSize: size * 0.55 }}>
      {initials}
    </span>
  )
}
