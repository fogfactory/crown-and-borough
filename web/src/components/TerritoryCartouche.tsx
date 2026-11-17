import {
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import {
  IconChevronDown,
  IconChevronUp,
  IconGripHorizontal,
  IconX,
} from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import { clamp, DRAG_THRESHOLD } from '@/lib/map-gestures'
import type { ScreenBox } from '@/components/MapViewer'
import { MAP_INK } from '@/lib/map-palette'

/** Widest cartouche, and its margin to the map's edges and to the territory. */
const WIDTH = 280
const MARGIN = 8
const GAP = 24
/** Width of the collapsed pill. */
const PILL_WIDTH = 200

/**
 * A sheet with an arrow entering it high on its left edge, drawn after Tabler's
 * file-import (MIT) with the arrow raised: the territory goes onto the order
 * sheet.
 */
function FileArrowInIcon({ className }: { className?: string }) {
  return (
    <svg
      aria-hidden="true"
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M14 3v4a1 1 0 0 0 1 1h4" />
      <path d="M5 10v-5a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-2" />
      <path d="M3 13.5h7m-3 -3l3 3l-3 3" />
    </svg>
  )
}

/** An icon button of the grip bar; its label names it for sight and for assistive tech. */
function BarButton({
  label,
  onClick,
  children,
  ...aria
}: {
  label: string
  onClick: () => void
  children: ReactNode
} & Pick<ButtonHTMLAttributes<HTMLButtonElement>, 'aria-expanded' | 'aria-controls'>) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className="flex size-6 shrink-0 items-center justify-center rounded text-ink-500 transition hover:bg-paper-100 hover:text-ink-900"
      onClick={onClick}
      {...aria}
    >
      {children}
    </button>
  )
}

interface TerritoryCartoucheProps {
  /** The selected territory's box on screen. */
  anchor: ScreenBox
  /** The map's size on screen. */
  size: { width: number; height: number }
  /** The selected territory; a new one resets the card's position. */
  territoryId: string
  /** Name and code shown on the collapsed pill. */
  label: string
  onClose: () => void
  /** Fills the active order field with this territory; absent when none is active. */
  onUseForOrder?: () => void
  children: ReactNode
}

interface DragStart {
  pointerId: number
  x: number
  y: number
  left: number
  top: number
  /** True once the pointer has travelled past the drag threshold. */
  moved: boolean
}

/**
 * The territory sheet as a cartouche laid on the map (charte-graphique.md
 * § Fiche de territoire): 280 px wide at most, by default on the side of the
 * territory away from the drawer (the left, the drawer being on the right)
 * when there is room, never over the territory itself when it can avoid it,
 * and tied to it by an ink line. Its grip bar drags it within the map and
 * folds it into a pill; Escape closes it.
 */
export function TerritoryCartouche({
  anchor,
  size,
  territoryId,
  label,
  onClose,
  onUseForOrder,
  children,
}: TerritoryCartoucheProps) {
  const { t } = useLanguage()
  const bodyId = useId()
  // Moved by hand, the cartouche keeps its place on screen through zoom and
  // pan; until then it follows the territory.
  const [dragged, setDragged] = useState<{ left: number; top: number } | null>(null)
  const [collapsed, setCollapsed] = useState(false)
  // A new selection brings the card back to its default place; it stays
  // folded or open as it was, like the sheet's « Détails », which is not
  // remounted.
  const [shownId, setShownId] = useState(territoryId)
  if (shownId !== territoryId) {
    setShownId(territoryId)
    setDragged(null)
  }
  const toggleLabel = t(collapsed ? 'map.expandCard' : 'map.collapseCard')
  const Chevron = collapsed ? IconChevronDown : IconChevronUp
  const dragStart = useRef<DragStart | null>(null)
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

  // Folding or unfolding changes the height at once: measure before paint so
  // the card is never drawn for a frame at the previous height's place.
  useLayoutEffect(() => {
    if (boxRef.current) setHeight(boxRef.current.offsetHeight)
  }, [collapsed])

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

  const fullWidth = Math.min(WIDTH, size.width - MARGIN * 2)
  const shownWidth = collapsed ? Math.min(PILL_WIDTH, fullWidth) : fullWidth
  const roomLeft = anchor.left - GAP - MARGIN
  const roomRight = size.width - anchor.right - GAP - MARGIN
  const onLeft = roomLeft >= fullWidth || roomLeft >= roomRight
  const centerY = (anchor.top + anchor.bottom) / 2
  const maxHeight = size.height - MARGIN * 2
  const shownHeight = Math.min(height, maxHeight)
  const clampLeft = (v: number) => clamp(v, MARGIN, size.width - MARGIN - shownWidth)
  const clampTop = (v: number) => clamp(v, MARGIN, size.height - MARGIN - shownHeight)
  const defaultLeft = onLeft ? anchor.left - GAP - shownWidth : anchor.right + GAP
  const left = clampLeft(dragged?.left ?? defaultLeft)
  const top = clampTop(dragged?.top ?? centerY - shownHeight / 2)

  const onGripDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.button !== 0 || dragStart.current) return
    event.currentTarget.setPointerCapture(event.pointerId)
    dragStart.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY,
      left,
      top,
      moved: false,
    }
  }
  const onGripMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const start = dragStart.current
    if (!start || start.pointerId !== event.pointerId) return
    const dx = event.clientX - start.x
    const dy = event.clientY - start.y
    // A click on the grip must not pin the card in place.
    if (!start.moved && Math.hypot(dx, dy) < DRAG_THRESHOLD) return
    start.moved = true
    const nextLeft = clampLeft(start.left + dx)
    const nextTop = clampTop(start.top + dy)
    setDragged((d) =>
      d && d.left === nextLeft && d.top === nextTop
        ? d
        : { left: nextLeft, top: nextTop },
    )
  }
  const onGripUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragStart.current?.pointerId === event.pointerId) dragStart.current = null
  }

  // The ink line runs from the cartouche's near edge to the territory centre.
  const lineToX = (anchor.left + anchor.right) / 2
  const cartoucheOnLeft = dragged ? left + shownWidth / 2 < lineToX : onLeft
  const lineFromX = cartoucheOnLeft ? left + shownWidth : left
  const lineFromY = Math.max(
    top + 12,
    Math.min(centerY, top + Math.max(shownHeight, 24) - 12),
  )
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
        data-side={cartoucheOnLeft ? 'left' : 'right'}
        data-collapsed={collapsed ? '' : undefined}
        className="absolute z-30 flex flex-col overflow-hidden rounded-md border border-line-strong bg-paper-50 shadow-panel"
        style={{ left, top, width: shownWidth, maxHeight }}
      >
        <div
          className={`flex shrink-0 items-center gap-1 px-1 py-1 ${collapsed ? '' : 'border-b border-line'}`}
        >
          <div
            data-cartouche-grip
            className="flex h-6 min-w-0 flex-1 cursor-grab touch-none select-none items-center gap-1 text-ink-500 active:cursor-grabbing"
            onPointerDown={onGripDown}
            onPointerMove={onGripMove}
            onPointerUp={onGripUp}
            onPointerCancel={onGripUp}
            onLostPointerCapture={onGripUp}
          >
            <IconGripHorizontal aria-hidden="true" className="size-4 shrink-0" />
            {collapsed && (
              <span
                className="truncate text-sm font-semibold italic text-ink-900"
                title={label}
              >
                {label}
              </span>
            )}
          </div>
          {/* Folded, the sheet's own « Use for the order » button is hidden. */}
          {collapsed && onUseForOrder && (
            <BarButton label={t('map.useForOrder')} onClick={onUseForOrder}>
              <FileArrowInIcon className="size-4" />
            </BarButton>
          )}
          <BarButton
            label={toggleLabel}
            onClick={() => setCollapsed((c) => !c)}
            aria-expanded={!collapsed}
            aria-controls={bodyId}
          >
            <Chevron aria-hidden="true" className="size-4" />
          </BarButton>
          <BarButton label={t('map.closeCard')} onClick={onClose}>
            <IconX aria-hidden="true" className="size-4" />
          </BarButton>
        </div>
        <div
          id={bodyId}
          data-cartouche-body
          className="min-h-0 overflow-y-auto p-3"
          hidden={collapsed}
        >
          {children}
        </div>
      </div>
    </>
  )
}
