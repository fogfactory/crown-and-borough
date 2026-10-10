import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ReligiousHierarchyPanel } from '@/components/ReligiousHierarchyPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, StateData } from '@/types'

const noble = (code: string, name: string, owner: string): Noble => ({
  id: code,
  code,
  name,
  owner,
  location: 'AAA',
  status: 'free',
})

const state: StateData = {
  turn: 4,
  season: 'spring',
  players: [
    { id: 'P1', name: 'One', color: '#a84632' },
    { id: 'P2', name: 'Two', color: '#315a75' },
  ],
  territories: [],
  nobles: [noble('PAP', 'Innocent', 'P1'), noble('BIS', 'Anselme', 'P2')],
  pope: 'PAP',
  cardinals: [],
  bishoprics: [
    { region: 'R1', name: 'Alpilles', territories: [], bishop: 'BIS' },
    { region: 'R2', name: 'Brisecote', territories: [] },
  ],
}

describe('ReligiousHierarchyPanel', () => {
  it('lists the pope, bishops with their owner, and vacant seats', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <ReligiousHierarchyPanel state={state} />
      </LanguageProvider>,
    )
    expect(screen.getByText(/Innocent \(PAP\) · One/)).toBeInTheDocument()
    expect(screen.getByText(/Anselme \(BIS\) · Two/)).toBeInTheDocument()
    expect(screen.getAllByText('vacant').length).toBeGreaterThanOrEqual(2)
  })

  it('renders nothing without any religious data', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <ReligiousHierarchyPanel state={{ ...state, pope: undefined, bishoprics: [] }} />
      </LanguageProvider>,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
