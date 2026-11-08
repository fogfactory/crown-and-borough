import { useId } from 'react'

import { SELECTION_STROKE } from '@/lib/map-palette'
import { centroid } from '@/lib/map-svg-geometry'
import type { Territory } from '@/types'

/**
 * Arrows from one territory to another (an attack, a support, a retreat, a
 * supply line), drawn centre to centre with the head on the destination.
 */
export function EventArrows({
  arrows,
  territories,
  scale,
}: {
  arrows: Array<[string, string]> | undefined
  territories: Territory[]
  /** Map units per screen unit, so strokes keep their look when zoomed. */
  scale: number
}) {
  const markerId = useId()
  if (!arrows || arrows.length === 0) return null
  return (
    <g data-event-arrows>
      <defs>
        <marker
          id={markerId}
          viewBox="0 0 10 10"
          refX="9"
          refY="5"
          markerWidth="5"
          markerHeight="5"
          orient="auto-start-reverse"
        >
          <path d="M0,0 L10,5 L0,10 z" fill={SELECTION_STROKE} />
        </marker>
      </defs>
      {arrows.map(([from, to], index) => {
        const start = territories.find((territory) => territory.id === from)
        const end = territories.find((territory) => territory.id === to)
        if (!start || !end) return null
        const [x1, y1] = centroid(start.points)
        const [x2, y2] = centroid(end.points)
        return (
          <line
            key={`${from}-${to}-${index}`}
            data-event-arrow={`${from}-${to}`}
            x1={x1}
            y1={y1}
            x2={x2}
            y2={y2}
            stroke={SELECTION_STROKE}
            strokeWidth={3 * scale}
            strokeLinecap="round"
            markerEnd={`url(#${markerId})`}
          />
        )
      })}
    </g>
  )
}
