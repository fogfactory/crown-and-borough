import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProjectedIncomeSummary } from '@/components/ProjectedIncomeSummary'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    { id: 'P1', name: 'Alice', color: '#a84632', capitalTerritory: 'ROS', projectedIncome: 6 },
    { id: 'P2', name: 'Bob', color: '#2d5f9e', projectedIncome: 0 },
  ],
  territories: [],
  nobles: [],
}

describe('ProjectedIncomeSummary', () => {
  it('shows the amount and the capital destination', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={state} playerId="P1" />
      </LanguageProvider>,
    )

    expect(screen.getByText('Projected income')).toBeInTheDocument()
    expect(screen.getByText('+6 R per action turn → ROS (capital)')).toBeInTheDocument()
  })

  it('omits the destination without a capital', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={state} playerId="P2" />
      </LanguageProvider>,
    )

    expect(screen.getByText('+0 R per action turn')).toBeInTheDocument()
  })

  it('renders nothing without a selected player', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={state} playerId={null} />
      </LanguageProvider>,
    )

    expect(container).toBeEmptyDOMElement()
  })
})
