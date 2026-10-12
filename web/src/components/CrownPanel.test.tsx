import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { CrownPanel } from '@/components/CrownPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, StateData } from '@/types'

const noble = (code: string, name: string, owner: string, status: Noble['status'] = 'free'): Noble => ({
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
  nobles: [noble('KIN', 'Louis', 'P1', 'dungeon'), noble('QUE', 'Aliénor', 'P2')],
  king: 'KIN',
  queen: 'QUE',
  fiefs: [{ capital: 'FFF', title: 'barony', territories: ['FFF', 'GGG'], owner: 'P1', crown: true }],
}

const renderPanel = (s: StateData) =>
  render(
    <LanguageProvider initialLanguage="fr">
      <CrownPanel state={s} />
    </LanguageProvider>,
  )

describe('CrownPanel', () => {
  it('shows the king, queen, regent and crown fiefs', () => {
    const { container } = renderPanel(state)
    expect(screen.getByText(/Louis \(KIN\) · One · au cachot/)).toBeInTheDocument()
    expect(screen.getByText('Reine')).toBeInTheDocument()
    expect(container.querySelector('[data-crown-regent]')).not.toBeNull()
    expect(container.querySelector('[data-crown-fief="FFF"]')).not.toBeNull()
  })

  it('renders nothing without crown data', () => {
    const { container } = renderPanel({ ...state, king: undefined, queen: undefined, fiefs: [] })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('CrownPanel regency', () => {
  it('shows a dowager queen as regent of a vacant throne', () => {
    const { container } = renderPanel({ ...state, king: undefined, queenDowager: true, fiefs: [] })
    expect(screen.getByText('Reine douairière')).toBeInTheDocument()
    expect(container.querySelector('[data-crown-regent]')).not.toBeNull()
  })

  it('shows no regent while the king rules', () => {
    const free = { ...state, nobles: [noble('KIN', 'Louis', 'P1'), noble('QUE', 'Aliénor', 'P2')] }
    const { container } = renderPanel(free)
    expect(container.querySelector('[data-crown-regent]')).toBeNull()
  })
})
