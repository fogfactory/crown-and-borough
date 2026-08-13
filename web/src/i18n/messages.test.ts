import { describe, expect, it } from 'vitest'

import { formatOrderLabel } from '@/lib/order-label'
import { translate } from '@/i18n/messages'

describe('i18n messages', () => {
  it('provides season labels in both languages', () => {
    expect(translate('en', 'season.spring')).toBe('Spring')
    expect(translate('fr', 'season.spring')).toBe('Printemps')
    expect(translate('en', 'season.winter')).toBe('Winter')
    expect(translate('fr', 'season.winter')).toBe('Hiver')
  })

  it('keeps memorable order symbols identical in both languages', () => {
    const order = {
      type: 'support' as const,
      position: 'ROS',
      targets: ['BRI', 'CHA'],
      liaison: 'loop' as const,
    }

    expect(formatOrderLabel(order)).toBe('(ROS S BRI - CHA)')
    expect(formatOrderLabel(order)).toBe('(ROS S BRI - CHA)')
  })
})
