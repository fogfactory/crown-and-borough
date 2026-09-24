import { chaoticIconPlacements, type IconPlacement } from '@/lib/chaotic-icons'
import type { GameIconGlyph } from '@/lib/game-icon-glyphs'
import {
  CALAMITY_ICONS,
  CANCELED_KIND_BY_CARD,
  CARD_ICONS,
  parseSpecialOrderPlacements,
} from '@/lib/game-icons'
import type { ActiveRegionEffect, PlayerId, Region, StateData, Territory } from '@/types'

export interface SpecialOrderDraft {
  player: PlayerId
  text: string
}

export interface ScatteredIcon {
  key: string
  glyph: GameIconGlyph
  fill: string
  stroke?: string
  strokeWidth?: number
  opacity: number
  placement: IconPlacement
}

export interface CalamityIcon extends ScatteredIcon {
  /** A single drafted card counters this calamity: draw a circle-slash badge. */
  canceled: boolean
}

/**
 * Drafted canceling cards per (canceled calamity, region): BT against bad
 * weather, RA against famine. The first card cancels the calamity; per the
 * rules a second card of the same kind applies its regional bonus instead.
 */
export function countCancelingCards(
  specialOrders: SpecialOrderDraft[],
): Map<string, number> {
  const counts = new Map<string, number>()
  for (const { text } of specialOrders) {
    for (const placement of parseSpecialOrderPlacements(text)) {
      const canceledKind =
        placement.kind === 'fair_weather' || placement.kind === 'abundant_harvest'
          ? CANCELED_KIND_BY_CARD[placement.kind]
          : null
      if (canceledKind) {
        const key = `${canceledKind}-${placement.target}`
        counts.set(key, (counts.get(key) ?? 0) + 1)
      }
    }
  }
  return counts
}

/**
 * Regions where the drafted cards fully clear the calamity (two canceling
 * cards or more): the calamity icons disappear and the residual bonus
 * scatters instead. With a single canceling card the calamity icons stay,
 * marked with a cancellation badge.
 */
export function fullyCanceledCalamities(
  canceledCardCounts: Map<string, number>,
  activeRegionEffects: ActiveRegionEffect[] | undefined,
): Set<string> {
  const canceled = new Set<string>()
  for (const [key, count] of canceledCardCounts) {
    if (
      count >= 2 &&
      (activeRegionEffects ?? []).some(
        (effect) => `${effect.kind}-${effect.regionSeed}` === key,
      )
    ) {
      canceled.add(key)
    }
  }
  return canceled
}

/**
 * Regions where a single canceling card counters an active calamity: the
 * calamity icons stay until resolution, each marked with a circle-slash
 * badge.
 */
export function singleCanceledRegionSeeds(
  canceledCardCounts: Map<string, number>,
  activeRegionEffects: ActiveRegionEffect[] | undefined,
): Set<string> {
  const regions = new Set<string>()
  for (const [key, count] of canceledCardCounts) {
    if (
      count === 1 &&
      (activeRegionEffects ?? []).some(
        (effect) => `${effect.kind}-${effect.regionSeed}` === key,
      )
    ) {
      regions.add(key.split('-')[1] ?? '')
    }
  }
  return regions
}

/** Calamity glyphs scattered over every territory of each afflicted region. */
export function calamityIconItems(
  activeRegionEffects: ActiveRegionEffect[] | undefined,
  regions: Region[] | undefined,
  territories: Territory[],
  canceledCalamityRegions: Set<string>,
  singleCanceledRegions: Set<string>,
): CalamityIcon[] {
  const regionsBySeed = new Map((regions ?? []).map((region) => [region.seed, region]))
  const territoriesById = new Map(
    territories.map((territory) => [territory.id, territory]),
  )
  const items: CalamityIcon[] = []
  for (const effect of activeRegionEffects ?? []) {
    if (canceledCalamityRegions.has(`${effect.kind}-${effect.regionSeed}`)) {
      continue
    }
    const style = CALAMITY_ICONS[effect.kind as keyof typeof CALAMITY_ICONS]
    if (!style) {
      continue
    }
    const region = regionsBySeed.get(effect.regionSeed)
    if (!region) {
      continue
    }
    for (const territoryID of region.territories) {
      const territory = territoriesById.get(territoryID)
      if (!territory) {
        continue
      }
      const seedKey = `${effect.kind}-${effect.regionSeed}-${territory.id}`
      for (const placement of chaoticIconPlacements(
        territory.points,
        style.count,
        seedKey,
      )) {
        items.push({
          key: `${seedKey}-${placement.x.toFixed(1)}-${placement.y.toFixed(1)}`,
          glyph: style.glyph,
          fill: style.fill,
          stroke: style.stroke,
          strokeWidth: style.strokeWidth,
          opacity: style.opacity,
          canceled: singleCanceledRegions.has(effect.regionSeed),
          placement,
        })
      }
    }
  }
  return items
}

/**
 * Drafted deck cards scattered over their target: revolts over the target
 * territory in the drafting player's color, regional bonuses over every
 * territory of the region (once per card kind and region).
 */
export function cardIconItems(
  specialOrders: SpecialOrderDraft[],
  regions: Region[] | undefined,
  territories: Territory[],
  activeRegionEffects: ActiveRegionEffect[] | undefined,
  players: StateData['players'],
  canceledCalamityRegions: Set<string>,
): ScatteredIcon[] {
  const regionsBySeed = new Map((regions ?? []).map((region) => [region.seed, region]))
  const territoriesById = new Map(
    territories.map((territory) => [territory.id, territory]),
  )
  const activeByRegion = new Map(
    (activeRegionEffects ?? []).map((effect) => [
      `${effect.kind}-${effect.regionSeed}`,
      true,
    ]),
  )
  const colorsByPlayer = new Map(players.map((player) => [player.id, player.color]))
  const scatterItems: ScatteredIcon[] = []
  const scatteredRegions = new Set<string>()
  for (const { player, text } of specialOrders) {
    for (const placement of parseSpecialOrderPlacements(text)) {
      if (placement.kind === 'revolt') {
        const territory = territoriesById.get(placement.target)
        if (!territory) {
          continue
        }
        const style = CARD_ICONS.revolt
        const seedKey = `revolt-${placement.target}`
        for (const placement2 of chaoticIconPlacements(
          territory.points,
          style.count,
          seedKey,
        )) {
          scatterItems.push({
            key: `${seedKey}-${placement2.x.toFixed(1)}-${placement2.y.toFixed(1)}`,
            glyph: style.glyph,
            fill: colorsByPlayer.get(player) ?? '#475569',
            stroke: style.stroke,
            strokeWidth: style.strokeWidth,
            opacity: style.opacity,
            placement: placement2,
          })
        }
        continue
      }
      // One canceling card against an active calamity only marks the
      // calamity with a cancellation badge; two or more cards clear the
      // calamity and the residual bonus scatters instead.
      const canceledKind = CANCELED_KIND_BY_CARD[placement.kind]
      if (
        activeByRegion.has(`${canceledKind}-${placement.target}`) &&
        !canceledCalamityRegions.has(`${canceledKind}-${placement.target}`)
      ) {
        continue
      }
      const region = regionsBySeed.get(placement.target)
      if (!region || scatteredRegions.has(`${placement.kind}-${placement.target}`)) {
        continue
      }
      scatteredRegions.add(`${placement.kind}-${placement.target}`)
      const style = CARD_ICONS[placement.kind]
      for (const territoryID of region.territories) {
        const territory = territoriesById.get(territoryID)
        if (!territory) {
          continue
        }
        const seedKey = `${placement.kind}-${placement.target}-${territory.id}`
        for (const placement2 of chaoticIconPlacements(
          territory.points,
          style.count,
          seedKey,
        )) {
          scatterItems.push({
            key: `${seedKey}-${placement2.x.toFixed(1)}-${placement2.y.toFixed(1)}`,
            glyph: style.glyph,
            fill: style.fill,
            stroke: style.stroke,
            strokeWidth: style.strokeWidth,
            opacity: style.opacity,
            placement: placement2,
          })
        }
      }
    }
  }
  return scatterItems
}
