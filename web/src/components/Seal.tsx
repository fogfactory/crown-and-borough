import { useId } from 'react'
import { deployedVersion } from '@/components/VersionBadge'
import { cn } from '@/lib/utils'
import {
  SEAL_LETTER,
  SEAL_WAX_DARK,
  SEAL_WAX_LIGHT,
  SEAL_WAX_RIM,
  SEAL_WAX_SHADOW,
} from '@/lib/map-palette'
import { SEAL_MONOGRAM_PATH, SEAL_MONOGRAM_WIDTH } from '@/components/seal-monogram'

/** Wax edge: a circle with a slightly uneven rim, in a 200 x 200 box. */
export function sealWaxPath(): string {
  const steps = 96
  const points: string[] = []
  for (let i = 0; i < steps; i++) {
    const a = (i / steps) * Math.PI * 2
    const r = 92 + 2.6 * Math.sin(7 * a + 0.4) + 1.6 * Math.sin(13 * a + 1.1)
    points.push(
      `${(100 + r * Math.cos(a)).toFixed(1)} ${(100 + r * Math.sin(a)).toFixed(1)}`,
    )
  }
  return `M${points.join('L')}Z`
}

const WAX_PATH = sealWaxPath()
const LEGEND_FONT_SIZE = 15
/** Baseline radius of the name (above) and of the version (below): both bands share the same ring. */
const NAME_RADIUS = 64
const VERSION_RADIUS = 75
/** Arc length between the middle of "and" and the inner end of the words beside it. */
const NAME_GAP = 23
const NAME_MIDDLE = (Math.PI * NAME_RADIUS) / 2
/** "and" sits on the axis of the seal; the other words hang off it. */
const NAME_WORDS = [
  { word: 'Crown', anchor: 'end', offset: NAME_MIDDLE - NAME_GAP },
  { word: 'and', anchor: 'middle', offset: NAME_MIDDLE },
  { word: 'Borough', anchor: 'start', offset: NAME_MIDDLE + NAME_GAP },
] as const

/** At most "vXX.XX.XX": the build number and pre-release suffix are dropped, and anything that is not a version (such as "dev") is kept short. */
export function sealVersion(version: string): string {
  const match = /^v?\d+(?:\.\d+){0,2}(?!\w)/.exec(version.trim())
  if (!match) return version.trim().slice(0, 8)
  return match[0].startsWith('v') ? match[0] : `v${match[0]}`
}

/**
 * Crown & Borough wax seal (specs/charte-graphique.md, Emblème).
 * `full` carries the circular legend (home, sign-in, 96 px and more; "Crown and Borough" above, the deployed version below);
 * `compact` keeps the monogram only (header, favicon sizes).
 */
export function Seal({
  variant = 'compact',
  className,
}: {
  variant?: 'full' | 'compact'
  className?: string
}) {
  const id = useId()
  const wax = `${id}-wax`
  const namePath = `${id}-name`
  const versionPath = `${id}-version`
  const full = variant === 'full'
  const scale = (full ? 78 : 108) / SEAL_MONOGRAM_WIDTH
  const ringRadius = full ? 86 : 84
  return (
    <svg viewBox="0 0 200 200" aria-hidden="true" className={cn('shrink-0', className)}>
      <defs>
        <linearGradient id={wax} x1="0.15" x2="0.85" y1="0" y2="1">
          <stop offset="0%" stopColor={SEAL_WAX_LIGHT} />
          <stop offset="100%" stopColor={SEAL_WAX_DARK} />
        </linearGradient>
        {full && (
          <>
            <path
              id={namePath}
              d={`M${100 - NAME_RADIUS} 100 A${NAME_RADIUS} ${NAME_RADIUS} 0 0 1 ${100 + NAME_RADIUS} 100`}
            />
            <path
              id={versionPath}
              d={`M${100 - VERSION_RADIUS} 100 A${VERSION_RADIUS} ${VERSION_RADIUS} 0 0 0 ${100 + VERSION_RADIUS} 100`}
            />
          </>
        )}
      </defs>
      <path d={WAX_PATH} fill={`url(#${wax})`} stroke={SEAL_WAX_SHADOW} strokeWidth={2} />
      <circle
        cx={100}
        cy={100}
        r={ringRadius}
        fill="none"
        stroke={SEAL_WAX_RIM}
        strokeWidth={full ? 2.5 : 4}
      />
      {full && (
        <>
          <circle
            cx={100}
            cy={100}
            r={55}
            fill="none"
            stroke={SEAL_WAX_RIM}
            strokeWidth={2}
          />
          <g
            fill={SEAL_LETTER}
            fontFamily="var(--font-display)"
            fontSize={LEGEND_FONT_SIZE}
            fontWeight={700}
          >
            {NAME_WORDS.map(({ word, anchor, offset }) => (
              <text key={word} textAnchor={anchor}>
                <textPath href={`#${namePath}`} startOffset={offset}>
                  {word}
                </textPath>
              </text>
            ))}
            <text textAnchor="middle">
              <textPath href={`#${versionPath}`} startOffset="50%">
                {sealVersion(deployedVersion())}
              </textPath>
            </text>
          </g>
        </>
      )}
      <g transform={`translate(100 101) scale(${scale})`}>
        <path d={SEAL_MONOGRAM_PATH} fill={SEAL_LETTER} />
      </g>
    </svg>
  )
}
