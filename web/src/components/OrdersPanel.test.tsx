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
    renderOrdersPanel(
      'spring',
      vi.fn(),
      [
        {
          id: 'N1',
          code: 'DUN',
          name: 'Dungeon',
          owner: 'P1',
          location: 'ROS',
          status: 'dungeon',
        },
      ],
      'P BT ROS',
    )

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

    expect(
      screen.getByText(/Beau temps \(BT\)x2, Bonne récolte \(RA\)/),
    ).toBeInTheDocument()
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
    expect(
      screen.getByText(/Mauvaise récolte \(MR\) — Hiver dans BOI/),
    ).toBeInTheDocument()
  })

  it('warns about scheduled calamities in the winter deck summary', () => {
    const warningState: StateData = {
      ...state,
      season: 'winter',
      announcements: [{ kind: 'bad_weather', season: 'spring', region: 'ROS', year: 3 }],
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
    expect(
      screen.getByText(/Mauvais temps \(MT\) — Printemps dans ROS/),
    ).toBeInTheDocument()
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

describe('OrdersPanel noble deck (winter)', () => {
  const deckState: StateData = {
    ...state,
    season: 'winter',
    handLimit: 4,
    territories: [
      {
        id: 'ROS',
        owner: 'P1',
        resources: 0,
        army: { owner: 'P1', size: 2, chain: null },
        infrastructures: [{ type: 'castle', level: 1 }],
      },
      {
        id: 'BRU',
        owner: 'P1',
        resources: 0,
        army: null,
        infrastructures: [{ type: 'village', level: 1 }],
      },
    ],
    nobles: [
      {
        id: 'n1',
        code: 'HUG',
        name: 'Hugues',
        owner: 'P1',
        location: 'ROS',
        status: 'free',
      },
    ],
    nobleDeckSize: 5,
    nobleHand: [
      { id: 'c1', kind: 'noble', code: 'ALB', name: 'Albert', sex: 'male' },
      { id: 'c2', kind: 'dignity', code: 'BAS', dignity: 'bastard' },
    ],
  }

  function renderDeck(winterDraft: string, onWinterChange = vi.fn(), s = deckState) {
    render(
      <LanguageProvider initialLanguage="fr">
        <OrdersPanel
          state={s}
          player="P1"
          chainDrafts={{}}
          winterDraft={winterDraft}
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={onWinterChange}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )
    return onWinterChange
  }

  it('draws a card and shows the deck size', () => {
    const onWinterChange = renderDeck('')
    expect(screen.getByText('Deck : 5 carte(s) restante(s)')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Piocher une carte de noble/ }))
    expect(onWinterChange).toHaveBeenCalledWith('T N\n')
  })

  it('disables the draw when already ordered or the deck is empty', () => {
    renderDeck('T N\n')
    expect(
      screen.getByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeDisabled()
  })

  it('shows the combined hand counter and disables the draw when the hand is full', () => {
    renderDeck('', vi.fn(), {
      ...deckState,
      specialHand: ['fair_weather', 'fair_weather'],
    })
    expect(
      screen.getByText('Main : 4/4 cartes (spéciales : 2, nobles : 2)'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeDisabled()
    expect(screen.getByText(/Votre main est pleine/)).toBeInTheDocument()
  })

  it('keeps the draw enabled while the shared hand has room', () => {
    renderDeck('', vi.fn(), { ...deckState, specialHand: ['fair_weather'] })
    expect(
      screen.getByText('Main : 3/4 cartes (spéciales : 1, nobles : 2)'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeEnabled()
  })

  it('disables the draw on an empty deck', () => {
    renderDeck('', vi.fn(), { ...deckState, nobleDeckSize: 0 })
    expect(
      screen.getByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeDisabled()
  })

  it('plays a noble card only on controlled settlements with an army', () => {
    const onWinterChange = renderDeck('')
    fireEvent.click(screen.getByRole('button', { name: 'Albert (ALB, homme)' }))
    const dialog = screen.getByRole('dialog')
    expect(
      within(dialog)
        .getAllByRole('option')
        .map((o) => o.textContent),
    ).toEqual(['ROS'])
    fireEvent.click(within(dialog).getByRole('button', { name: "Ajouter l'ordre" }))
    expect(onWinterChange).toHaveBeenCalledWith('R N ALB ROS # recruter Albert sur ROS\n')
  })

  it('plays a dignity card on an own noble', () => {
    const onWinterChange = renderDeck('# note')
    fireEvent.click(screen.getByRole('button', { name: 'Dignité : Bâtard (BAS)' }))
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: "Ajouter l'ordre" }))
    expect(onWinterChange).toHaveBeenCalledWith('# note\nD N HUG BAS # Bâtard pour Hugues\n')
  })

  it('plays a claim card with an own heir on a noble married to one of ours', () => {
    const onWinterChange = renderDeck('', vi.fn(), {
      ...deckState,
      nobles: [
        ...deckState.nobles,
        {
          id: 'n2',
          code: 'ANN',
          name: 'Anne',
          owner: 'P2',
          location: 'BRU',
          status: 'free',
        },
        {
          id: 'n3',
          code: 'BOB',
          name: 'Bob',
          owner: 'P2',
          location: 'BRU',
          status: 'free',
        },
      ],
      marriages: [{ nobleA: 'HUG', nobleB: 'ANN', turn: 1 }],
      nobleHand: [{ id: 'c3', kind: 'claim', code: 'CLM' }],
    })
    fireEvent.click(screen.getByRole('button', { name: 'Prétention (CLM)' }))
    const dialog = screen.getByRole('dialog')
    expect(
      within(screen.getByLabelText('Noble visé'))
        .getAllByRole('option')
        .map((o) => o.textContent),
    ).toEqual(['ANN · Anne'])
    fireEvent.click(within(dialog).getByRole('button', { name: "Ajouter l'ordre" }))
    expect(onWinterChange).toHaveBeenCalledWith('C N HUG ANN # prétention sur Anne par Hugues\n')
  })

  it('offers the card orders outside winter, without draw nor discard', () => {
    renderDeck('', vi.fn(), { ...deckState, season: 'spring' })
    expect(
      screen.getByRole('button', { name: 'Albert (ALB, homme)' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeNull()
    expect(screen.queryByRole('button', { name: 'Défausser la carte ALB' })).toBeNull()
  })

  it('discards a noble or dignity card from the hand', () => {
    const onWinterChange = renderDeck('')
    fireEvent.click(screen.getByRole('button', { name: 'Défausser la carte ALB' }))
    expect(onWinterChange).toHaveBeenCalledWith('D C ALB # défausser Albert\n')
    fireEvent.click(screen.getByRole('button', { name: 'Défausser la carte BAS' }))
    expect(onWinterChange).toHaveBeenLastCalledWith('D C BAS # défausser BAS\n')
  })

  it('disables the discard of a card already used by a draft line', () => {
    renderDeck('D C ALB\nD N HUG BAS\n')
    expect(screen.getByRole('button', { name: 'Défausser la carte ALB' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Défausser la carte BAS' })).toBeDisabled()
  })

  it('lets a drafted discard free a slot for the draw', () => {
    renderDeck('D C ALB\n', vi.fn(), {
      ...deckState,
      specialHand: ['fair_weather', 'fair_weather'],
    })
    expect(
      screen.getByRole('button', { name: /Piocher une carte de noble/ }),
    ).toBeEnabled()
  })
})

describe('OrdersPanel winter elections', () => {
  function renderElections(onWinterChange = vi.fn(), winterDraft = '') {
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{
            ...state,
            season: 'winter',
            bishoprics: [{ region: 'R1', name: 'Ros', territories: ['AAA'] }],
            openElections: [
              {
                kind: 'bishop',
                region: 'R1',
                seat: 'AAA',
                voices: 3,
                voiceSources: [
                  { kind: 'seat', territory: 'AAA', votes: 2 },
                  { kind: 'territory', territory: 'BBB', votes: 1 },
                ],
                candidates: [{ code: 'HUG', name: 'Sieur Hugues de Ros' }],
              },
              { kind: 'pope', required: 2, voices: 0, voiceSources: [], candidates: [] },
            ],
          }}
          player="P1"
          chainDrafts={{}}
          winterDraft={winterDraft}
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={onWinterChange}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )
  }

  it('announces open elections with their rules and the timing note', () => {
    renderElections()
    expect(screen.getByText('Elections this winter')).toBeTruthy()
    expect(screen.getByText('Bishopric of Ros (seat AAA)')).toBeTruthy()
    expect(screen.getByText('Relative majority. Candidacy: K E NNN AAA. Vote: V E NNN AAA')).toBeTruthy()
    expect(screen.getByText('Your voices: 3')).toBeTruthy()
    expect(screen.getByText(/Absolute majority: 2 voices/)).toBeTruthy()
    expect(screen.getByText(/can only be used next winter/)).toBeTruthy()
  })

  it('adds a candidacy line when a candidate is clicked', () => {
    const onWinterChange = vi.fn()
    renderElections(onWinterChange)
    fireEvent.click(screen.getAllByRole('button', { name: 'Candidacy (K)' })[0])
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenCalledWith(
      'K E HUG AAA # candidacy Sieur Hugues de Ros\n',
    )
  })

  it('adds a commented vote line from the vote dialog', () => {
    const onWinterChange = vi.fn()
    renderElections(onWinterChange)
    fireEvent.click(screen.getAllByRole('button', { name: 'Vote (V)' })[0])
    fireEvent.change(screen.getByLabelText('Candidate code'), { target: { value: 'hug' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenCalledWith('V E HUG AAA # vote HUG\n')
  })

  it('disables a candidate already on the sheet', () => {
    renderElections(vi.fn(), 'K E HUG AAA # candidacy Sieur Hugues de Ros\n')
    const button = screen.getAllByRole('button', {
      name: 'Candidacy (K)',
    })[0] as HTMLButtonElement
    expect(button.disabled).toBe(true)
  })
})

describe('OrdersPanel title and card order dialogs', () => {
  const nobles: Noble[] = [
    { id: 'n1', code: 'POP', name: 'Pie', owner: 'P1', location: 'AAA', status: 'free' },
    { id: 'n2', code: 'LEO', name: 'Leon', owner: 'P2', location: 'BBB', status: 'free' },
    { id: 'n3', code: 'ABE', name: 'Abel', owner: 'P2', location: 'BBB', status: 'free' },
  ]

  const aids = {
    excommunicable: [],
    liftable: [],
    buyableCardinals: [],
    cardinalCost: 8,
    fiefSites: [],
    fiefCostPerTerritory: 2,
  }

  function renderWinter(
    extra: Partial<StateData>,
    onWinterChange = vi.fn(),
    season: StateData['season'] = 'winter',
    winterDraft = '',
  ) {
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{ ...state, season, nobles, ...extra }}
          player="P1"
          regions={[{ id: 'R1', name: 'Ros', seed: 'AAA', territories: ['AAA'] }]}
          chainDrafts={{}}
          winterDraft={winterDraft}
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={onWinterChange}
          onSpecialChange={vi.fn()}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )
    return onWinterChange
  }

  it('lets the pope excommunicate and lift through dialogs', () => {
    const onWinterChange = renderWinter({
      pope: 'POP',
      excommunicated: [{ noble: 'ABE', reason: 'papal', turn: 3 }],
      winterAids: { ...aids, excommunicable: [{ code: 'LEO' }], liftable: ['ABE'] },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Excommunicate (X E)' }))
    const options = within(screen.getByRole('dialog'))
      .getAllByRole('option')
      .map((o) => o.textContent)
    expect(options).toEqual(['LEO · Leon (P2)'])
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenCalledWith('X E LEO # excommunicate Leon\n')

    fireEvent.click(screen.getByRole('button', { name: 'Lift an excommunication (X L)' }))
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenLastCalledWith('X L ABE # lift excommunication of Abel\n')
  })

  it('offers no papal order to a player who is not the pope', () => {
    renderWinter({ pope: 'LEO', winterAids: aids })
    expect(screen.queryByRole('button', { name: 'Excommunicate (X E)' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Lift an excommunication (X L)' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Buy a cardinal (N C)' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Found a fief (T F)' })).toBeNull()
  })

  it('lets a bishop buy a cardinal', () => {
    const onWinterChange = renderWinter({
      bishoprics: [{ region: 'R1', name: 'Ros', territories: ['AAA'], bishop: 'POP' }],
      winterAids: { ...aids, buyableCardinals: ['POP'] },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Buy a cardinal (N C)' }))
    expect(screen.getByText('Cost: 8 R')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenCalledWith('N C POP # buy cardinal Pie\n')
  })

  it('founds a fief from a connected group and prices it', () => {
    const onWinterChange = renderWinter({
      winterAids: {
        ...aids,
        fiefSites: [
          {
            territories: ['AAA', 'BBB', 'CCC', 'DDD'],
            castles: ['AAA'],
            edges: [['AAA', 'BBB'], ['BBB', 'CCC'], ['CCC', 'DDD']],
          },
        ],
      },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Found a fief (T F)' }))
    const dialog = screen.getByRole('dialog')
    const confirm = within(dialog).getByRole('button', { name: 'Add the order' })
    expect(confirm).toBeDisabled()
    fireEvent.click(within(dialog).getByLabelText('DDD'))
    fireEvent.click(within(dialog).getByLabelText('BBB'))
    expect(within(dialog).getByText(/connected to the capital/)).toBeTruthy()
    fireEvent.click(within(dialog).getByLabelText('CCC'))
    expect(within(dialog).getByText('County · 4 territories · cost 8 R')).toBeTruthy()
    fireEvent.click(confirm)
    expect(onWinterChange).toHaveBeenCalledWith(
      expect.stringMatching(/^T F POP AAA BBB CCC DDD # found fief AAA for Pie\n$/),
    )
  })

  it('hides the buy-cardinal order once it is on the sheet', () => {
    renderWinter({
      winterAids: { ...aids, buyableCardinals: ['POP'] },
      // the sheet below already carries the order
    }, vi.fn(), 'winter', 'N C POP # buy cardinal Pie\n')
    expect(screen.queryByRole('button', { name: 'Buy a cardinal (N C)' })).toBeNull()
  })

  it('plays a special card on a region and discards it in winter', () => {
    const onWinterChange = renderWinter({ specialHand: ['fair_weather'] })
    fireEvent.click(screen.getByRole('button', { name: /Discard Fair weather|Discard Beau temps/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onWinterChange).toHaveBeenCalledWith(expect.stringMatching(/^D C BT # discard /))
  })

  it('shows the calamity a bonus card cancels in the chosen region', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{
            ...state,
            season: 'spring',
            nobles,
            specialHand: ['fair_weather'],
            announcements: [{ kind: 'bad_weather', season: 'summer', region: 'AAA', year: 1001 }],
          }}
          player="P1"
          regions={[
            { id: 'R0', name: 'Quiet', seed: 'ZZZ', territories: ['ZZZ'] },
            { id: 'R1', name: 'Ros', seed: 'AAA', territories: ['AAA'] },
          ]}
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
    fireEvent.click(screen.getByRole('button', { name: /^Play .*\(P\)$/ }))
    const dialog = screen.getByRole('dialog')
    const options = within(dialog).getAllByRole('option').map((o) => o.textContent)
    expect(options[0]).toContain('AAA')
    expect(options[0]).toContain('cancels a calamity')
    expect(within(dialog).getByText(/^Cancels: Bad weather/)).toBeTruthy()
  })

  it('plays a special card on a region outside winter', () => {
    const onSpecialChange = vi.fn()
    render(
      <LanguageProvider initialLanguage="en">
        <OrdersPanel
          state={{ ...state, season: 'spring', nobles, specialHand: ['fair_weather'] }}
          player="P1"
          regions={[{ id: 'R1', name: 'Ros', seed: 'AAA', territories: ['AAA'] }]}
          chainDrafts={{}}
          winterDraft=""
          specialDraft=""
          submitted={false}
          submitting={false}
          error={null}
          onChainChange={vi.fn()}
          onWinterChange={vi.fn()}
          onSpecialChange={onSpecialChange}
          onSubmit={vi.fn()}
          onOpenRules={vi.fn()}
        />
      </LanguageProvider>,
    )
    fireEvent.click(screen.getByRole('button', { name: /^Play .*\(P\)$/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Add the order' }))
    expect(onSpecialChange).toHaveBeenCalledWith(expect.stringMatching(/^P BT AAA # .* on AAA\n$/))
  })
})
