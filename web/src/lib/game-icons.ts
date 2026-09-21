
/**
 * Map artwork for the calamity overlay, from game-icons.net
 * (CC BY 3.0, icons by Delapouite):
 * https://game-icons.net/1x1/delapouite/reaper-scythe.html
 * https://game-icons.net/1x1/delapouite/raining.html
 * https://game-icons.net/1x1/delapouite/desert-skull.html
 *
 * The SVG files are served locally from /icons so the map never depends on
 * an external host. Glyphs are drawn black on transparent; the CSS classes
 * recolor them for readability (see the legend swatches).
 */
export interface CalamityIconStyle {
  src: string
  count: number
  className: string
  opacity: number
}

export const CALAMITY_ICONS: Record<
  'plague' | 'bad_weather' | 'famine',
  CalamityIconStyle
> = {
  plague: {
    src: '/icons/reaper-scythe.svg',
    count: 5,
    className: '',
    opacity: 0.6,
  },
  bad_weather: {
    src: '/icons/raining.svg',
    count: 4,
    className: 'invert opacity-80',
    opacity: 0.8,
  },
  famine: {
    src: '/icons/desert-skull.svg',
    count: 3,
    className: 'sepia brightness-75',
    opacity: 0.65,
  },
}

/**
 * Map artwork for the card-order overlay, from game-icons.net
 * (CC BY 3.0, icons by Delapouite):
 * https://game-icons.net/1x1/delapouite/sun.html
 * https://game-icons.net/1x1/delapouite/wheat.html
 * https://game-icons.net/1x1/delapouite/uprising.html
 */
export interface CardIconStyle {
  src: string
  className: string
  opacity: number
  /** Icon count per territory when the card scatters over a region. */
  count: number
}

export type CardIconKind = 'fair_weather' | 'abundant_harvest' | 'revolt'

export const CARD_ICONS: Record<CardIconKind, CardIconStyle> = {
  fair_weather: {
    src: '/icons/sun.svg',
    className: 'sepia saturate-200 hue-rotate-15 brightness-110',
    opacity: 0.85,
    count: 3,
  },
  abundant_harvest: {
    src: '/icons/wheat.svg',
    className: 'sepia saturate-150 hue-rotate-30',
    opacity: 0.85,
    count: 3,
  },
  revolt: {
    src: '/icons/uprising.svg',
    className: '',
    opacity: 0.95,
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

const SPECIAL_KIND_CODES: Record<string, CardIconKind> = {
  BT: 'fair_weather',
  FW: 'fair_weather',
  RA: 'abundant_harvest',
  AH: 'abundant_harvest',
  RE: 'revolt',
  RV: 'revolt',
}

export interface SpecialOrderPlacement {
  kind: CardIconKind
  /** Region seed for weather cards, territory for revolts. */
  target: string
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
