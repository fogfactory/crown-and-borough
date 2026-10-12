/**
 * Raw colors for SVG drawings and inline styles (map, markers, lineage tree,
 * brand mark). UI chrome colors are Tailwind theme tokens in `src/index.css`;
 * this module is the only production file allowed to hold hex literals
 * (enforced by ESLint).
 */
import type { Terrain } from '@/types'

// Terrain

/** Territory fill per terrain. */
export const TERRAIN_COLORS: Record<Terrain, string> = {
  plain: '#d4e8b4',
  forest: '#85b092',
  hill: '#cfb694',
  mountain: '#aab3bc',
  swamp: '#90c6c0',
}

/** Stroke of the hatching drawn over each terrain. */
export const TERRAIN_PATTERN_STROKES: Record<Terrain, string> = {
  plain: '#5a7a34',
  forest: '#14291d',
  hill: '#6b4a30',
  mountain: '#4d565e',
  swamp: '#2e5f5a',
}

// Map canvas and shared ink/paper tones

export const MAP_BACKGROUND = '#e6d8bb'
/** Dark ink used for casings, labels and glyph fills. */
export const MAP_INK = '#30291f'
/** Light parchment used for marker foregrounds and strokes. */
export const MAP_PARCHMENT = '#fff8e7'
export const MAP_PAPER = '#fffaf0'
/** Halo around labels and icon casings. */
export const MAP_HALO = '#f5ecd9'
export const MAP_BORDER_STROKE = '#594b3c'
export const MAP_BORDER_DARK = '#39271b'
export const MAP_AMBER_STROKE = '#d28b22'
export const MAP_NEUTRAL_STROKE = '#808080'
export const MAP_BROWN_FILL = '#59401f'

// Players, owners and fallbacks

export const PLAYER_RED = '#a84632'
export const PLAYER_PALETTE = [PLAYER_RED, '#2d5f9e', '#7052a1', '#0e7490', '#ad7a25']
/** Color of a player with no color set. */
export const DEFAULT_PLAYER_COLOR = '#b7a786'
/** Color of an owner missing from the player list. */
export const UNKNOWN_OWNER_COLOR = '#475569'
/** Rebel armies answer to no crown: they render in a neutral gray. */
export const NEUTRAL_ARMY_COLOR = '#6b7280'

// Regions (bishoprics)

export const HERALDIC_COLORS = [
  '#1d4ed8',
  '#b91c1c',
  '#15803d',
  '#d97706',
  '#7c3aed',
  '#374151',
] as const
/** Fill of a region with no assigned heraldic color. */
export const DEFAULT_REGION_COLOR = '#315a75'
export const REGION_CALAMITY_COLOR = '#b91c1c'
export const REGION_BLESSING_COLOR = '#15803d'

// Seasons

export const SNOW_FILL = '#eaf3ff'
export const SNOW_STROKE = '#f8fbff'
export const WINTER_ERROR_COLOR = '#c43b2a'
export const WINTER_WARNING_COLOR = '#e07a30'

// Markers

/** Outline color shared by every intention/order annotation on the map. */
export const INTENT_OUTLINE_COLOR = '#17120f'
/** Fill used for a drafted (not yet submitted) order's markers. */
export const DRAFT_INTENTION_COLOR = '#d4a39b'
export const FORTIFICATION_WALL_FILL = '#b8b8b4'
/** Neutral fill for infrastructure without a controlling player. */
export const NEUTRAL_MARKER_FILL = '#efe6d0'
export const MARKER_BRASS = '#815f1e'
export const MARKER_BROWN = '#4e3828'
export const MARKER_GLINT = '#fff3c4'
export const FAMINE_MARKER_COLOR = '#8d321e'
/** Ring around a captive noble's pawn. */
export const PRISONER_RING_COLOR = '#8d321e'
export const LEGEND_SWATCH_BACKGROUND = '#f8f0e2'

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
