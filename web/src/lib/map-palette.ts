/**
 * Raw colors for SVG drawings and inline styles (map, markers, lineage tree,
 * brand mark). UI chrome colors are Tailwind theme tokens in `src/index.css`;
 * this module is the only production file allowed to hold hex literals
 * (enforced by ESLint).
 */
import type { Season } from '@/types'

// Terrain

/** Common paper of the engraved map, shared by every territory. */
export const MAP_PAPER_FILL = '#f1e6cb'

/** Water of a river frontier (cartographie.md). */
export const RIVER_BLUE = '#3d76a6'

// Map canvas and shared ink/paper tones

export const MAP_BACKGROUND = '#e6d8bb'
/** Dark ink used for casings, labels and glyph fills. */
export const MAP_INK = '#30291f'
/** Light parchment used for marker foregrounds and strokes. */
export const MAP_PARCHMENT = '#fff8e7'
export const MAP_PAPER = '#fffaf0'
/** Halo around labels and icon casings. */
export const MAP_HALO = MAP_PAPER_FILL
/** Selection halo (`brass-400`) and stroke (`brass-700`). */
export const SELECTION_HALO = '#c9a24a'
export const SELECTION_STROKE = '#7d5a1c'
/** Fine inner rule of the bishopric frame (`brass-300`). */
export const FRAME_RULE = '#e2c47a'
export const MAP_BROWN_FILL = '#59401f'

// Players, owners and fallbacks

/** Player tinctures in seat order (the engine's `players[].color`). */
export const PLAYER_PALETTE = [
  '#b3322b',
  '#2b5c9e',
  '#3f7d3c',
  '#6e3f8f',
  '#2f2b27',
  '#c8651b',
  '#4fa3c7',
  '#8c2f5a',
]
/** Gules, the first seat's tincture: default and sample player color. */
export const PLAYER_RED = PLAYER_PALETTE[0]
/** Color of a player with no color set. */
export const DEFAULT_PLAYER_COLOR = '#b7a786'
/** Color of an owner missing from the player list. */
export const UNKNOWN_OWNER_COLOR = '#475569'
/** Rebel armies answer to no crown: an ashen shield with no charge. */
export const NEUTRAL_ARMY_COLOR = '#7d766a'
/** Heraldic metals of the player charges. */
export const METAL_COLORS = { or: '#d8ab35', argent: '#f1ede3' } as const
/** Crowns of titled nobles and fiefs (`brass-400`). */
export const CROWN_FILL = '#c9a24a'
/** Robes of the cardinal and of the pope on a noble's pawn. */
export const CARDINAL_ROBE = '#a4231c'
export const POPE_ROBE = '#f1ede3'
/** Number on an army shield. */
export const SHIELD_NUMBER = '#ffffff'

// Regions (bishoprics)

/** Earthy bishopric tints: ochre, slate, olive, brick, lavender, sage. */
export const HERALDIC_COLORS = [
  '#a9803a',
  '#5f7d8c',
  '#7b7d47',
  '#9a5f4b',
  '#7f7496',
  '#6b8a74',
] as const
/**
 * Each tint darkened to reach 4.5:1 on the map paper, for the name of the
 * bishopric's chef-lieu.
 */
export const HERALDIC_TEXT_COLORS = [
  '#80612c',
  '#516a77',
  '#67693c',
  '#8f5846',
  '#6b617e',
  '#536c5a',
] as const
/** Fill of a region with no assigned tint: `line-strong`, outside the tints. */
export const DEFAULT_REGION_COLOR = '#8f7d5c'

// Seasons

/** Veil laid over the whole map, announcing the season. */
export const SEASON_VEILS: Record<Season, { color: string; opacity: number }> = {
  spring: { color: '#9cc36b', opacity: 0.14 },
  summer: { color: '#e3b341', opacity: 0.14 },
  autumn: { color: '#c0703a', opacity: 0.17 },
  winter: { color: '#e3edf7', opacity: 0.42 },
}
export const SNOW_STROKE = '#f8fbff'
export const WINTER_ERROR_COLOR = '#c43b2a'
export const WINTER_WARNING_COLOR = '#e07a30'

// Markers

/** Outline color shared by every intention/order annotation on the map. */
export const INTENT_OUTLINE_COLOR = MAP_INK
/** Opacity of a drafted (not yet submitted) order's arrows and badges. */
export const DRAFT_INTENTION_OPACITY = 0.45
export const FORTIFICATION_WALL_FILL = '#b8b8b4'
/** Neutral fill for infrastructure without a controlling player (`paper-50`). */
export const NEUTRAL_MARKER_FILL = '#fbf7ee'
/** Capital crown over a castle (`brass-700`). */
export const MARKER_BRASS = '#7d5a1c'
export const FAMINE_MARKER_COLOR = '#8d321e'

// Game icons

export const ICON_GOLD = '#e3b341'
export const ICON_GREEN = '#4e7d3b'

// Lineage tree

export const LINEAGE_GOLD = '#b8860b'
export const LINEAGE_SILVER = '#9aa3ad'
export const LINEAGE_CLAIM = '#7a4fa3'
export const LINEAGE_PATTERN_STROKE = '#8c8372'
export const LINEAGE_SECONDARY_LINK = '#dfe3e7'

// Brand mark

export const BRAND_BLACK = '#000000'
export const BRAND_LEAF_LIGHT = '#7ed321'
export const BRAND_LEAF_DARK = '#417505'
export const BRAND_GOLD_LIGHT = '#f8e71c'
export const BRAND_GOLD_DARK = '#f5a623'
export const BRAND_PARCHMENT = '#ebc893'
export const BRAND_PARCHMENT_EDGE = '#e0aa58'
export const BRAND_TERRACOTTA = '#c87137'
export const BRAND_CREAM = '#f6e7d0'
export const BRAND_SAND = '#f2dbb9'
export const BRAND_TAN = '#eed3a4'
export const BRAND_BRASS = '#b57204'
