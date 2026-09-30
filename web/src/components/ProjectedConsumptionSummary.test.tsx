import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProjectedConsumptionSummary } from '@/components/ProjectedConsumptionSummary'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    {
      id: 'P1',
      name: 'Alice',
      color: '#a84632',
      projectedConsumption: 6,
      armiesAtRisk: [{ territoryId: 'MOR', size: 2, deficit: 1 }],
    },
    { id: 'P2', name: 'Bob', color: '#2d5f9e', projectedConsumption: 1, armiesAtRisk: [] },
  ],
  territories: [],
  nobles: [],
}

const winterState: StateData = { ...state, season: 'winter' }

describe('ProjectedConsumptionSummary', () => {
  it('shows the amount and the armies at risk', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedConsumptionSummary state={state} playerId="P1" />
      </LanguageProvider>,
    )

    expect(screen.getByText('Projected consumption')).toBeInTheDocument()
    expect(screen.getByText('−6 R per action turn')).toBeInTheDocument()
    expect(screen.getByText('MOR (2 troops): −1 R short')).toBeInTheDocument()
  })

  it('shows no army at risk when none is flagged', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedConsumptionSummary state={state} playerId="P2" />
      </LanguageProvider>,
    )

    expect(screen.getByText('No army looks at risk')).toBeInTheDocument()
  })

  it('renders nothing in winter', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <ProjectedConsumptionSummary state={winterState} playerId="P1" />
      </LanguageProvider>,
    )

    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing without a selected player', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <ProjectedConsumptionSummary state={state} playerId={null} />
      </LanguageProvider>,
    )

    expect(container).toBeEmptyDOMElement()
  })
})
