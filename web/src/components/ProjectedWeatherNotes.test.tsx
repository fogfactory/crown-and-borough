import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ProjectedWeatherNotes } from '@/components/ProjectedWeatherNotes'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { StateData } from '@/types'

const state = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [{ id: 'VPR', owner: 'P1', resources: 0, army: null, infrastructures: [] }],
  nobles: [],
  bishoprics: [{ region: 'VPR', name: 'Valpierre', territories: ['VPR'] }],
  forecastEffects: [
    { kind: 'famine', region: 'VPR', income: -2, rations: -3 },
    { kind: 'abundant_harvest', region: 'VPR', income: 2, rations: 3 },
    { kind: 'bad_weather_canceled', region: 'VPR' },
  ],
} as StateData

describe('ProjectedWeatherNotes', () => {
  it('colors negative effects red, positive green and canceled ones black, with their totals', () => {
    render(
      <LanguageProvider>
        <ProjectedWeatherNotes state={state} playerId="P1" />
      </LanguageProvider>,
    )
    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(3)
    expect(items[0]).toHaveClass('text-vermilion-700')
    expect(items[0].textContent).toContain('−2')
    expect(items[0].textContent).toContain('−3')
    expect(items[1]).toHaveClass('text-success-700')
    expect(items[1].textContent).toContain('+2')
    expect(items[2]).toHaveClass('text-ink-900')
    expect(items[2].textContent).not.toMatch(/[+−]\d/)
  })
})
