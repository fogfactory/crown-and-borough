import { describe, expect, it } from 'vitest'

import { translate, type Translate } from '@/i18n/messages'
import { formatCardHand } from '@/lib/card-hand'

const french: Translate = (key, values) => translate('fr', key, values)
const english: Translate = (key, values) => translate('en', key, values)

describe('formatCardHand', () => {
  it('formats an empty hand with the localized empty label', () => {
    expect(formatCardHand([], french)).toBe('Vide')
    expect(formatCardHand([], english)).toBe('Empty')
  })

  it('uses localized short labels and aggregates duplicates', () => {
    expect(
      formatCardHand(['revolt', 'fair_weather', 'fair_weather', 'abundant_harvest'], french),
    ).toBe('Beau temps (BT)x2, Bonne récolte (RA), Révolte (RE)')
    expect(formatCardHand(['abundant_harvest', 'fair_weather', 'abundant_harvest'], english)).toBe(
      'Fair weather (FW), Abundant harvest (AH)x2',
    )
  })

  it('keeps a canonical order independent of draw order', () => {
    expect(formatCardHand(['revolt', 'abundant_harvest', 'fair_weather'], french)).toBe(
      'Beau temps (BT), Bonne récolte (RA), Révolte (RE)',
    )
  })
})
