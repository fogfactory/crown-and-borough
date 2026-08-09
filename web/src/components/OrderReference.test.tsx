import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { OrderReference } from '@/components/OrderReference'

describe('OrderReference', () => {
  it('shows action orders and multi-territory dispersion examples', () => {
    const { container } = render(<OrderReference season="spring" />)

    expect(screen.getByText("Ordres des saisons d'action")).toBeInTheDocument()
    expect(screen.getByText('Attaque / déplacement')).toBeInTheDocument()
    expect(screen.getByText('BRI D ATL NOR BRU')).toBeInTheDocument()
    expect(screen.getByText('BRI D ATL*HUG*JEA NOR*ROS')).toBeInTheDocument()
    expect(screen.getByText('(BRI D ATL NOR)')).toBeInTheDocument()
    expect(screen.getByText('BRI S ATL - NOR')).toBeInTheDocument()
    expect(screen.getByText('single')).toBeInTheDocument()
    expect(screen.getByText('loop')).toBeInTheDocument()
    expect(container.querySelector('details')).not.toHaveAttribute('open')
  })

  it('shows winter investments and their costs separately', () => {
    render(<OrderReference season="winter" />)

    expect(screen.getByText('Investissements d’hiver')).toBeInTheDocument()
    expect(screen.getByText('Recruter un noble')).toBeInTheDocument()
    expect(screen.getByText('R N ATL')).toBeInTheDocument()
    expect(screen.getByText('C C ATL')).toBeInTheDocument()
    expect(screen.getByText('10 R')).toBeInTheDocument()
    expect(screen.getByText('L N HUG')).toBeInTheDocument()
    expect(screen.queryByText("Ordres des saisons d'action")).not.toBeInTheDocument()
  })
})
