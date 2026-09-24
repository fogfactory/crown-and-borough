import { useMemo } from 'react'

import type { RegionStyle } from '@/lib/region-color'
import { computeRegionOutlines, type RegionOutlineResult } from '@/lib/region-geometry'
import {
  REGION_BAND_MARGIN,
  computeRegionBands,
  interiorRegionIds,
  regionHoleTesters as computeRegionHoleTesters,
  regionIdByTerritory,
  regionStylesById,
  territoryLocator,
  type RegionBand,
} from '@/lib/region-layout'
import type { MapData, Point, Region } from '@/types'

export interface RegionLayout {
  /** Regions are toggled on and the map defines at least one. */
  regionsActive: boolean
  regionStyleByID: Map<string, RegionStyle>
  regionsById: Map<string, Region>
  regionBySeed: Map<string, Region>
  /** Region outlines, only computed while regions are active. */
  regionOutlines: RegionOutlineResult | null
  interiorRegionIDs: Set<string>
  regionBands: RegionBand[]
  regionHoleTesters: Map<string, (loop: Point[]) => boolean>
  /** Frame padding around the map reserved for the region bands. */
  bandMargin: number
}

/**
 * Region lookups, outlines and the name frame around the map. The outline
 * chain (outlines → bands, hole testers, interior regions) only runs while
 * regions are shown.
 */
export function useRegionLayout(
  map: MapData,
  showRegions: boolean,
  mapWidth: number,
  mapHeight: number,
): RegionLayout {
  const regionByTerritory = useMemo(() => regionIdByTerritory(map.regions), [map.regions])
  const regionStyleByID = regionStylesById(regionByTerritory)

  const regionsActive = showRegions && (map.regions?.length ?? 0) > 0
  const regionsById = useMemo(
    () => new Map((map.regions ?? []).map((region) => [region.id, region])),
    [map.regions],
  )
  const regionBySeed = useMemo(
    () => new Map((map.regions ?? []).map((region) => [region.seed, region])),
    [map.regions],
  )

  const regionOutlines = useMemo(
    () =>
      regionsActive ? computeRegionOutlines(map.territories, map.regions ?? []) : null,
    [regionsActive, map.territories, map.regions],
  )
  const interiorRegionIDs = useMemo(
    () => interiorRegionIds(regionOutlines),
    [regionOutlines],
  )
  const territoryAt = useMemo(() => territoryLocator(map.territories), [map.territories])
  const regionBands = useMemo(
    () => computeRegionBands(regionOutlines, mapWidth, mapHeight),
    [regionOutlines, mapWidth, mapHeight],
  )
  const regionHoleTesters = useMemo(
    () => computeRegionHoleTesters(regionOutlines, territoryAt, regionByTerritory),
    [regionOutlines, territoryAt, regionByTerritory],
  )

  return {
    regionsActive,
    regionStyleByID,
    regionsById,
    regionBySeed,
    regionOutlines,
    interiorRegionIDs,
    regionBands,
    regionHoleTesters,
    bandMargin: regionsActive && regionBands.length > 0 ? REGION_BAND_MARGIN : 0,
  }
}
