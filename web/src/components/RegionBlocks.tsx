import type { ReactNode } from 'react'

import { RegionRef } from '@/components/MapLinks'
import type { MapData } from '@/types'

/** Seed of the region holding the territory, or '' when the map has none. */
export function regionSeedOf(map: MapData | null, territory: string): string {
  return (
    map?.regions?.find((region) => region.territories.includes(territory))?.seed ?? ''
  )
}

/** The frame of a block, its left rule coloured by the caller. */
export const BLOCK_FRAME =
  'overflow-hidden rounded-md border border-line-soft border-l-4 bg-paper-100/70'
/** The tinted band heading a framed block. */
export const BLOCK_HEADER =
  'flex items-center justify-between gap-3 bg-brass-300/25 px-2 py-1 font-semibold'

interface RegionBlocksProps<T> {
  items: T[]
  map: MapData | null
  regionOf: (item: T) => string
  territoryOf: (item: T) => string
  keyOf: (item: T) => string
  renderItem: (item: T) => ReactNode
  /** Total shown on a region's header. */
  subtotal: (items: T[]) => ReactNode
}

/**
 * Report lines set in one framed block per region (map order), territories
 * inside, so a line visibly belongs to its region. Lines with no region are
 * listed bare.
 */
export function RegionBlocks<T>({
  items,
  map,
  regionOf,
  territoryOf,
  keyOf,
  renderItem,
  subtotal,
}: RegionBlocksProps<T>) {
  const order = (map?.regions ?? []).map((region) => region.seed)
  const rank = (seed: string) => {
    const index = order.indexOf(seed)
    return index < 0 ? order.length : index
  }
  const regions = Array.from(new Set(items.map(regionOf))).sort(
    (a, b) => rank(a) - rank(b) || a.localeCompare(b),
  )
  const name = (id: string) => map?.territories.find((c) => c.id === id)?.name ?? id

  return (
    <ul className="space-y-2">
      {regions.map((region) => {
        const own = items
          .filter((item) => regionOf(item) === region)
          .sort((a, b) => territoryOf(a).localeCompare(territoryOf(b)))
        if (region === '') {
          return (
            <li key="none">
              <ul className="space-y-1.5">
                {own.map((item) => (
                  <li key={keyOf(item)}>{renderItem(item)}</li>
                ))}
              </ul>
            </li>
          )
        }
        return (
          <li
            key={region}
            data-report-region={region}
            className={`${BLOCK_FRAME} border-l-brass-500`}
          >
            <div
              className={`${BLOCK_HEADER} text-2xs uppercase tracking-wide text-brass-700`}
            >
              <RegionRef id={region} name={name(region)} />
              <span className="shrink-0 normal-case tracking-normal">
                {subtotal(own)}
              </span>
            </div>
            <ul className="divide-y divide-line-soft px-2 [&>li]:py-1.5">
              {own.map((item) => (
                <li key={keyOf(item)}>{renderItem(item)}</li>
              ))}
            </ul>
          </li>
        )
      })}
    </ul>
  )
}
