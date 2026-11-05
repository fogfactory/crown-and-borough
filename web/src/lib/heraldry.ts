import { METAL_COLORS, NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'
import {
  NEUTRAL_PLAYER_ID,
  type Charge,
  type FiefTitle,
  type Player,
  type StateData,
} from '@/types'

/** Arms ready to draw: the field color, and the charge in its metal if any. */
export interface ResolvedArms {
  field: string
  charge: { kind: Charge; metal: string } | null
}

/**
 * Arms of every owner on the map. A player's field is its `color`; rebels
 * carry a plain ashen shield, and a player from a game created before arms
 * existed a plain field of its color.
 */
export function resolveArmsByOwner(
  state: StateData,
  playerColors: Map<string, string>,
): Map<string, ResolvedArms> {
  const armsByPlayer = new Map(state.players.map((player) => [player.id, player.arms]))
  const resolved = new Map<string, ResolvedArms>()
  for (const [owner, color] of playerColors) {
    resolved.set(
      owner,
      owner === NEUTRAL_PLAYER_ID
        ? { field: NEUTRAL_ARMY_COLOR, charge: null }
        : playerArms({ color, arms: armsByPlayer.get(owner) }),
    )
  }
  // A vacant fief carries a neutral shield even when no rebel is on the map.
  if (!resolved.has(NEUTRAL_PLAYER_ID)) {
    resolved.set(NEUTRAL_PLAYER_ID, { field: NEUTRAL_ARMY_COLOR, charge: null })
  }
  return resolved
}

/** A player's arms: its colour as the field, and its charge if it has one. */
export function playerArms(player: Pick<Player, 'color' | 'arms'>): ResolvedArms {
  return {
    field: player.color,
    charge: player.arms
      ? { kind: player.arms.charge, metal: METAL_COLORS[player.arms.metal] }
      : null,
  }
}

/** A rank crown, or the king's closed crown. */
export type CrownKind = FiefTitle | 'king'

// Heater shield, 15 × 19 units centred on the origin.
export const SHIELD_WIDTH = 15
export const SHIELD_HEIGHT = 19
export const SHIELD_BOX: [number, number, number, number] = [
  -SHIELD_WIDTH / 2,
  -SHIELD_HEIGHT / 2,
  SHIELD_WIDTH,
  SHIELD_HEIGHT,
]
export const SHIELD_PATH =
  'M-7.5 -9.5 H7.5 V-0.5 C7.5 5 3.8 8.3 0 9.5 C-3.8 8.3 -7.5 5 -7.5 -0.5 Z'

// Swallowtail flag, 12 × 8 units, hoisted at the origin.
export const BANNER_WIDTH = 12
export const BANNER_HEIGHT = 8
export const BANNER_PATH = 'M0 0 H12 L9.2 4 L12 8 H0 Z'

/**
 * The charge laid over a field filling the box (x, y, width, height), to be
 * clipped by the field's outline. Bands are drawn as strokes so one width
 * rule fits both the shield and the banner.
 */
export function chargeShape(
  kind: Charge,
  x: number,
  y: number,
  width: number,
  height: number,
): { d: string; stroke?: number } {
  const band = Math.min(width, height) * 0.3
  const right = x + width
  const bottom = y + height
  const midX = x + width / 2
  const midY = y + height / 2
  switch (kind) {
    case 'bend':
      return { d: `M${x} ${y} L${right} ${bottom}`, stroke: band }
    case 'fess':
      return { d: `M${x} ${midY} L${right} ${midY}`, stroke: band }
    case 'pale':
      return { d: `M${midX} ${y} L${midX} ${bottom}`, stroke: band }
    case 'cross':
      return {
        d: `M${midX} ${y} L${midX} ${bottom} M${x} ${y + height * 0.42} L${right} ${y + height * 0.42}`,
        stroke: band * 0.85,
      }
    case 'saltire':
      return {
        d: `M${x} ${y} L${right} ${bottom} M${right} ${y} L${x} ${bottom}`,
        stroke: band * 0.8,
      }
    case 'chevron':
      return {
        d: `M${x} ${y + height * 0.9} L${midX} ${y + height * 0.38} L${right} ${y + height * 0.9}`,
        stroke: band * 0.9,
      }
    case 'chief':
      return {
        d: `M${x} ${y} H${right} V${y + height * 0.32} H${x} Z`,
      }
    case 'bordure':
      // Stroked along the outline by the caller: the clip keeps the inner half.
      return { d: '', stroke: band * 0.9 }
  }
}

/**
 * Crown drawings, 10 × 6 units, centred on x = 0 and resting on y = 0: a
 * circlet with pearls (baron, count), fleurons and pearls (marquis),
 * fleurons (duke), or closed arches and a cross (king).
 */
export function crownShape(kind: CrownKind): {
  solid: string
  pearls: Array<[number, number, number]>
} {
  const circlet = 'M-5 0 V-2.2 H5 V0 Z'
  const fleuron = (x: number, height: number) =>
    `M${x - 1.1} -2.2 L${x - 0.9} ${-2.2 - height * 0.55} Q${x} ${-2.2 - height * 1.1} ${x + 0.9} ${-2.2 - height * 0.55} L${x + 1.1} -2.2 Z`
  switch (kind) {
    case 'barony':
      return {
        solid: circlet,
        pearls: [
          [-3.6, -3, 1],
          [0, -3, 1],
          [3.6, -3, 1],
        ],
      }
    case 'county':
      return {
        solid: circlet,
        pearls: Array.from({ length: 9 }, (_, index): [number, number, number] => [
          -4.4 + index * 1.1,
          -2.85,
          0.55,
        ]),
      }
    case 'marquisate':
      return {
        solid: `${circlet} ${fleuron(-3.4, 3.4)} ${fleuron(0, 3.8)} ${fleuron(3.4, 3.4)}`,
        pearls: [
          [-1.7, -2.9, 0.7],
          [1.7, -2.9, 0.7],
        ],
      }
    case 'duchy':
      return {
        solid: `${circlet} ${fleuron(-3.4, 3.6)} ${fleuron(0, 4)} ${fleuron(3.4, 3.6)}`,
        pearls: [],
      }
    case 'king':
      return {
        solid: `${circlet} M-4.6 -2.2 Q-4.4 -5.4 0 -5.6 Q4.4 -5.4 4.6 -2.2 L3.5 -2.2 Q3.3 -4.5 0 -4.6 Q-3.3 -4.5 -3.5 -2.2 Z M-0.4 -5.4 V-7.9 H0.4 V-5.4 Z M-1.2 -7.3 H1.2 V-6.5 H-1.2 Z`,
        pearls: [[0, -5.7, 0.7]],
      }
  }
}

// Control shield: the heater shield, reduced to sit beside a territory name.
export const CONTROL_SHIELD_SCALE = 0.8

/** A color's grey of equal luminance (Rec. 709 weights). */
export function desaturate(hex: string): string {
  const match = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex.trim())
  if (!match) return NEUTRAL_ARMY_COLOR
  const value =
    match[1].length === 3
      ? [...match[1]].map((digit) => digit + digit).join('')
      : match[1]
  const channel = (index: number) => parseInt(value.slice(index * 2, index * 2 + 2), 16)
  const grey = Math.round(0.2126 * channel(0) + 0.7152 * channel(1) + 0.0722 * channel(2))
  const digits = grey.toString(16).padStart(2, '0')
  return `#${digits}${digits}${digits}`
}

/** The arms of a territory held under occupation: same charge, no color. */
export function greyArms(arms: ResolvedArms): ResolvedArms {
  return {
    field: desaturate(arms.field),
    charge: arms.charge
      ? { kind: arms.charge.kind, metal: desaturate(arms.charge.metal) }
      : null,
  }
}

// Soldier figurines, in units: a pack of at most six, drawn from the front.
export const MAX_FIGURES = 6
/** Spear of a figurine, left of its body (it is held in the right hand). */
export const SPEAR_X = -3.65
export const SPEAR_TOP = -12.2
/** Stroke of a spear, and of the banner pole that continues it. */
export const SPEAR_WIDTH = 0.55
export const FIGURE_HEAD_TOP = -10.95
export const FIGURE_COLUMN = 4
export const FIGURE_BACK_ROW = -2.4
export const FIGURE_FRONT_ROW = 1.5

export interface FigurePosition {
  x: number
  y: number
}

export interface ArmyFigures {
  /** Up to six soldiers, in no particular order. */
  soldiers: FigurePosition[]
  /** Grey shadows behind a big army. */
  shadows: FigurePosition[]
  /** Index in `soldiers` of the leftmost one, who carries the banner. */
  bearer: number
}

/** A stable 32-bit hash of a string, to seed a pack's disorder. */
export function hashSeed(text: string): number {
  let hash = 2166136261
  for (let index = 0; index < text.length; index++) {
    hash = Math.imul(hash ^ text.charCodeAt(index), 16777619)
  }
  return hash >>> 0
}

function random(seed: number): () => number {
  let state = seed
  return () => {
    state = (Math.imul(state, 1664525) + 1013904223) >>> 0
    return state / 4294967296
  }
}

/** Number of grey shadows behind the six figurines of an army of `size`. */
export function shadowCount(size: number): number {
  if (size <= MAX_FIGURES) return 0
  if (size < 10) return 2
  if (size < 20) return 3
  return 4
}

/**
 * Lays an army out as a pack: one upright figurine per troop up to six,
 * shifted irregularly and overlapping, centred on x = 0 with its feet on
 * y = 0; shadows stand behind when the army is bigger. The disorder is a
 * function of `seed`, so a pack does not move between renders.
 */
export function armyFigures(size: number, seed: number): ArmyFigures {
  const key = `${seed}:${Math.min(size, MAX_FIGURES)}:${shadowCount(size)}`
  const cached = packCache.get(key)
  if (cached) return cached
  const pack = layoutArmy(size, seed)
  if (packCache.size >= PACK_CACHE_LIMIT) packCache.clear()
  packCache.set(key, pack)
  return pack
}

const PACK_CACHE_LIMIT = 512
const packCache = new Map<string, ArmyFigures>()

function layoutArmy(size: number, seed: number): ArmyFigures {
  const next = random(seed)
  const count = Math.max(1, Math.min(size, MAX_FIGURES))
  const columns = count <= 3 ? count : 3
  const twoRows = count > 3
  const soldiers: FigurePosition[] = []
  for (let index = 0; index < count; index++) {
    const column = index % columns
    const row = Math.floor(index / columns)
    const x = (column - (columns - 1) / 2) * FIGURE_COLUMN + (row ? 1.4 : 0)
    const y = twoRows ? (row ? FIGURE_FRONT_ROW : FIGURE_BACK_ROW) : 0
    soldiers.push({ x: x + (next() - 0.5) * 3.3, y: y + (next() - 0.5) * 2.4 })
  }
  const shadows: FigurePosition[] = []
  const extra = shadowCount(size)
  for (let index = 0; index < extra; index++) {
    shadows.push({
      x: (index - (extra - 1) / 2) * 5.3 + (next() - 0.5) * 2.3,
      y: -5.3 - next() * 2.3,
    })
  }
  let bearer = 0
  soldiers.forEach((soldier, index) => {
    if (soldier.x < soldiers[bearer].x) bearer = index
  })
  return { soldiers, shadows, bearer }
}

/** Owner whose soldiers draw an army: rebels when its owner has no arms. */
export function armyDrawingOwner(
  armsByOwner: Map<string, ResolvedArms>,
  armyOwner: string,
): string {
  return armsByOwner.has(armyOwner) ? armyOwner : NEUTRAL_PLAYER_ID
}
