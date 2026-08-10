import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { OrdersPanel } from '@/components/OrdersPanel'
import type { StateData } from '@/types'

const state: StateData = {
  turn: 4,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [],
  nobles: [],
}

function renderOrdersPanel(season: StateData['season'], onOpenRules = vi.fn()) {
  return render(
    <OrdersPanel
      state={{ ...state, season }}
      player="P1"
      chainDrafts={{}}
      winterDraft=""
      submitted={false}
      submitting={false}
      onChainChange={vi.fn()}
      onWinterChange={vi.fn()}
      onSubmit={vi.fn()}
      onOpenRules={onOpenRules}
    />,
  )
}

describe('OrdersPanel seasonal presentation', () => {
  it('makes winter direct investments visually and textually distinct', () => {
    const { container } = renderOrdersPanel('winter')

    const heading = screen.getByRole('heading', { name: "Ordres d'hiver" })
    expect(heading.querySelector('svg')).toBeInTheDocument()
    expect(screen.getByText(/Investissements directs uniquement/)).toBeInTheDocument()
    expect(screen.getByText(/sans chaînes ni mouvements militaires/)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: "Soumettre les ordres d'hiver" }),
    ).toBeInTheDocument()
    expect(container.querySelector('section')).toHaveClass('bg-[#eaf3ff]/80')
  })

  it('keeps the ordinary command panel outside winter', () => {
    renderOrdersPanel('spring')

    expect(screen.getByRole('heading', { name: "Chaînes d'ordres" })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Soumettre' })).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: "Soumettre les ordres d'hiver" }),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByText(/Investissements directs uniquement/),
    ).not.toBeInTheDocument()
  })

  it('targets the winter rules section from the winter shortcut', () => {
    const onOpenRules = vi.fn()
    renderOrdersPanel('winter', onOpenRules)

    fireEvent.click(screen.getByRole('button', { name: 'Aide-mémoire des ordres' }))

    expect(onOpenRules).toHaveBeenCalledWith('winter-orders')
  })

  it('targets the action-order rules section outside winter', () => {
    const onOpenRules = vi.fn()
    renderOrdersPanel('spring', onOpenRules)

    fireEvent.click(screen.getByRole('button', { name: 'Aide-mémoire des ordres' }))

    expect(onOpenRules).toHaveBeenCalledWith('action-orders')
  })
})
