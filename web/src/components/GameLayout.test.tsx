import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GameLayout } from '@/components/GameLayout'

describe('GameLayout', () => {
  it('renders the map and the sheet content without a nested main landmark', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>}>
        <p>Panel body</p>
      </GameLayout>,
    )

    expect(screen.getByText('Map body')).toBeInTheDocument()
    expect(screen.getByText('Panel body')).toBeInTheDocument()
    expect(container.querySelectorAll('main')).toHaveLength(0)
  })

  it('lets the map section shrink so the sidebar never overflows horizontally', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>}>
        <p>Panel body</p>
      </GameLayout>,
    )

    const section = container.querySelector('section')
    expect(section?.className).toContain('min-w-0')
    expect(section?.className).toContain('flex-1')
    expect(section?.className).not.toContain('lg:flex-none')
  })

  it('brings the sheet back to half when the focus signal increments', () => {
    const { rerender } = render(
      <GameLayout map={<p>Map body</p>} focusSignal={0}>
        <p>Panel body</p>
      </GameLayout>,
    )
    const sheet = screen.getByTestId('panel-sheet')

    fireEvent.click(screen.getByRole('button', { name: 'Expand panel' }))
    expect(sheet).toHaveAttribute('data-sheet-snap', 'full')

    rerender(
      <GameLayout map={<p>Map body</p>} focusSignal={1}>
        <p>Panel body</p>
      </GameLayout>,
    )
    expect(sheet).toHaveAttribute('data-sheet-snap', 'half')
  })
})
