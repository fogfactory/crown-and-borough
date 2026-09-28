import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { Scoreboard } from '@/components/Scoreboard'

describe('Scoreboard', () => {
  it('shows each player total and category breakdown', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <Scoreboard
          players={[{ id: 'P1', name: 'Alice', color: '#a84632' }]}
          scores={{
            P1: {
              territories: 2,
              villages: 4,
              mills: 1,
              castles: 5,
              nobles: 2,
              troops: 3,
              resources: 7,
              fiefs: 2,
              total: 26,
            },
          }}
        />
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: 'Scores' })).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('26')).toBeInTheDocument()
    expect(screen.getByText('Territories')).toBeInTheDocument()
    expect(screen.getByText('Resources')).toBeInTheDocument()
    expect(screen.getByText('Fiefs')).toBeInTheDocument()
  })

  it('defaults fiefs to 0 for a score snapshot recorded before issue #194', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <Scoreboard
          players={[{ id: 'P1', name: 'Alice', color: '#a84632' }]}
          scores={{
            P1: {
              territories: 2,
              villages: 0,
              mills: 0,
              castles: 0,
              nobles: 0,
              troops: 0,
              resources: 0,
              total: 2,
            },
          }}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Fiefs')).toBeInTheDocument()
  })
})
