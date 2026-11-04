import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProjectedConsumptionSummary } from '@/components/ProjectedConsumptionSummary'
import type { StateData } from '@/types'
import { withText } from '@/test/text'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    {
      id: 'P1',
      name: 'Alice',
      color: '#a84632',
      projectedConsumption: 6,
      armiesAtRisk: [
        { territoryId: 'MOR', size: 2, deficit: 1, suppliedStrength: 1, expectedLoss: 1 },
      ],
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
    expect(screen.getByText(withText('MOR (2 troops): −1 R short, fed up to 1 · −1 troops'))).toBeInTheDocument()
  })

  it('tells which infrastructure a starving army pillages', () => {
    const pillaging: StateData = {
      ...state,
      players: [
        {
          ...state.players[0],
          armiesAtRisk: [
            { territoryId: 'MOR', size: 4, deficit: 5, suppliedStrength: 2, expectedLoss: 2, pillageInfrastructure: 'mill' },
            { territoryId: 'VIL', size: 2, deficit: 1, suppliedStrength: 1, expectedLoss: 0, pillageInfrastructure: 'castle', savedByPillage: true },
          ],
        },
      ],
    }
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedConsumptionSummary state={pillaging} playerId="P1" />
      </LanguageProvider>,
    )

    expect(screen.getByText(withText('MOR (4 troops): −5 R short, fed up to 2 · pillages its position (Mill) · −2 troops'))).toBeInTheDocument()
    expect(screen.getByText(withText('VIL (2 troops): −1 R short, fed up to 1 · pillages its position (Castle), which feeds it'))).toBeInTheDocument()
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
