import type { ComponentProps, ReactNode } from 'react'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { TerritoryCartouche } from '@/components/TerritoryCartouche'

const size = { width: 1000, height: 700 }
const anchor = { left: 600, top: 300, right: 660, bottom: 340 }

function cartouche(
  props: Partial<ComponentProps<typeof TerritoryCartouche>> = {},
  body: ReactNode = <p>Rochevent</p>,
) {
  return (
    <TerritoryCartouche
      anchor={anchor}
      size={size}
      territoryId="ROC"
      label="Rochevent (ROC)"
      onClose={vi.fn()}
      {...props}
    >
      {body}
    </TerritoryCartouche>
  )
}

describe('TerritoryCartouche', () => {
  it('sits on the left of the territory when there is room, tied by an ink line', () => {
    const { container } = render(cartouche())

    const box = container.querySelector('[data-map-cartouche]') as HTMLElement
    expect(box).toHaveAttribute('data-side', 'left')
    expect(box.style.width).toBe('280px')
    // Its right edge stays clear of the territory.
    expect(parseFloat(box.style.left) + 280).toBeLessThan(600)
    expect(container.querySelector('[data-cartouche-line] line')).not.toBeNull()
    expect(screen.getByText('Rochevent')).toBeInTheDocument()
  })

  it('moves to the right of a territory near the left edge', () => {
    const { container } = render(
      cartouche({ anchor: { left: 40, top: 300, right: 100, bottom: 340 } }),
    )

    const box = container.querySelector('[data-map-cartouche]') as HTMLElement
    expect(box).toHaveAttribute('data-side', 'right')
    expect(parseFloat(box.style.left)).toBeGreaterThan(100)
  })

  it('closes on Escape', () => {
    const onClose = vi.fn()
    render(cartouche({ onClose }))

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('collapses to a pill and expands again without closing', () => {
    const onClose = vi.fn()
    const { container } = render(cartouche({ onClose }, <p>Detail</p>))
    const box = container.querySelector('[data-map-cartouche]') as HTMLElement
    const body = container.querySelector('[data-cartouche-body]') as HTMLElement
    expect(screen.queryByText('Rochevent (ROC)')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Collapse the territory card' }))
    expect(box).toHaveAttribute('data-collapsed')
    expect(body).toHaveAttribute('hidden')
    expect(box.style.width).toBe('200px')
    expect(screen.getByText('Rochevent (ROC)')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Expand the territory card' }))
    expect(box).not.toHaveAttribute('data-collapsed')
    expect(body).not.toHaveAttribute('hidden')
    expect(box.style.width).toBe('280px')
  })

  it('offers the use-for-order button on the pill only', () => {
    const onUseForOrder = vi.fn()
    render(cartouche({ onUseForOrder }))
    expect(screen.queryByRole('button', { name: 'Use for the order' })).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Collapse the territory card' }))
    const box = document.querySelector('[data-map-cartouche]') as HTMLElement
    expect(box.style.width).toBe('200px')
    fireEvent.click(screen.getByRole('button', { name: 'Use for the order' }))
    expect(onUseForOrder).toHaveBeenCalledOnce()
  })

  it('has no use-for-order button on the pill when no field waits', () => {
    render(cartouche())
    fireEvent.click(screen.getByRole('button', { name: 'Collapse the territory card' }))
    expect(screen.queryByRole('button', { name: 'Use for the order' })).toBeNull()
  })

  it('closes from the pill and on Escape while collapsed', () => {
    const onClose = vi.fn()
    render(cartouche({ onClose }))
    fireEvent.click(screen.getByRole('button', { name: 'Collapse the territory card' }))

    fireEvent.click(screen.getByRole('button', { name: 'Close the territory card' }))
    expect(onClose).toHaveBeenCalledOnce()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  describe('dragging', () => {
    // jsdom has no layout: the card is 0 px high, so the default spot is
    // left 296 (left of the territory) and top 320 (centred on it).
    function setup() {
      const view = render(cartouche())
      const box = view.container.querySelector('[data-map-cartouche]') as HTMLElement
      const grip = view.container.querySelector('[data-cartouche-grip]') as HTMLElement
      return { ...view, box, grip }
    }
    const down = (grip: HTMLElement, x = 100, y = 100) =>
      fireEvent.pointerDown(grip, { clientX: x, clientY: y, button: 0, pointerId: 1 })
    const move = (grip: HTMLElement, x: number, y: number, pointerId = 1) =>
      fireEvent.pointerMove(grip, { clientX: x, clientY: y, pointerId })

    it('follows the pointer by its delta, then stops on release', () => {
      const { box, grip } = setup()
      down(grip)
      move(grip, 130, 90)
      expect(box.style.left).toBe('326px')
      expect(box.style.top).toBe('310px')

      fireEvent.pointerUp(grip, { pointerId: 1 })
      move(grip, 400, 400)
      expect(box.style.left).toBe('326px')
      expect(box.style.top).toBe('310px')
    })

    it('stays within the map on every side', () => {
      const { box, grip } = setup()
      down(grip)
      move(grip, -5000, -5000)
      expect(box.style.left).toBe('8px')
      expect(box.style.top).toBe('8px')
      move(grip, 5000, 5000)
      // 1000 − 8 − 280 on the right; no height to subtract in jsdom.
      expect(box.style.left).toBe('712px')
      expect(box.style.top).toBe('692px')
    })

    it('ignores a click or a jitter below the drag threshold', () => {
      const { box, grip, rerender } = setup()
      down(grip)
      move(grip, 101, 101)
      fireEvent.pointerUp(grip, { pointerId: 1 })
      // Still following its territory: it shifts with the anchor.
      rerender(cartouche({ anchor: { ...anchor, left: 500, right: 560 } }))
      expect(box.style.left).toBe('196px')
    })

    it('ignores other buttons and other pointers', () => {
      const { box, grip } = setup()
      fireEvent.pointerDown(grip, { clientX: 100, clientY: 100, button: 2, pointerId: 1 })
      move(grip, 300, 300)
      expect(box.style.left).toBe('296px')

      down(grip)
      move(grip, 300, 300, 2)
      expect(box.style.left).toBe('296px')
    })

    it('keeps its place on screen when the territory moves', () => {
      const { box, grip, rerender } = setup()
      down(grip)
      move(grip, 130, 90)
      fireEvent.pointerUp(grip, { pointerId: 1 })

      rerender(cartouche({ anchor: { ...anchor, left: 500, right: 560 } }))
      expect(box.style.left).toBe('326px')
    })

    it('puts the line start on the near edge once dragged to the right', () => {
      const { container, box, grip } = setup()
      down(grip)
      move(grip, 900, 100)
      fireEvent.pointerUp(grip, { pointerId: 1 })

      expect(box).toHaveAttribute('data-side', 'right')
      const line = container.querySelector('[data-cartouche-line] line') as SVGLineElement
      expect(line.getAttribute('x1')).toBe(String(parseFloat(box.style.left)))
    })
  })

  it('returns to its default place for another territory but stays folded', () => {
    const { container, rerender } = render(cartouche())
    const box = container.querySelector('[data-map-cartouche]') as HTMLElement
    const grip = container.querySelector('[data-cartouche-grip]') as HTMLElement
    fireEvent.pointerDown(grip, { clientX: 100, clientY: 100, button: 0, pointerId: 1 })
    fireEvent.pointerMove(grip, { clientX: 300, clientY: 300, pointerId: 1 })
    fireEvent.pointerUp(grip, { pointerId: 1 })
    fireEvent.click(screen.getByRole('button', { name: 'Collapse the territory card' }))
    expect(box).toHaveAttribute('data-collapsed')

    // Same territory: nothing changes.
    rerender(cartouche())
    expect(box).toHaveAttribute('data-collapsed')

    rerender(cartouche({ territoryId: 'BER', label: 'Berville (BER)' }))
    expect(box).toHaveAttribute('data-collapsed')
    expect(box.style.top).toBe('320px')

    fireEvent.click(screen.getByRole('button', { name: 'Expand the territory card' }))
    rerender(cartouche({ territoryId: 'ROC' }))
    expect(box).not.toHaveAttribute('data-collapsed')
    expect(box.style.left).toBe('296px')
  })

  it('keeps the sheet mounted, with its own state, across territories', () => {
    const { container, rerender } = render(
      cartouche(
        {},
        <details data-testid="more">
          <summary>More</summary>
        </details>,
      ),
    )
    const details = screen.getByTestId('more') as HTMLDetailsElement
    details.open = true

    rerender(
      cartouche(
        { territoryId: 'BER', label: 'Berville (BER)' },
        <details data-testid="more">
          <summary>More</summary>
        </details>,
      ),
    )
    expect((screen.getByTestId('more') as HTMLDetailsElement).open).toBe(true)
    expect(container.querySelectorAll('[data-map-cartouche]')).toHaveLength(1)
  })
})
