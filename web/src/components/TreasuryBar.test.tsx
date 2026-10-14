import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { TreasuryBar } from '@/components/TreasuryBar'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 2,
  season: 'summer',
  players: [
    {
      id: 'P1',
      name: 'Alice',
      color: '#a84632',
      capitalTerritory: 'ROS',
      projectedIncome: 8,
      projectedMillIncome: 2,
      projectedConsumption: 1,
      armiesAtRisk: [],
    },
  ],
  territories: [
    { id: 'ROS', owner: 'P1', resources: 15, army: null, infrastructures: [] },
    { id: 'BOI', owner: 'P1', resources: 5, army: null, infrastructures: [] },
  ],
  nobles: [],
}

const renderBar = (s: StateData) =>
  render(
    <LanguageProvider initialLanguage="fr">
      <TreasuryBar state={s} playerId="P1" />
    </LanguageProvider>,
  )

describe('TreasuryBar', () => {
  it('sums the treasury up on one line and unfolds the detail', () => {
    const { container } = renderBar(state)

    const bar = screen.getByRole('button', { expanded: false })
    expect(container.querySelector('[data-treasury-summary]')).toHaveTextContent(
      '20 R·+10 R / tour·−1 R·famine : aucune',
    )
    expect(screen.queryByText('Revenu prévu')).toBeNull()

    fireEvent.click(bar)
    expect(bar).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Revenu prévu')).toBeInTheDocument()
    expect(screen.getByText('Consommation prévue')).toBeInTheDocument()
  })

  it('writes a famine risk in the warning tone', () => {
    const { container } = renderBar({
      ...state,
      players: [
        {
          ...state.players[0],
          armiesAtRisk: [
            { territoryId: 'ROS', size: 3, deficit: 2 },
            { territoryId: 'BOI', size: 1, deficit: 1 },
          ],
        },
      ],
    })

    const risk = container.querySelector('[data-famine-risk]')
    expect(risk).toHaveTextContent('famine : 2 armées')
    expect(risk).toHaveClass('text-warning-700')
  })

  it('leaves consumption and famine out in winter', () => {
    const { container } = renderBar({ ...state, season: 'winter' })

    expect(container.querySelector('[data-treasury-summary]')).toHaveTextContent(
      '20 R·+10 R / tour',
    )
    expect(screen.queryByText(/famine/)).toBeNull()
  })
})
