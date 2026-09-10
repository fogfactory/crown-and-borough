import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { OrdersPanel } from '@/components/OrdersPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, StateData } from '@/types'

const state: StateData = {
  turn: 4,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [],
  nobles: [],
}

function renderOrdersPanel(
  season: StateData['season'],
  onOpenRules = vi.fn(),
  nobles: Noble[] = state.nobles,
) {
  return render(
    <LanguageProvider initialLanguage="fr">
      <OrdersPanel
        state={{ ...state, season, nobles }}
        player="P1"
        chainDrafts={{}}
        winterDraft=""
        submitted={false}
        submitting={false}
        error={null}
        onChainChange={vi.fn()}
        onWinterChange={vi.fn()}
        onSubmit={vi.fn()}
        onOpenRules={onOpenRules}
      />
    </LanguageProvider>,
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

  it('shows the live winter cost estimate against controlled stock', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={{
            ...state,
            season: 'winter',
            territories: [
              {
                id: 'ROS',
                owner: 'P1',
                resources: 25,
                army: null,
                infrastructures: [{ type: 'castle', level: 1 }],
              },
              {
                id: 'XXX',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
              {
                id: 'YYY',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
              {
                id: 'ZZZ',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
            ],
          }}
          player="P1"
          chainDrafts={{}}
          winterDraft={'R T XXX\nR N XXX\nC C YYY\nC M ZZZ\nC M ZZZ'}
          winterCosts={{
            castle: 10,
            millLevels: [3, 5, 7],
            troop: 1,
            noble: 2,
            supplyDepot: 3,
            liberation: 0,
          }}
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    const estimate = screen.getByRole('status')
    expect(estimate).toHaveTextContent('Coût estimé : 21 / 25 ressources')
    expect(estimate).toHaveClass('text-[#376341]')
  })

  it('marks the estimate when the draft exceeds available stock', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{
            ...state,
            season: 'winter',
            territories: [
              {
                id: 'ROS',
                owner: 'P1',
                resources: 25,
                army: null,
                infrastructures: [{ type: 'castle', level: 1 }],
              },
              {
                id: 'XXX',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
              {
                id: 'YYY',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
            ],
          }}
          player="P1"
          chainDrafts={{}}
          winterDraft="G XXX YYY 26"
          winterCosts={{
            castle: 10,
            millLevels: [3, 5, 7],
            troop: 1,
            noble: 2,
            supplyDepot: 3,
            liberation: 0,
          }}
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    const estimate = screen.getByRole('status')
    expect(estimate).toHaveTextContent('Estimated cost: 26 / 25 resources')
    expect(estimate).toHaveClass('text-[#8d321e]')
  })

  it('shows live syntax errors without charging invalid lines', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={{
            ...state,
            season: 'winter',
            territories: [
              {
                id: 'ROS',
                owner: 'P1',
                resources: 25,
                army: null,
                infrastructures: [{ type: 'castle', level: 1 }],
              },
              {
                id: 'ZZZ',
                owner: 'P1',
                resources: 0,
                army: null,
                infrastructures: [],
              },
            ],
          }}
          map={{
            territories: [
              {
                id: 'ROS',
                name: 'Rosemont',
                terrain: 'plain',
                village: false,
                points: [],
                adjacencies: [],
                impassable: [],
              },
            ],
          }}
          player="P1"
          chainDrafts={{}}
          winterDraft={'R X ROS\nR T ROS\nC M ZZZ'}
          winterCosts={{
            castle: 10,
            millLevels: [3, 5, 7],
            troop: 1,
            noble: 2,
            supplyDepot: 3,
            liberation: 0,
          }}
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    const errors = screen.getByRole('alert', {
      name: "Erreurs de syntaxe des ordres d'hiver",
    })
    expect(within(errors).getAllByRole('listitem')).toHaveLength(2)
    expect(errors).toHaveTextContent(/Ligne 1 .*Ordre d’hiver inconnu/)
    expect(errors).toHaveTextContent(/Ligne 3 .*code de territoire.*ZZZ/)
    expect(screen.getByRole('status')).toHaveTextContent(
      'Coût estimé : 1 / 25 ressources',
    )
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

  it('allows hostage chains but blocks dungeon chains', () => {
    renderOrdersPanel('spring', vi.fn(), [
      {
        id: 'N1',
        code: 'HOS',
        name: 'Hostage',
        owner: 'P1',
        location: 'ROS',
        status: 'hostage',
      },
      {
        id: 'N2',
        code: 'DUN',
        name: 'Dungeon',
        owner: 'P1',
        location: 'ROS',
        status: 'dungeon',
      },
    ])

    const textareas = screen.getAllByRole('textbox')
    expect(textareas[0]).not.toBeDisabled()
    expect(textareas[1]).toBeDisabled()
    expect(screen.getByText('Otage')).toBeInTheDocument()
    expect(screen.getByText('Donjon')).toBeInTheDocument()
  })

  it('does not require action orders when every noble is in a dungeon', () => {
    renderOrdersPanel('spring', vi.fn(), [
      {
        id: 'N1',
        code: 'DUN',
        name: 'Dungeon',
        owner: 'P1',
        location: 'ROS',
        status: 'dungeon',
      },
    ])

    expect(screen.getByText(/Aucun noble apte à émettre/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Soumettre' })).toBeDisabled()
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

  it('shows divergence note and restores winter orders', () => {
    const onRestore = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={{ ...state, season: 'winter' }}
          player="P1"
          chainDrafts={{}}
          winterDraft="R T ROS"
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={{ winter: true }}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
          onRestoreFromServer={onRestore}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Brouillon différent du serveur')).toBeInTheDocument()
    const restoreBtn = screen.getByRole('button', { name: 'Restaurer depuis le serveur' })
    expect(restoreBtn).toBeInTheDocument()
    fireEvent.click(restoreBtn)
    expect(onRestore).toHaveBeenCalledWith('winter')
  })

  it('shows divergence note and restores chain orders in english', () => {
    const onRestore = vi.fn()
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{
            ...state,
            season: 'spring',
            nobles: [
              {
                id: 'N1',
                code: 'GUI',
                name: 'Guillaume',
                owner: 'P1',
                location: 'ROS',
                status: 'free',
              },
            ],
          }}
          player="P1"
          chainDrafts={{ GUI: 'ROS A BT' }}
          winterDraft=""
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={{ chains: { GUI: true } }}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
          onRestoreFromServer={onRestore}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Local draft differs from server')).toBeInTheDocument()
    const restoreBtn = screen.getByRole('button', { name: 'Restore from server' })
    expect(restoreBtn).toBeInTheDocument()
    fireEvent.click(restoreBtn)
    expect(onRestore).toHaveBeenCalledWith('GUI')
  })

  it('does not show divergence note when there is no divergence', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={{
            ...state,
            season: 'spring',
            nobles: [
              {
                id: 'N1',
                code: 'GUI',
                name: 'Guillaume',
                owner: 'P1',
                location: 'ROS',
                status: 'free',
              },
            ],
          }}
          player="P1"
          chainDrafts={{ GUI: 'ROS A BT' }}
          winterDraft=""
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={undefined}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    expect(screen.queryByText('Brouillon différent du serveur')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Restaurer depuis le serveur' }),
    ).not.toBeInTheDocument()
  })
})
