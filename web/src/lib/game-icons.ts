import { GAME_ICON_GLYPHS, type GameIconGlyph } from '@/lib/game-icon-glyphs'

function glyph(name: string): GameIconGlyph {
  const found = GAME_ICON_GLYPHS[name]
  if (!found) {
    throw new Error(`unknown game icon glyph: ${name}`)
  }
  return found
}

/**
 * Map artwork for the calamity overlay, from game-icons.net
 * (CC BY 3.0, icons by Delapouite). Glyphs are inline SVG paths so each
 * calamity can carry its own fill and outline.
 */
export interface CalamityIconStyle {
  glyph: GameIconGlyph
  count: number
  fill: string
  stroke?: string
  strokeWidth?: number
  opacity: number
}

export const CALAMITY_ICONS: Record<
  'plague' | 'bad_weather' | 'famine',
  CalamityIconStyle
> = {
  plague: {
    glyph: glyph('reaper-scythe'),
    count: 5,
    fill: '#30291f',
    opacity: 0.6,
  },
  bad_weather: {
    glyph: glyph('raining'),
    count: 4,
    fill: '#fff8e7',
    opacity: 0.8,
  },
  famine: {
    glyph: glyph('desert-skull'),
    count: 3,
    fill: '#fff8e7',
    stroke: '#30291f',
    strokeWidth: 16,
    opacity: 0.9,
  },
}

/**
 * Map artwork for the card-order overlay, from game-icons.net
 * (CC BY 3.0, icons by Delapouite).
 */
export interface CardIconStyle {
  glyph: GameIconGlyph
  className?: string
  fill: string
  stroke?: string
  strokeWidth?: number
  opacity: number
  /** Icon count per territory when the card scatters over a region. */
  count: number
}

export type CardIconKind = 'fair_weather' | 'abundant_harvest' | 'revolt'

export const CARD_ICONS: Record<CardIconKind, CardIconStyle> = {
  fair_weather: {
    glyph: glyph('sun'),
    fill: '#e3b341',
    stroke: '#fff8e7',
    strokeWidth: 16,
    opacity: 0.95,
    count: 3,
  },
  abundant_harvest: {
    glyph: glyph('wheat'),
    fill: '#4e7d3b',
    stroke: '#fff8e7',
    strokeWidth: 16,
    opacity: 0.95,
    count: 3,
  },
  revolt: {
    glyph: glyph('uprising'),
    // The villager color comes from the player who drafted the card.
    fill: '#30291f',
    stroke: '#fff8e7',
    strokeWidth: 14,
    opacity: 1,
    count: 1,
  },
}

/** The calamity kind each bonus card cancels when played on a region. */
export const CANCELED_KIND_BY_CARD: Record<
  'fair_weather' | 'abundant_harvest',
  'bad_weather' | 'famine'
> = {
  fair_weather: 'bad_weather',
  abundant_harvest: 'famine',
}

export interface SpecialOrderPlacement {
  kind: CardIconKind
  /** Region seed for weather cards, territory for revolts. */
  target: string
}

const SPECIAL_KIND_CODES: Record<string, CardIconKind> = {
  BT: 'fair_weather',
  FW: 'fair_weather',
  RA: 'abundant_harvest',
  AH: 'abundant_harvest',
  RE: 'revolt',
  RV: 'revolt',
}

/**
 * Parses the deck-order draft lines (`P KIND TARGET`) into overlay icon
 * placements. Discards (`D C KIND`) and malformed lines are ignored: the
 * overlay only shows cards about to be played.
 */
export function parseSpecialOrderPlacements(
  lines: string,
): SpecialOrderPlacement[] {
  const placements: SpecialOrderPlacement[] = []
  for (const rawLine of lines.split('\n')) {
    const fields = rawLine.trim().split(/\s+/)
    if (fields.length !== 3 || fields[0].toUpperCase() !== 'P') {
      continue
    }
    const kind = SPECIAL_KIND_CODES[fields[1].toUpperCase()]
    if (!kind) {
      continue
    }
    placements.push({ kind, target: fields[2] })
  }
  return placements
}
