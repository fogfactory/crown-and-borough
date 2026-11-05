import { useMemo } from 'react'

import {
  calamityIconItems,
  cardIconItems,
  countCancelingCards,
  fullyCanceledCalamities,
  singleCanceledRegionSeeds,
  type CalamityIcon,
  type ScatteredIcon,
  type SpecialOrderDraft,
} from '@/lib/region-effect-icons'
import type { MapData, StateData } from '@/types'

export interface RegionEffectIconsOptions {
  map: MapData
  state: StateData
  specialOrders: SpecialOrderDraft[]
  showCalamities: boolean
  showCards: boolean
}

export interface RegionEffectIcons {
  calamityIcons: CalamityIcon[]
  cardIcons: ScatteredIcon[]
}

/**
 * Glyphs scattered over the map for active calamities and drafted deck
 * cards. Drafted canceling cards feed both: one card badges the calamity,
 * two clear it and scatter the residual bonus instead.
 */
export function useRegionEffectIcons({
  map,
  state,
  specialOrders,
  showCalamities,
  showCards,
}: RegionEffectIconsOptions): RegionEffectIcons {
  const canceledCardCounts = useMemo(
    () => countCancelingCards(specialOrders),
    [specialOrders],
  )
  const canceledCalamityRegions = useMemo(
    () => fullyCanceledCalamities(canceledCardCounts, state.activeRegionEffects),
    [canceledCardCounts, state.activeRegionEffects],
  )
  const singleCanceledRegions = useMemo(
    () => singleCanceledRegionSeeds(canceledCardCounts, state.activeRegionEffects),
    [canceledCardCounts, state.activeRegionEffects],
  )

  const calamityIcons = useMemo(
    () =>
      showCalamities
        ? calamityIconItems(
            state.activeRegionEffects,
            map.regions,
            map.territories,
            canceledCalamityRegions,
            singleCanceledRegions,
          )
        : [],
    [
      showCalamities,
      map.regions,
      map.territories,
      state.activeRegionEffects,
      canceledCalamityRegions,
      singleCanceledRegions,
    ],
  )

  const cardIcons = useMemo(
    () =>
      showCards
        ? cardIconItems(
            specialOrders,
            map.regions,
            map.territories,
            state.activeRegionEffects,
            state.players,
            canceledCalamityRegions,
          )
        : [],
    [
      showCards,
      specialOrders,
      map.regions,
      map.territories,
      state.activeRegionEffects,
      state.players,
      canceledCalamityRegions,
    ],
  )

  return { calamityIcons, cardIcons }
}
