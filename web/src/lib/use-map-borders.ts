import { useMemo } from 'react'

import {
  computeMapBorders,
  impassableBorderIcons as computeImpassableBorderIcons,
  mapAnnotationScale,
  type ImpassableBorderIcon,
  type MapBorders,
} from '@/lib/map-borders'
import type { MapData } from '@/types'

export interface MapBordersLayout extends MapBorders {
  /** Size multiplier for markers and labels, relative to the reference map. */
  annotationScale: number
  impassableBorderIcons: ImpassableBorderIcon[]
}

/**
 * Map extent, border segments and the mountain chains drawn along the
 * impassable ones. Borders only recompute when the map changes; the chains
 * also follow the annotation scale.
 */
export function useMapBorders(map: MapData): MapBordersLayout {
  const borders = useMemo(() => computeMapBorders(map), [map])
  const annotationScale = mapAnnotationScale(map.territories)
  const { sharedBorders } = borders
  const impassableBorderIcons = useMemo(
    () => computeImpassableBorderIcons(sharedBorders, annotationScale),
    [sharedBorders, annotationScale],
  )
  return { ...borders, annotationScale, impassableBorderIcons }
}
