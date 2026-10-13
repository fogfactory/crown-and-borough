import type { StateData, Territory } from '@/types'

/** Name: Alegreya italic 500, 12 units. Code: Cinzel 700, 8 units. */
export const NAME_FONT_SIZE = 12
export const CODE_FONT_SIZE = 8
export const CODE_LETTER_SPACING = 0.08
/** Longer names break at a hyphen onto two lines. */
const NAME_BREAK_LENGTH = 14
/** An ordinary territory shows only its code below this on-screen name size. */
export const NAME_MIN_SCREEN_PX = 10

// Average advances, as a share of the font size, to place the banner.
const NAME_ADVANCE = 0.47
const CODE_ADVANCE = 0.78 + CODE_LETTER_SPACING
const NAME_LINE_HEIGHT = 1.05

export interface TerritoryLabelLayout {
  /** Name lines, empty when only the code shows. */
  nameLines: string[]
  /** Baselines relative to the centroid, in map units before annotation scale. */
  nameBaselines: number[]
  codeBaseline: number
  /** Where a banner plants its foot: left of the line it stands by. */
  bannerAnchor: { x: number; y: number }
}

/**
 * Territories that always show their full name: a player's capital, a fief
 * capital (crown fiefs included) and a bishopric's chef-lieu.
 */
export function particularTerritoryIds(state: StateData, regionSeeds: Iterable<string>) {
  const ids = new Set<string>(regionSeeds)
  for (const player of state.players) {
    if (player.capitalTerritory) ids.add(player.capitalTerritory)
  }
  for (const fief of state.fiefs ?? []) ids.add(fief.capital)
  return ids
}

/** Splits a long name after its middle-most hyphen, keeping the hyphen. */
export function nameLines(name: string): string[] {
  if (name.length <= NAME_BREAK_LENGTH || !name.includes('-')) return [name]
  let best = -1
  for (
    let index = name.indexOf('-');
    index !== -1;
    index = name.indexOf('-', index + 1)
  ) {
    if (
      best === -1 ||
      Math.abs(index - name.length / 2) < Math.abs(best - name.length / 2)
    ) {
      best = index
    }
  }
  return [name.slice(0, best + 1), name.slice(best + 1)]
}

/**
 * Lays out one territory's label around its centroid: the name above the
 * code when shown, the code alone otherwise. The banner stands left of the
 * name (on its last baseline), or of the code when the name is hidden.
 */
export function territoryLabelLayout(
  territory: Territory,
  showName: boolean,
): TerritoryLabelLayout {
  const lines = showName ? nameLines(territory.name) : []
  const codeWidth = territory.id.length * CODE_FONT_SIZE * CODE_ADVANCE
  if (lines.length === 0) {
    const codeBaseline = CODE_FONT_SIZE * 0.35
    return {
      nameLines: [],
      nameBaselines: [],
      codeBaseline,
      bannerAnchor: { x: -codeWidth / 2, y: codeBaseline },
    }
  }
  const lineStep = NAME_FONT_SIZE * NAME_LINE_HEIGHT
  const lastBaseline = NAME_FONT_SIZE * 0.3
  const nameBaselines = lines.map(
    (_, index) => lastBaseline - (lines.length - 1 - index) * lineStep,
  )
  const nameWidth =
    Math.max(...lines.map((line) => line.length)) * NAME_FONT_SIZE * NAME_ADVANCE
  return {
    nameLines: lines,
    nameBaselines,
    codeBaseline: lastBaseline + CODE_FONT_SIZE * 1.15,
    bannerAnchor: { x: -Math.max(nameWidth, codeWidth) / 2, y: lastBaseline },
  }
}

/** On-screen size of a territory name, in pixels. */
export function nameScreenSize(annotationScale: number, pixelsPerUnit: number): number {
  return NAME_FONT_SIZE * annotationScale * pixelsPerUnit
}
