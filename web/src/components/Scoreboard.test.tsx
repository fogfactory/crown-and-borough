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
})
