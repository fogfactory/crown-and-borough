import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { TerritoryCartouche } from '@/components/TerritoryCartouche'

const size = { width: 1000, height: 700 }

describe('TerritoryCartouche', () => {
  it('sits on the left of the territory when there is room, tied by an ink line', () => {
    const { container } = render(
      <TerritoryCartouche
        anchor={{ left: 600, top: 300, right: 660, bottom: 340 }}
        size={size}
        onClose={vi.fn()}
      >
        <p>Rochevent</p>
      </TerritoryCartouche>,
    )

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
      <TerritoryCartouche
        anchor={{ left: 40, top: 300, right: 100, bottom: 340 }}
        size={size}
        onClose={vi.fn()}
      >
        <p>Rochevent</p>
      </TerritoryCartouche>,
    )

    const box = container.querySelector('[data-map-cartouche]') as HTMLElement
    expect(box).toHaveAttribute('data-side', 'right')
    expect(parseFloat(box.style.left)).toBeGreaterThan(100)
  })

  it('closes on Escape', () => {
    const onClose = vi.fn()
    render(
      <TerritoryCartouche
        anchor={{ left: 600, top: 300, right: 660, bottom: 340 }}
        size={size}
        onClose={onClose}
      >
        <p>Rochevent</p>
      </TerritoryCartouche>,
    )

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('leaves the Escape to an open popover, and drops the line off screen', () => {
    const onClose = vi.fn()
    const { container } = render(
      <TerritoryCartouche
        anchor={{ left: -200, top: 300, right: -140, bottom: 340 }}
        size={size}
        onClose={onClose}
      >
        <div data-radix-popper-content-wrapper>popover</div>
      </TerritoryCartouche>,
    )

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
    expect(container.querySelector('[data-cartouche-line]')).toBeNull()
  })
})
