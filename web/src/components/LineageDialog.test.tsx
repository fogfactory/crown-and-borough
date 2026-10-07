import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { LineageDialog } from '@/components/LineageDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { SimulationRejectedError } from '@/lib/use-victory-simulation'
import type { Noble, SimulationAction, VictoryReading, VictorySimulation } from '@/types'

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

function renderDialog(simulationRequest?: (actions: SimulationAction[]) => Promise<VictorySimulation>) {
  render(
    <LanguageProvider initialLanguage="en">
      <LineageDialog
        defaultFocus="P1"
        simulationRequest={simulationRequest}
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

  describe('simulation', () => {
    const reading = (total: number, status: VictoryReading['status']): VictoryReading => ({
      score: { titles: total, total },
      victory: { mode: 'solo', required: 3 },
      status,
      missing: Math.max(3 - total, 0),
    })
    const projection = (killed: boolean): VictorySimulation => ({
      current: { P1: reading(2, 'ongoing'), P2: reading(1, 'ongoing') },
      projected: {
        P1: reading(killed ? 3 : 2, killed ? 'major' : 'ongoing'),
        P2: reading(1, killed ? 'failure' : 'ongoing'),
      },
      state: {
        turn: 1,
        season: 'spring',
        players: [
          { id: 'P1', name: 'Alice', color: '#a84632' },
          { id: 'P2', name: 'Bob', color: '#2f6f9f' },
        ],
        nobles: [noble('ANN', 'P2', 'Duchesse Anne', { sex: 'female' })],
        deceased: killed
          ? [{ code: 'JEA', name: 'Duc Jean', owner: 'P1', cause: 'natural', turn: 1 }]
          : [],
        fiefs: [],
        marriages: [],
        claims: [],
        scores: { P1: { titles: 3, total: 3 }, P2: { titles: 1, total: 1 } },
        victory: { soloThreshold: 3, allianceThreshold: 4, players: {} },
      } as unknown as VictorySimulation['state'],
    })

    it('kills a noble, updates the board and the status panel, then undoes it', async () => {
      const request = vi.fn(async (actions: SimulationAction[]) => projection(actions.length > 0))
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const card = screen.getByRole('region', { name: 'Alice' }).querySelector('[data-noble="JEA"]')!
      fireEvent.click(card)
      fireEvent.click(screen.getByRole('button', { name: 'Kill' }))

      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith([{ type: 'kill', noble: 'JEA' }], expect.anything()),
      )
      expect(await screen.findByTestId('panel-status-P1')).toHaveTextContent('Major victory')
      expect(screen.getByTestId('panel-delta-P1')).toHaveTextContent('+1')
      expect(screen.getByTestId('panel-status-P2')).toHaveTextContent('Defeat')
      expect(screen.getByText('Duc Jean dies')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: 'Undo' }))
      await waitFor(() => expect(request).toHaveBeenLastCalledWith([], expect.anything()))
      expect(screen.queryByText('Duc Jean dies')).not.toBeInTheDocument()
    })

    it('asks for a second click to marry two nobles', async () => {
      const request = vi.fn(async () => projection(false))
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      fireEvent.click(screen.getByRole('region', { name: 'Bob' }).querySelector('[data-noble="EVE"]')!)
      fireEvent.click(screen.getByRole('button', { name: 'Marry to…' }))
      expect(screen.getByRole('status')).toHaveTextContent('Click the spouse of Dame Eve')
      fireEvent.click(screen.getByRole('region', { name: 'Alice' }).querySelector('[data-noble="JEA"]')!)
      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [{ type: 'marry', noble: 'EVE', other: 'JEA' }],
          expect.anything(),
        ),
      )
    })

    it('drops an action the server refuses and says why', async () => {
      const request = vi.fn(async (actions: SimulationAction[]): Promise<VictorySimulation> => {
        if (actions.length > 0) throw new SimulationRejectedError('noble "JEA" is already married')
        return projection(false)
      })
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const card = screen.getByRole('region', { name: 'Alice' }).querySelector('[data-noble="JEA"]')!
      fireEvent.click(card)
      fireEvent.click(screen.getByRole('button', { name: 'Kill' }))
      expect(await screen.findByRole('alert')).toHaveTextContent('already married')
      await waitFor(() => expect(screen.queryByText('Duc Jean dies')).not.toBeInTheDocument())
      expect(within(screen.getByRole('button', { name: 'Reset' }).parentElement!).getByRole('button', { name: 'Reset' })).toBeDisabled()
    })
  })
})
