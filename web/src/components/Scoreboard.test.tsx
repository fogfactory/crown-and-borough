import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { Scoreboard } from '@/components/Scoreboard'

describe('Scoreboard', () => {
  it('shows each player total and title count', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <Scoreboard
          players={[{ id: 'P1', name: 'Alice', color: '#a84632' }]}
          scores={{
            P1: {
              titles: 3,
              total: 3,
            },
          }}
        />
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: 'Scores' })).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Titles')).toBeInTheDocument()
    expect(screen.getAllByText('3')).toHaveLength(2)
  })

  it('defaults titles to 0 for a score snapshot recorded before issue #251', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <Scoreboard
          players={[{ id: 'P1', name: 'Alice', color: '#a84632' }]}
          scores={{
            P1: {
              total: 0,
            },
          }}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Titles')).toBeInTheDocument()
    expect(screen.getAllByText('0')).toHaveLength(2)
  })

  it('shows the victory goal and mode of each player', () => {
    render(
      <LanguageProvider initialLanguage="en">
        <Scoreboard
          players={[
            { id: 'P1', name: 'Alice', color: '#a84632' },
            { id: 'P2', name: 'Bob', color: '#325ca8' },
          ]}
          scores={{ P1: { titles: 1, total: 1 }, P2: { titles: 0, total: 0 } }}
          victory={{
            soloThreshold: 4,
            allianceThreshold: 6,
            players: {
              P1: { mode: 'solo', required: 4 },
              P2: { mode: 'alliance', required: 6, partner: 'P1' },
            },
          }}
        />
      </LanguageProvider>,
    )

    expect(screen.getByText('Goal: 4 titles')).toBeInTheDocument()
    expect(screen.getByTestId('victory-mode-P1')).toHaveTextContent('Solo')
    expect(screen.getByText('Goal: 6 titles combined')).toBeInTheDocument()
    expect(screen.getByTestId('victory-mode-P2')).toHaveTextContent('Alliance with Alice')
  })
})
