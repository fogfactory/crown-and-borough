import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GameLayout } from '@/components/GameLayout'

describe('GameLayout', () => {
  it('renders the map and the drawer without a nested main landmark', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>}>
        <aside>Drawer body</aside>
      </GameLayout>,
    )

    expect(screen.getByText('Map body')).toBeInTheDocument()
    expect(screen.getByText('Drawer body')).toBeInTheDocument()
    expect(container.querySelectorAll('main')).toHaveLength(0)
  })

  it('lets the map take the free width and drops the public map banner', () => {
    const { container } = render(
      <GameLayout map={<p>Map body</p>}>
        <aside>Drawer body</aside>
      </GameLayout>,
    )

    const section = container.querySelector('section')
    expect(section?.className).toContain('min-w-0')
    expect(section?.className).toContain('flex-1')
    expect(screen.queryByText(/public map/i)).toBeNull()
  })
})
