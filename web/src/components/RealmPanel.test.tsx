import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { RealmPanel } from '@/components/RealmPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, StateData, Territory } from '@/types'

const noble = (
  code: string,
  name: string,
  owner: string,
  status: Noble['status'] = 'free',
): Noble => ({
  id: code,
  code,
  name,
  owner,
  location: 'AAA',
  status,
})

const state: StateData = {
  turn: 4,
  season: 'spring',
  players: [
    { id: 'P1', name: 'One', color: '#a84632' },
    { id: 'P2', name: 'Two', color: '#315a75' },
  ],
  territories: [],
  nobles: [
    noble('KIN', 'Louis', 'P1', 'dungeon'),
    noble('QUE', 'Aliénor', 'P2'),
    noble('PAP', 'Innocent', 'P1'),
    noble('BIS', 'Anselme', 'P2'),
  ],
  king: 'KIN',
  queen: 'QUE',
  fiefs: [
    {
      capital: 'FFF',
      title: 'barony',
      territories: ['FFF', 'GGG'],
      owner: 'P1',
      crown: true,
    },
  ],
  pope: 'PAP',
  cardinals: [],
  bishoprics: [
    { region: 'R1', name: 'Alpilles', territories: [], bishop: 'BIS' },
    { region: 'R2', name: 'Brisecote', territories: [] },
  ],
}

const territories = [{ id: 'FFF', name: 'Rochevent' }] as Territory[]

const renderPanel = (s: StateData, onOpenLineage?: () => void) =>
  render(
    <LanguageProvider initialLanguage="fr">
      <RealmPanel state={s} territories={territories} onOpenLineage={onOpenLineage} />
    </LanguageProvider>,
  )

describe('RealmPanel', () => {
  it('shows the crown: king, queen, regent and crown fiefs in full', () => {
    const { container } = renderPanel(state)

    expect(screen.getByRole('heading', { name: 'Couronne' })).toBeInTheDocument()
    expect(screen.getByText('Louis')).toBeInTheDocument()
    expect(screen.getByText('(KIN)')).toBeInTheDocument()
    expect(screen.getByText('· au cachot')).toBeInTheDocument()
    expect(screen.getByText('Reine')).toBeInTheDocument()
    expect(container.querySelector('[data-crown-regent]')).not.toBeNull()
    expect(container.querySelector('[data-crown-fief="FFF"]')).toHaveTextContent(
      'Baronnie de Rochevent (FFF) · 2 territoires',
    )
  })

  it('writes each holder with the tincture of its player', () => {
    const { container } = renderPanel(state)

    const pastille = container.querySelector('[data-owner-pastille="P1"]')
    expect(pastille).toHaveAttribute('title', 'One')
    expect(pastille).toHaveStyle({ backgroundColor: '#a84632' })
  })

  it('shows the Church: pope, bishops and vacant seats', () => {
    const { container } = renderPanel(state)

    expect(screen.getByRole('heading', { name: 'Église' })).toBeInTheDocument()
    expect(screen.getByText('Innocent')).toBeInTheDocument()
    expect(container.querySelector('[data-bishopric="R1"]')).toHaveTextContent(
      'Alpilles : Anselme (BIS)',
    )
    expect(container.querySelector('[data-bishopric="R2"]')).toHaveTextContent('vacant')
  })

  it('shows a dowager queen as regent of a vacant throne', () => {
    const { container } = renderPanel({
      ...state,
      king: undefined,
      queenDowager: true,
      fiefs: [],
    })

    expect(screen.getByText('Reine douairière')).toBeInTheDocument()
    expect(container.querySelector('[data-crown-regent]')).not.toBeNull()
  })

  it('shows no regent while the king rules', () => {
    const { container } = renderPanel({
      ...state,
      nobles: state.nobles.map((item) => ({ ...item, status: 'free' as const })),
    })

    expect(container.querySelector('[data-crown-regent]')).toBeNull()
  })

  it('says so when the kingdom has neither crown nor church', () => {
    const { container } = renderPanel({
      ...state,
      king: undefined,
      queen: undefined,
      fiefs: [],
      pope: undefined,
      bishoprics: [],
    })

    expect(container.querySelector('[data-realm-crown]')).toBeNull()
    expect(container.querySelector('[data-realm-church]')).toBeNull()
    expect(screen.getByText(/ni couronne ni Église/)).toBeInTheDocument()
  })

  it('opens the lineage from its footer button', () => {
    const onOpenLineage = vi.fn()
    renderPanel(state, onOpenLineage)

    fireEvent.click(
      screen.getByRole('button', { name: 'Ouvrir la lignée et les alliances' }),
    )
    expect(onOpenLineage).toHaveBeenCalledOnce()
  })
})
