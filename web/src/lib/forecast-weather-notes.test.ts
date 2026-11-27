import { describe, expect, it } from 'vitest'

import { forecastWeatherNotes } from '@/lib/forecast-weather-notes'
import type { StateData } from '@/types'

const state = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [
    { id: 'VPR', owner: 'P1', resources: 0, army: null, infrastructures: [] },
    { id: 'BLS', owner: 'P2', resources: 0, army: null, infrastructures: [] },
  ],
  nobles: [],
  bishoprics: [
    { region: 'VPR', name: 'Valpierre', territories: ['VPR'] },
    { region: 'BLS', name: 'Blésoleil', territories: ['BLS'] },
  ],
  forecastEffects: [
    { kind: 'famine_canceled', region: 'VPR' },
    { kind: 'abundant_harvest', region: 'VPR', income: 3, rations: 4 },
    { kind: 'bad_weather', region: 'BLS', income: -2 },
  ],
} as StateData

describe('forecastWeatherNotes', () => {
  it('lists the effects of the regions the player holds, with their tone and amounts', () => {
    expect(forecastWeatherNotes(state, 'P1')).toEqual([
      {
        key: 'treasury.cause.famineCanceled',
        tone: 'neutral',
        region: 'VPR',
        income: 0,
        rations: 0,
      },
      {
        key: 'treasury.cause.abundantHarvest',
        tone: 'positive',
        region: 'VPR',
        income: 3,
        rations: 4,
      },
    ])
  })

  it('marks a calamity as negative', () => {
    const lost = {
      ...state,
      forecastEffects: [{ kind: 'famine', region: 'VPR', income: -1, rations: -3 }],
    } as StateData
    expect(forecastWeatherNotes(lost, 'P1')[0]).toMatchObject({
      tone: 'negative',
      income: -1,
      rations: -3,
    })
  })

  it('is empty without effects and in winter', () => {
    expect(forecastWeatherNotes({ ...state, forecastEffects: undefined }, 'P1')).toEqual(
      [],
    )
    expect(forecastWeatherNotes({ ...state, season: 'winter' }, 'P1')).toEqual([])
  })
})
