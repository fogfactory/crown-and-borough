/**
 * Raw colors for SVG drawings and inline styles (map, markers, lineage tree,
 * seal). UI chrome colors are Tailwind theme tokens in `src/index.css`;
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
/** Number on an army banner. */
export const SHIELD_NUMBER = '#ffffff'
/** Soldier figurines: steel helmet, face and the grey shadows of a big army. */
export const SOLDIER_STEEL = '#8d8f93'
export const SOLDIER_FACE = '#e8d4b0'
export const SOLDIER_SHADOW = '#4a443a'

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

/** Brown outline of the head marriage bond, filled with brass gold (`ink-700` warmed). */
export const LINEAGE_HEAD_BROWN = '#4a3417'
/**
 * Ivy of the head marriage: a dark olive, yellower and darker than the green
 * player tincture (`#3f7d3c`), so that it never reads as a player colour.
 */
export const LINEAGE_IVY = '#4a5526'
/** Secondary marriage, claim stems and pending selection (`ink-700`). */
export const LINEAGE_SECONDARY = '#594b3c'
/** Ended marriage hatching (`ink-500`). */
export const LINEAGE_ENDED = '#74644d'
/** Parchment of the cartouches and the legend (`paper-50`). */
export const LINEAGE_PAPER = '#fbf7ee'
/** Cloth of the tapestry the tree is woven on (`paper-100`). */
export const LINEAGE_CLOTH = '#f3ead9'
/** Threads of the weave, barely darker than the cloth (`line-soft`). */
export const LINEAGE_WEAVE = '#d9cfba'
/** Woven band of the tapestry border and rolled ends of a noble's scroll (`paper-300`). */
export const LINEAGE_BAND = '#e6d8bb'
/** Outer rule of the border and outline of the cartouches (`ink-900`). */
export const LINEAGE_INK = MAP_INK
/** Head marriage, inner rule and corner ornaments (`brass-700` / `brass-400`). */
export const LINEAGE_BRASS = '#7d5a1c'
export const LINEAGE_BRASS_LIGHT = '#c9a24a'
/** Digit of a rank seal, outlined in ink on the house colour (near-white). */
export const LINEAGE_RANK_DIGIT = MAP_PAPER
/** Shaded face and inner curl of a roll (`paper-200`). */
export const LINEAGE_SCROLL_SHADE = '#efe7d8'

// Seal (emblem). The wax red is not an interface color.

export const SEAL_WAX_LIGHT = '#c8433a'
export const SEAL_WAX_DARK = '#6d1210'
export const SEAL_WAX_RIM = '#8f2420'
export const SEAL_WAX_SHADOW = '#4a0b0a'
export const SEAL_LETTER = '#f6e7d0'
