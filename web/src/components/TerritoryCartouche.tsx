import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'

import type { ScreenBox } from '@/components/MapViewer'
import { MAP_INK } from '@/lib/map-palette'

/** Widest cartouche, and its margin to the map's edges and to the territory. */
const WIDTH = 280
const MARGIN = 8
const GAP = 24

interface TerritoryCartoucheProps {
  /** The selected territory's box on screen. */
  anchor: ScreenBox
  /** The map's size on screen. */
  size: { width: number; height: number }
  onClose: () => void
  children: ReactNode
}

/**
 * The territory sheet as a cartouche laid on the map (charte-graphique.md
 * § Fiche de territoire): 280 px wide at most, on the side of the territory
 * away from the drawer (the left, the drawer being on the right) when there
 * is room, never over the territory itself when it can avoid it, and tied to
 * it by an ink line. Escape closes it.
 */
export function TerritoryCartouche({
  anchor,
  size,
  onClose,
  children,
}: TerritoryCartoucheProps) {
  const boxRef = useRef<HTMLDivElement>(null)
  const [height, setHeight] = useState(0)
  const close = useRef(onClose)
  close.current = onClose

  useLayoutEffect(() => {
    const box = boxRef.current
    if (!box) return
    const measure = () => setHeight(box.offsetHeight)
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(box)
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      // An open popover (a noble's detail, the scores) takes the Escape.
      if (document.querySelector('[data-radix-popper-content-wrapper]')) return
      close.current()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  const width = Math.min(WIDTH, size.width - MARGIN * 2)
  const roomLeft = anchor.left - GAP - MARGIN
  const roomRight = size.width - anchor.right - GAP - MARGIN
  const onLeft = roomLeft >= width || roomLeft >= roomRight
  const left = Math.max(
    MARGIN,
    Math.min(
      onLeft ? anchor.left - GAP - width : anchor.right + GAP,
      size.width - MARGIN - width,
    ),
  )
  const centerY = (anchor.top + anchor.bottom) / 2
  const maxHeight = size.height - MARGIN * 2
  const top = Math.max(
    MARGIN,
    Math.min(
      centerY - Math.min(height, maxHeight) / 2,
      size.height - MARGIN - Math.min(height, maxHeight),
    ),
  )

  // The ink line runs from the cartouche's near edge to the territory centre.
  const lineFromX = onLeft ? left + width : left
  const lineFromY = Math.max(top + 12, Math.min(centerY, top + Math.max(height, 24) - 12))
  const lineToX = (anchor.left + anchor.right) / 2
  // Panned out of view, the territory gets no line toward the map's edge.
  const anchorVisible =
    anchor.right > 0 &&
    anchor.left < size.width &&
    anchor.bottom > 0 &&
    anchor.top < size.height

  return (
    <>
      {anchorVisible && (
        <svg
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 z-20 h-full w-full"
          data-cartouche-line
        >
          <line
            x1={lineFromX}
            y1={lineFromY}
            x2={lineToX}
            y2={centerY}
            stroke={MAP_INK}
            strokeWidth={1.5}
          />
          <circle cx={lineToX} cy={centerY} r={3} fill={MAP_INK} />
        </svg>
      )}
      <div
        ref={boxRef}
        data-map-cartouche
        data-side={onLeft ? 'left' : 'right'}
        className="absolute z-30 overflow-y-auto rounded-md border border-line-strong bg-paper-50 p-3 shadow-panel"
        style={{ left, top, width, maxHeight }}
      >
        {children}
      </div>
    </>
  )
}
