import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProjectedIncomeSummary } from '@/components/ProjectedIncomeSummary'
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
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 10,
      army: null,
      infrastructures: [],
    },
    {
      id: 'BOI',
      owner: 'P1',
      resources: 15,
      army: null,
      infrastructures: [],
    },
    {
      id: 'BRU',
      owner: 'P2',
      resources: 5,
      army: null,
      infrastructures: [],
    },
  ],
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
      screen.getByText(withText('Territory: +6 R → ROS (capital)')),
    ).toBeInTheDocument()
    expect(screen.getByText('Mills: +2 R')).toBeInTheDocument()
    expect(screen.getByText('Current stockpile: 25 R')).toBeInTheDocument()
  })

  it('omits the destination without a capital', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <ProjectedIncomeSummary state={state} playerId="P2" />
      </LanguageProvider>,
    )

    expect(screen.getByText('Territory: +0 R')).toBeInTheDocument()
    expect(screen.getByText('Mills: +0 R')).toBeInTheDocument()
    expect(screen.getByText('Current stockpile: 5 R')).toBeInTheDocument()
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
      territories: [
        {
          id: 'ROS',
          owner: 'P1',
          resources: 12,
          army: null,
          infrastructures: [],
        },
        {
          id: 'BOI',
          owner: 'P1',
          resources: 8,
          army: null,
          infrastructures: [],
        },
        {
          id: 'BRU',
          owner: 'P1',
          resources: 10,
          army: null,
          infrastructures: [],
        },
        {
          id: 'CHA',
          owner: 'P1',
          resources: 6,
          army: null,
          infrastructures: [],
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
      screen.getByText(withText('Outside fief: +3 R → ROS (capital)')),
    ).toBeInTheDocument()
    expect(screen.getByText(withText('Barony of BOI: +4 R'))).toBeInTheDocument()
    expect(screen.getByText(withText('County of CHA: +3 R'))).toBeInTheDocument()
    expect(screen.getByText('Mills: +2 R')).toBeInTheDocument()
    expect(screen.getByText('Current stockpile: 36 R')).toBeInTheDocument()
    expect(screen.queryByText('Territory: +10 R → ROS (capital)')).not.toBeInTheDocument()
  })

  describe('royal income and drafted taxes', () => {
    const royalState: StateData = {
      ...state,
      king: 'KNG',
      nobles: [
        {
          id: 'n1',
          code: 'KNG',
          name: 'Robert',
          owner: 'P1',
          location: 'ROS',
          status: 'free',
        },
        {
          id: 'n2',
          code: 'JEA',
          name: 'Jean',
          owner: 'P1',
          location: 'BOI',
          status: 'free',
        },
        {
          id: 'n3',
          code: 'LEO',
          name: 'Leon',
          owner: 'P2',
          location: 'BRU',
          status: 'free',
        },
      ],
      players: [
        {
          ...state.players[0],
          projectedIncome: 4,
          projectedMillIncome: 0,
          projectedRoyalRent: 3,
        },
        state.players[1],
      ],
      fiefs: [
        {
          capital: 'BOI',
          title: 'barony',
          territories: ['BOI'],
          owner: 'P1',
          holder: 'JEA',
          projectedIncome: 4,
        },
        {
          capital: 'BRU',
          title: 'barony',
          territories: ['BRU'],
          owner: 'P2',
          holder: 'LEO',
          projectedIncome: 3,
        },
      ],
    }

    function renderSummary(specialDraft?: string) {
      render(
        <LanguageProvider initialLanguage="en">
          <ProjectedIncomeSummary
            state={royalState}
            playerId="P1"
            specialDraft={specialDraft}
          />
        </LanguageProvider>,
      )
    }

    it('adds the royal rent to the total', () => {
      renderSummary()
      expect(screen.getByText('Royal rent: +3 R')).toBeInTheDocument()
      expect(screen.getByText(withText('Barony of BOI: +4 R'))).toBeInTheDocument()
      expect(screen.getByText('Total: +7 R per action turn')).toBeInTheDocument()
    })

    it('marks a taxed fief and counts the doubling and the royal supplement in the total', () => {
      renderSummary('P TX JEA BOI # Tax\nP TX KNG BRU # Tax')
      expect(screen.getByText(withText('Barony of BOI: +8 R (Tax)'))).toBeInTheDocument()
      expect(screen.getByText('Royal tax: +3 R')).toBeInTheDocument()
      // 4 territory + 4 doubling + 3 rent + 3 royal supplement
      expect(screen.getByText('Total: +14 R per action turn')).toBeInTheDocument()
    })

    it('shows a drafted tithe even when it diverts nothing', () => {
      const popeState: StateData = {
        ...royalState,
        nobles: [
          ...royalState.nobles,
          {
            id: 'n4',
            code: 'POP',
            name: 'Pie',
            owner: 'P1',
            location: 'ROS',
            status: 'free',
            religiousTitle: 'pope',
          },
        ],
        bishoprics: [{ region: 'BRU', name: 'Bru', territories: ['BRU'] }],
      }
      render(
        <LanguageProvider initialLanguage="en">
          <ProjectedIncomeSummary
            state={popeState}
            playerId="P1"
            specialDraft="P DI POP # Tax"
          />
        </LanguageProvider>,
      )
      expect(screen.getByText('Tithe: +0 R')).toBeInTheDocument()
    })

    it('shows no tithe line without a drafted tithe', () => {
      renderSummary('P TX JEA BOI # Tax')
      expect(screen.queryByText(/^Tithe:/)).not.toBeInTheDocument()
    })
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
