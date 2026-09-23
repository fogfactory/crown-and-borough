import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { OrdersPanel } from '@/components/OrdersPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, OrdersPreview, StateData } from '@/types'

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
  specialDraft = '',
) {
  return render(
    <LanguageProvider initialLanguage="fr">
      <OrdersPanel
        state={{ ...state, season, nobles }}
        player="P1"
        chainDrafts={{}}
         winterDraft=""
         specialDraft={specialDraft}
         submitted={false}
        submitting={false}
        error={null}
        onChainChange={vi.fn()}
         onWinterChange={vi.fn()}
         onSpecialChange={vi.fn()}
         onSubmit={vi.fn()}
        onOpenRules={onOpenRules}
      />
    </LanguageProvider>,
  )
}

function renderWithPreview(
  season: StateData['season'],
  preview: OrdersPreview,
  language: 'fr' | 'en' = 'fr',
) {
  return render(
    <LanguageProvider initialLanguage={language}>
      <OrdersPanel
        state={{ ...state, season }}
        player="P1"
        chainDrafts={{}}
        winterDraft=""
        preview={preview}
        specialDraft=""
        submitted={false}
        submitting={false}
        error={null}
        onChainChange={vi.fn()}
        onWinterChange={vi.fn()}
        onSpecialChange={vi.fn()}
        onSubmit={vi.fn()}
        onOpenRules={vi.fn()}
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
    expect(screen.queryByLabelText('Ordres de cartes spéciales')).not.toBeInTheDocument()
    expect(screen.getByPlaceholderText(/D C BT/)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: "Soumettre les ordres d'hiver" }),
    ).toBeInTheDocument()
    expect(container.querySelector('section')).toHaveClass('bg-[#eaf3ff]/80')
  })

  it('shows the server winter cost against the controlled stock', () => {
    renderWithPreview('winter', {
      errors: [],
      chains: [],
      winter: [],
      winterCost: { spent: 21, available: 25 },
    })

    const estimate = screen.getByRole('status')
    expect(estimate).toHaveTextContent('Coût estimé : 21 / 25 ressources')
    expect(estimate).toHaveClass('text-[#376341]')
  })

  it('marks the cost when the sheet exceeds the available stock', () => {
    renderWithPreview(
      'winter',
      { errors: [], chains: [], winter: [], winterCost: { spent: 26, available: 25 } },
      'en',
    )

    const estimate = screen.getByRole('status')
    expect(estimate).toHaveTextContent('Estimated cost: 26 / 25 resources')
    expect(estimate).toHaveClass('text-[#8d321e]')
  })

  it('lists the server messages of malformed winter lines', () => {
    renderWithPreview('winter', {
      errors: [],
      chains: [],
      winter: [
        { line: 1, status: 'invalid', message: 'Ordre d’hiver inconnu' },
        { line: 2, status: 'applied', type: 'recruit_troop', territory: 'ROS', cost: 1 },
        { line: 3, status: 'invalid', message: 'Code de territoire inconnu : ZZZ' },
      ],
      winterCost: { spent: 1, available: 25 },
    })

    const errors = screen.getByRole('alert', {
      name: "Erreurs de syntaxe des ordres d'hiver",
    })
    expect(within(errors).getAllByRole('listitem')).toHaveLength(2)
    expect(errors).toHaveTextContent(/Ligne 1 .*Ordre d’hiver inconnu/)
    expect(errors).toHaveTextContent(/Ligne 3 .*ZZZ/)
    expect(screen.getByText('Coût estimé : 1 / 25 ressources')).toBeInTheDocument()
  })

  it('lists refused winter lines and resource warnings with line numbers', () => {
    renderWithPreview('winter', {
      errors: [],
      chains: [],
      winter: [
        {
          line: 1,
          status: 'rejected',
          type: 'recruit_troop',
          territory: 'BRU',
          reason: 'troop_requires_adjacent_noble',
        },
        {
          line: 2,
          status: 'rejected',
          type: 'build',
          territory: 'BRU',
          reason: 'insufficient_resources',
        },
      ],
      winterCost: { spent: 0, available: 0 },
    })

    const diagnostics = screen.getByRole('status', {
      name: "Diagnostics des ordres d'hiver",
    })
    expect(within(diagnostics).getAllByRole('listitem')).toHaveLength(2)
    expect(diagnostics).toHaveTextContent(
      /Ligne 1 .*Un noble libre doit être sur le territoire ou adjacent/,
    )
    expect(diagnostics).toHaveTextContent(/Ligne 2 .*Ressources insuffisantes/)
  })

  it('shows chain errors found by the server while the player types', () => {
    renderWithPreview('spring', {
      errors: [
        {
          noble: 'HUG',
          line: 2,
          code: 'not_adjacent',
          message: 'BRU n’est pas adjacent',
        },
        { line: 1, code: 'winter_out_of_season', message: 'hors saison' },
      ],
      chains: [],
      winter: [],
    })

    const errors = screen.getByRole('alert', {
      name: "Erreurs dans les chaînes d'ordres",
    })
    expect(within(errors).getAllByRole('listitem')).toHaveLength(1)
    expect(errors).toHaveTextContent(/Ligne 2 .*BRU n’est pas adjacent/)
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

    expect(screen.getByLabelText('Chaîne de HOS')).not.toBeDisabled()
    expect(screen.getByLabelText('Chaîne de DUN')).toBeDisabled()
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
    expect(screen.getByLabelText('Ordres de cartes spéciales')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Soumettre' })).toBeDisabled()
  })

  it('allows a deck-only submission without an emitting noble', () => {
    renderOrdersPanel('spring', vi.fn(), [
      {
        id: 'N1',
        code: 'DUN',
        name: 'Dungeon',
        owner: 'P1',
        location: 'ROS',
        status: 'dungeon',
      },
    ], 'P BT ROS')

    expect(screen.getByRole('button', { name: 'Soumettre' })).not.toBeDisabled()
  })

  it('shows short card labels and aggregates duplicate cards', () => {
    const handState: StateData = {
      ...state,
      season: 'spring',
      specialHand: ['fair_weather', 'abundant_harvest', 'fair_weather'],
    }
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={handState}
          player="P1"
          chainDrafts={{}}
          winterDraft=""
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText(/Beau temps \(BT\)x2, Bonne récolte \(RA\)/)).toBeInTheDocument()
  })

  it('warns about scheduled calamities in the deck panel', () => {
    const warningState: StateData = {
      ...state,
      season: 'spring',
      announcements: [
        { kind: 'plague', season: 'summer', region: 'ROS', year: 2 },
        { kind: 'famine', season: 'winter', region: 'BOI', year: 2 },
      ],
    }
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={warningState}
          player="P1"
          chainDrafts={{}}
          winterDraft=""
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Calamités à venir')).toBeInTheDocument()
    expect(screen.getByText(/Peste \(PE\) — Été dans ROS/)).toBeInTheDocument()
    expect(screen.getByText(/Mauvaise récolte \(MR\) — Hiver dans BOI/)).toBeInTheDocument()
  })

  it('warns about scheduled calamities in the winter deck summary', () => {
    const warningState: StateData = {
      ...state,
      season: 'winter',
      announcements: [
        { kind: 'bad_weather', season: 'spring', region: 'ROS', year: 3 },
      ],
    }
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={warningState}
          player="P1"
          chainDrafts={{}}
          winterDraft=""
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Calamités à venir')).toBeInTheDocument()
    expect(screen.getByText(/Mauvais temps \(MT\) — Printemps dans ROS/)).toBeInTheDocument()
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
          specialDraft=""
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={{ winter: true }}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
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
          specialDraft=""
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={{ chains: { GUI: true } }}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
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
          specialDraft=""
          submitted={true}
          submitting={false}
          error={null}
          draftDiffers={undefined}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={vi.fn()}
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
