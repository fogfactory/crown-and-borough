import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProjectedIncomeSummary } from '@/components/ProjectedIncomeSummary'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 1,
  season: 'spring',
  players: [
    {
      id: 'P1',
      name: 'Alice',
      color: '#a84632',
      capitalTerritory: 'ROS',
      projectedIncome: 6,
      projectedMillIncome: 2,
    },
    {
      id: 'P2',
      name: 'Bob',
      color: '#2d5f9e',
      projectedIncome: 0,
      projectedMillIncome: 0,
    },
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
    expect(
      screen.getByText('Territory: +6 R per action turn → ROS (capital)'),
    ).toBeInTheDocument()
    expect(screen.getByText('Mills: +2 R per action turn')).toBeInTheDocument()
  })

  it('omits the destination without a capital', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={state} playerId="P2" />
      </LanguageProvider>,
    )

    expect(screen.getByText('Territory: +0 R per action turn')).toBeInTheDocument()
    expect(screen.getByText('Mills: +0 R per action turn')).toBeInTheDocument()
  })

  it('splits territory income into one line per fief plus the non-fief remainder', () => {
    const fiefState: StateData = {
      ...state,
      players: [
        {
          id: 'P1',
          name: 'Alice',
          color: '#a84632',
          capitalTerritory: 'ROS',
          projectedIncome: 10,
          projectedMillIncome: 2,
        },
      ],
      fiefs: [
        {
          capital: 'BOI',
          title: 'barony',
          territories: ['BOI', 'BRU'],
          owner: 'P1',
          holder: 'JEA',
          projectedIncome: 4,
        },
        {
          capital: 'CHA',
          title: 'county',
          territories: ['CHA', 'ATL', 'NOR'],
          owner: 'P1',
          projectedIncome: 3,
        },
      ],
    }

    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={fiefState} playerId="P1" />
      </LanguageProvider>,
    )

    expect(
      screen.getByText('Outside fief: +3 R per action turn → ROS (capital)'),
    ).toBeInTheDocument()
    expect(screen.getByText('Barony of BOI: +4 R per action turn')).toBeInTheDocument()
    expect(screen.getByText('County of CHA: +3 R per action turn')).toBeInTheDocument()
    expect(screen.getByText('Mills: +2 R per action turn')).toBeInTheDocument()
    expect(
      screen.queryByText('Territory: +10 R per action turn → ROS (capital)'),
    ).not.toBeInTheDocument()
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
