import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LineageDialog } from '@/components/LineageDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble } from '@/types'

const noble = (
  code: string,
  owner: string,
  name: string,
  extra: Partial<Noble> = {},
): Noble => ({
  id: code,
  code,
  name,
  owner,
  location: 'ROS',
  status: 'free',
  ...extra,
})

function renderDialog() {
  render(
    <LanguageProvider initialLanguage="en">
      <LineageDialog
        defaultFocus="P1"
        players={[
          { id: 'P1', name: 'Alice', color: '#a84632', succession: ['JEA'] },
          { id: 'P2', name: 'Bob', color: '#2f6f9f', succession: ['ANN', 'EVE'] },
        ]}
        nobles={[
          noble('JEA', 'P1', 'Duc Jean', { sex: 'male', dignities: ['bastard'] }),
          noble('ANN', 'P2', 'Duchesse Anne', { sex: 'female' }),
          noble('EVE', 'P2', 'Dame Eve', { sex: 'female' }),
        ]}
        deceased={[
          { code: 'OLD', name: 'Vieux Hugon', owner: 'P1', cause: 'execution', turn: 2 },
        ]}
        fiefs={[
          {
            capital: 'ROC',
            title: 'duchy',
            territories: ['ROC'],
            owner: 'P1',
            holder: 'JEA',
          },
          {
            capital: 'AAA',
            title: 'barony',
            territories: ['AAA'],
            owner: 'P1',
            holder: 'JEA',
          },
        ]}
        marriages={[
          {
            nobleA: 'JEA',
            nobleB: 'ANN',
            turn: 1,
            active: true,
            category: 'head',
            weight: 5,
            activeHeadFor: ['P1', 'P2'],
            headSuccessors: [
              { player: 'P1' },
              { player: 'P2', marriage: { nobleA: 'JEA', nobleB: 'EVE' } },
            ],
          },
          { nobleA: 'OLD', nobleB: 'EVE', turn: 1, active: false },
        ]}
        claims={[{ heir: 'EVE', target: 'JEA', spouse: 'ANN', turn: 1, rank: 1 }]}
        scores={{ P1: { titles: 2, total: 2 }, P2: { titles: 1, total: 1 } }}
        victory={{
          soloThreshold: 6,
          allianceThreshold: 9,
          players: {
            P1: { mode: 'alliance', required: 9, partner: 'P2' },
            P2: { mode: 'alliance', required: 9, partner: 'P1' },
          },
        }}
      />
    </LanguageProvider>,
  )
  fireEvent.click(screen.getByRole('button', { name: 'Lineage and alliances' }))
}

describe('LineageDialog', () => {
  it('shows each family as a column with every title, the deceased and the claims', () => {
    renderDialog()
    const alice = screen.getByRole('region', { name: 'Alice' })
    expect(alice).toHaveTextContent('Duc Jean')
    expect(alice).toHaveTextContent('Duchy · ROC')
    expect(alice).toHaveTextContent('Barony · AAA')
    expect(alice).toHaveTextContent('Bastard')
    expect(alice).toHaveTextContent('Claimed ×1')
    expect(alice).toHaveTextContent('Vieux Hugon')
    expect(alice).toHaveTextContent('Executed')
    expect(alice).toHaveTextContent('Head · 5 ★')
    expect(screen.getByRole('region', { name: 'Bob' })).toHaveTextContent(
      'Claimant to Duchy · ROC #1',
    )
  })

  it('filters families and ended marriages', () => {
    renderDialog()
    fireEvent.click(screen.getByRole('button', { name: 'Bob' }))
    expect(screen.queryByRole('region', { name: 'Bob' })).not.toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Alice' })).toBeInTheDocument()
  })

  it('lists the marriage to cancel and the one taking over in the graph', () => {
    renderDialog()
    fireEvent.click(screen.getByRole('tab', { name: 'Graph' }))
    expect(screen.getByText('Duc Jean × Duchesse Anne')).toBeInTheDocument()
    expect(screen.getByText(/Takes over for Bob/)).toBeInTheDocument()
    expect(screen.getByText('Duc Jean × Dame Eve')).toBeInTheDocument()
    expect(screen.getByText(/Alice would have no head/)).toBeInTheDocument()
  })
})
