import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { MapLegend } from '@/components/MapLegend'

describe('MapLegend', () => {
  it('renders the map legend as a standalone panel', () => {
    const { container } = render(<MapLegend />)

    expect(screen.getByText('Légende')).toBeInTheDocument()
    expect(screen.getByText('Plaine')).toBeInTheDocument()
    expect(
      screen.getByText(/Trait continu épais = frontière infranchissable/),
    ).toBeInTheDocument()
    expect(
      container.querySelector('svg[aria-label="Carte des territoires"]'),
    ).not.toBeInTheDocument()
  })
})
