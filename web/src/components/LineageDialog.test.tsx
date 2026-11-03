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

function renderDialog(
  simulationRequest?: (actions: SimulationAction[]) => Promise<VictorySimulation>,
  link = false,
  focus: string | null = null,
) {
  render(
    <LanguageProvider initialLanguage="en">
      <LineageDialog
        defaultFocus={focus}
        simulationRequest={simulationRequest}
        players={[
          { id: 'P1', name: 'Alice', color: '#a84632', succession: ['JEA'] },
          { id: 'P2', name: 'Bob', color: '#2f6f9f', succession: ['ANN', 'EVE'] },
        ]}
        nobles={[
          noble('JEA', 'P1', 'Duc Jean', {
            sex: 'male',
            dignities: ['bastard'],
            commissioned: link ? ['VIC'] : undefined,
          }),
          noble('ANN', 'P2', 'Duchesse Anne', { sex: 'female' }),
          noble('EVE', 'P2', 'Dame Eve', { sex: 'female' }),
        ]}
        deceased={[
          { code: 'OLD', name: 'Vieux Hugon', owner: 'P1', cause: 'execution', turn: 2 },
          {
            code: 'VIC',
            name: 'Victor',
            owner: 'P2',
            cause: 'assassination',
            turn: 3,
            commissioner: link ? 'JEA' : undefined,
          },
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
  it('lists the nobles of each family with every title and the deceased', () => {
    renderDialog()
    const alice = screen.getByRole('region', { name: 'Alice' })
    expect(alice).toHaveTextContent('Duc Jean')
    expect(alice).toHaveTextContent('Duchy · ROC')
    expect(alice).toHaveTextContent('Barony · AAA')
    expect(alice).toHaveTextContent('Bastard')
    expect(alice).toHaveTextContent('Vieux Hugon')
    expect(alice).toHaveTextContent('Executed')
    expect(screen.getByRole('region', { name: 'Bob' })).toHaveTextContent('Victor')
  })

  it('gives every noble a scroll with the blazon of its house, its name and its rank', () => {
    renderDialog()
    // No column banner, no pawn: the shield on each scroll tells the house.
    expect(screen.queryByRole('heading', { level: 3 })).not.toBeInTheDocument()
    const jean = document.querySelector('[data-noble="JEA"]')!
    expect(jean.querySelector('[data-shield-icon]')).toBeInTheDocument()
    expect(jean).toHaveTextContent('Duc Jean')
    expect(jean.querySelector('svg text')).toHaveTextContent('1')
    const ids = [
      ...document.querySelectorAll('[data-noble] [data-shield-icon] symbol'),
    ].map((symbol) => symbol.id)
    expect(new Set(ids).size).toBe(ids.length)
    // The dead have no rank.
    expect(document.querySelector('[data-noble="OLD"] svg text')).toBeNull()
  })

  it('hangs the heir below the couple that carries the claim, side by side', () => {
    renderDialog()
    const top = (code: string) =>
      parseFloat(
        (document.querySelector(`[data-noble="${code}"]`) as HTMLElement).style.top,
      )
    const left = (code: string) =>
      parseFloat(
        (document.querySelector(`[data-noble="${code}"]`) as HTMLElement).style.left,
      )
    expect(top('JEA')).toBe(top('ANN'))
    expect(Math.abs(left('JEA') - left('ANN'))).toBeGreaterThan(190)
    expect(top('EVE')).toBeGreaterThan(top('JEA'))
  })

  it('draws the head marriage and the claim as lines between the scrolls', () => {
    renderDialog()
    const marriage = document.querySelector('[data-marriage="JEA-ANN"]')!
    expect(marriage).toHaveAttribute('data-kind', 'head')
    expect(marriage).toHaveAttribute('data-aligned', 'true')
    expect(document.querySelector('[data-marriage="OLD-EVE"]')).toHaveAttribute(
      'data-kind',
      'ended',
    )
    const claim = document.querySelector('[data-claim="EVE>JEA"]')!
    expect(claim).toHaveAttribute('data-ended', 'false')
    expect(claim).toHaveAttribute('stroke-dasharray')
    // Vines end in a leaf; each marriage is tied by a knot of its kind.
    expect(claim.parentElement!.querySelectorAll('path')).toHaveLength(2)
    expect(
      document.querySelector('[data-marriage="JEA-ANN"] [data-knot]'),
    ).toHaveAttribute('data-knot', 'head')
    expect(
      document.querySelector('[data-marriage="OLD-EVE"] [data-knot]'),
    ).toHaveAttribute('data-knot', 'ended')
    // Only the head marriage grows ivy: leaves along its bond.
    expect(
      document.querySelectorAll('[data-marriage="JEA-ANN"] [data-ivy] path').length,
    ).toBeGreaterThanOrEqual(2)
    expect(document.querySelector('[data-marriage="OLD-EVE"] [data-ivy]')).toBeNull()
    expect(screen.getByTestId('lineage-tapestry')).toBeInTheDocument()
    // No badge repeats what the lines say, and cards cannot be dragged.
    expect(screen.queryByText(/Claimed ×/)).not.toBeInTheDocument()
    expect(screen.queryByText('Reset layout')).not.toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  })

  it('links the commissioner and its victim only when the state carries the link', () => {
    renderDialog(undefined, true)
    expect(screen.getByRole('region', { name: 'Bob' })).toHaveTextContent(
      'Killed on the orders of Duc Jean (JEA)',
    )
    expect(screen.getByRole('region', { name: 'Alice' })).toHaveTextContent(
      'Had Victor (VIC) killed',
    )
  })

  it('names no commissioner without the link', () => {
    renderDialog()
    expect(screen.getByRole('region', { name: 'Bob' })).toHaveTextContent('Victor')
    expect(screen.queryByTestId('assassination-commissioner')).not.toBeInTheDocument()
    expect(screen.queryByTestId('assassination-victim')).not.toBeInTheDocument()
  })

  it('has no family pills nor ended or claim checkboxes, and always draws them', () => {
    renderDialog()
    expect(screen.queryByRole('group', { name: 'Families' })).not.toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(document.querySelector('[data-marriage="OLD-EVE"]')).toBeInTheDocument()
    expect(document.querySelector('[data-claim="EVE>JEA"]')).toBeInTheDocument()
  })

  it('shows every house without a focus', () => {
    renderDialog()
    expect(screen.getByLabelText('Focus')).toHaveValue('')
    expect(screen.getByRole('option', { name: 'All houses' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Alice' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Bob' })).toBeInTheDocument()
    expect(document.querySelector('[data-noble="VIC"]')).toBeInTheDocument()
  })

  it('keeps the focused house, its spouses and its heirs, and hides the others', () => {
    renderDialog(undefined, false, 'P1')
    const nobles = [...document.querySelectorAll('[data-noble]')].map((el) =>
      el.getAttribute('data-noble'),
    )
    // JEA and OLD (P1), ANN and EVE (spouses, the ended marriage included).
    expect(nobles.sort()).toEqual(['ANN', 'EVE', 'JEA', 'OLD'])
    expect(document.querySelector('[data-noble="VIC"]')).toBeNull()
    // Linked nobles keep the blazon of their own house, and their succession rank.
    expect(screen.getByRole('region', { name: 'Bob' })).toHaveTextContent('Duchesse Anne')
    expect(document.querySelector('[data-noble="EVE"] svg text')).toHaveTextContent('2')
    expect(document.querySelector('[data-marriage="OLD-EVE"]')).toBeInTheDocument()
    expect(document.querySelector('[data-claim="EVE>JEA"]')).toBeInTheDocument()
    // The footer and the inset keep every house.
    fireEvent.click(screen.getByRole('button', { name: 'Alliances' }))
    expect(screen.getByText('Duc Jean × Dame Eve')).toBeInTheDocument()
  })

  it('shows the heirs of a focused house and the couple a focused heir claims through', () => {
    renderDialog(undefined, false, 'P2')
    // P2 owns ANN and EVE, so JEA (spouse, target) and OLD (spouse of EVE) are tied; VIC is P2's own.
    const nobles = [...document.querySelectorAll('[data-noble]')].map((el) =>
      el.getAttribute('data-noble'),
    )
    expect(nobles.sort()).toEqual(['ANN', 'EVE', 'JEA', 'OLD', 'VIC'])
  })

  it('changes the visible nobles with the focus select', () => {
    renderDialog()
    fireEvent.change(screen.getByLabelText('Focus'), { target: { value: 'P1' } })
    expect(document.querySelector('[data-noble="VIC"]')).toBeNull()
    fireEvent.change(screen.getByLabelText('Focus'), { target: { value: '' } })
    expect(document.querySelector('[data-noble="VIC"]')).toBeInTheDocument()
  })

  it('colours the rank seal with the house colour and outlines its digit', () => {
    renderDialog()
    const seal = document.querySelector('[data-noble="JEA"] svg g circle')!
    expect(seal).toHaveAttribute('fill', '#a84632')
    const digit = document.querySelector('[data-noble="JEA"] svg g text')!
    expect(digit).toHaveAttribute('paint-order', 'stroke')
    expect(digit).toHaveTextContent('1')
  })

  it('unfolds the alliance inset and lists the marriage to cancel and the one taking over', () => {
    renderDialog()
    const toggle = screen.getByRole('button', { name: 'Alliances' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Duc Jean × Duchesse Anne')).not.toBeInTheDocument()
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Duc Jean × Duchesse Anne')).toBeInTheDocument()
    expect(screen.getByText(/Takes over for Bob/)).toBeInTheDocument()
    expect(screen.getByText('Duc Jean × Dame Eve')).toBeInTheDocument()
    expect(screen.getByText(/Alice would have no head/)).toBeInTheDocument()
  })

  it('shows no watermark outside the simulation', () => {
    renderDialog(async () => {
      throw new Error('unused')
    })
    expect(screen.queryByTestId('lineage-watermark')).not.toBeInTheDocument()
  })

  describe('simulation', () => {
    const reading = (
      total: number,
      status: VictoryReading['status'],
    ): VictoryReading => ({
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

    it('applies the focus to the simulated state', async () => {
      renderDialog(async () => projection(true), false, 'P1')
      expect(document.querySelector('[data-noble="ANN"]')).toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      // The simulated state has no marriage left: only Alice's own nobles remain.
      await waitFor(() => expect(document.querySelector('[data-noble="ANN"]')).toBeNull())
      expect(document.querySelector('[data-noble="JEA"]')).toBeInTheDocument()
    })

    it('watermarks the tree as a hypothesis while simulating only', async () => {
      renderDialog(async () => projection(false))
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      expect(screen.getByTestId('lineage-watermark')).toHaveTextContent('Hypothesis')
      fireEvent.click(screen.getByRole('button', { name: 'Exit simulation' }))
      expect(screen.queryByTestId('lineage-watermark')).not.toBeInTheDocument()
    })

    it('kills a noble, updates the board and the status panel, then undoes it', async () => {
      const request = vi.fn(async (actions: SimulationAction[]) =>
        projection(actions.length > 0),
      )
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const card = screen
        .getByRole('region', { name: 'Alice' })
        .querySelector('[data-noble="JEA"]')!
      fireEvent.click(card)
      fireEvent.click(screen.getByRole('button', { name: 'Kill' }))

      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [{ type: 'kill', noble: 'JEA' }],
          expect.anything(),
        ),
      )
      expect(await screen.findByTestId('panel-status-P1')).toHaveTextContent(
        'Major victory',
      )
      expect(screen.getByTestId('panel-delta-P1')).toHaveTextContent('+1')
      expect(screen.getByTestId('panel-status-P2')).toHaveTextContent('Defeat')
      expect(screen.getByText('Duc Jean dies')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: 'Undo' }))
      await waitFor(() => expect(request).toHaveBeenLastCalledWith([], expect.anything()))
      expect(screen.queryByText('Duc Jean dies')).not.toBeInTheDocument()
    })

    it('lets the keyboard pick a noble, and gives its rank to assistive tech', async () => {
      const request = vi.fn(async () => projection(false))
      renderDialog(request)
      const jean = document.querySelector('[data-noble="JEA"]')!
      expect(jean).toHaveTextContent('Succession rank 1')
      // Without simulation, nothing is a button.
      expect(jean.querySelector('[role="button"]')).toBeNull()
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const button = document.querySelector('[data-noble="JEA"] [role="button"]')!
      expect(button).toHaveAttribute('tabindex', '0')
      fireEvent.keyDown(button, { key: 'Enter' })
      expect(screen.getByRole('button', { name: 'Kill' })).toBeInTheDocument()
      expect(button).toHaveAttribute('aria-pressed', 'true')
    })

    it('keeps the scroll where it is while an action changes the simulated tree', async () => {
      const request = vi.fn(async (actions: SimulationAction[]) =>
        projection(actions.length > 0),
      )
      renderDialog(request, false, 'P1')
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const scroller = document.querySelector('.overflow-auto') as HTMLElement
      scroller.scrollTop = 123
      fireEvent.click(document.querySelector('[data-noble="JEA"]')!)
      fireEvent.click(screen.getByRole('button', { name: 'Kill' }))
      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [{ type: 'kill', noble: 'JEA' }],
          expect.anything(),
        ),
      )
      await screen.findByText('Duc Jean dies')
      expect(scroller.scrollTop).toBe(123)
    })

    it('raises a fief by one rank and crowns a noble', async () => {
      const request = vi.fn(async () => projection(false))
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      fireEvent.click(
        screen
          .getByRole('region', { name: 'Alice' })
          .querySelector('[data-noble="JEA"]')!,
      )
      expect(screen.getByRole('button', { name: 'Raise ROC one rank' })).toBeDisabled()
      fireEvent.click(screen.getByRole('button', { name: 'Raise AAA one rank' }))
      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [{ type: 'rank', fief: 'AAA' }],
          expect.anything(),
        ),
      )
      fireEvent.click(
        screen.getByRole('region', { name: 'Bob' }).querySelector('[data-noble="ANN"]')!,
      )
      fireEvent.click(screen.getByRole('button', { name: 'Crown' }))
      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [
            { type: 'rank', fief: 'AAA' },
            { type: 'crown', noble: 'ANN' },
          ],
          expect.anything(),
        ),
      )
    })

    it('asks for a second click to marry two nobles', async () => {
      const request = vi.fn(async () => projection(false))
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      fireEvent.click(
        screen.getByRole('region', { name: 'Bob' }).querySelector('[data-noble="EVE"]')!,
      )
      fireEvent.click(screen.getByRole('button', { name: 'Marry to…' }))
      expect(screen.getByRole('status')).toHaveTextContent('Click the spouse of Dame Eve')
      fireEvent.click(
        screen
          .getByRole('region', { name: 'Alice' })
          .querySelector('[data-noble="JEA"]')!,
      )
      await waitFor(() =>
        expect(request).toHaveBeenLastCalledWith(
          [{ type: 'marry', noble: 'EVE', other: 'JEA' }],
          expect.anything(),
        ),
      )
    })

    it('drops an action the server refuses and says why', async () => {
      const request = vi.fn(
        async (actions: SimulationAction[]): Promise<VictorySimulation> => {
          if (actions.length > 0)
            throw new SimulationRejectedError('noble "JEA" is already married')
          return projection(false)
        },
      )
      renderDialog(request)
      fireEvent.click(screen.getByRole('button', { name: 'Simulate' }))
      const card = screen
        .getByRole('region', { name: 'Alice' })
        .querySelector('[data-noble="JEA"]')!
      fireEvent.click(card)
      fireEvent.click(screen.getByRole('button', { name: 'Kill' }))
      expect(await screen.findByRole('alert')).toHaveTextContent('already married')
      await waitFor(() =>
        expect(screen.queryByText('Duc Jean dies')).not.toBeInTheDocument(),
      )
      expect(
        within(screen.getByRole('button', { name: 'Reset' }).parentElement!).getByRole(
          'button',
          { name: 'Reset' },
        ),
      ).toBeDisabled()
    })
  })
})
