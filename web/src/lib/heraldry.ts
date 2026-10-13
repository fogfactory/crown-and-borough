import { METAL_COLORS, NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'
import { NEUTRAL_PLAYER_ID, type Charge, type FiefTitle, type StateData } from '@/types'

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
    const arms = owner === NEUTRAL_PLAYER_ID ? undefined : armsByPlayer.get(owner)
    resolved.set(owner, {
      field: owner === NEUTRAL_PLAYER_ID ? NEUTRAL_ARMY_COLOR : color,
      charge: arms ? { kind: arms.charge, metal: METAL_COLORS[arms.metal] } : null,
    })
  }
  // A vacant fief flies a neutral banner even when no rebel is on the map.
  if (!resolved.has(NEUTRAL_PLAYER_ID)) {
    resolved.set(NEUTRAL_PLAYER_ID, { field: NEUTRAL_ARMY_COLOR, charge: null })
  }
  return resolved
}

/** A rank crown, or the king's closed crown. */
export type CrownKind = FiefTitle | 'king'

// Heater shield, 15 × 19 units centred on the origin.
export const SHIELD_WIDTH = 15
export const SHIELD_HEIGHT = 19
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
