import { describe, expect, it } from 'vitest'

import {
  HERALDIC_COLORS,
  REGION_PATTERNS,
  regionStyle,
} from '@/lib/region-color'

describe('regionStyle', () => {
  it('uses the heraldic palette without motifs for six regions', () => {
    expect(regionStyle(0, 6)).toEqual({ fill: HERALDIC_COLORS[0], pattern: null })
    expect(regionStyle(5, 6)).toEqual({ fill: HERALDIC_COLORS[5], pattern: null })
  })

  it('adds a deterministic motif after the base palette', () => {
    expect(regionStyle(6, 7)).toEqual({ fill: HERALDIC_COLORS[0], pattern: 'diagonal' })
    expect(regionStyle(12, 13)).toEqual({ fill: HERALDIC_COLORS[0], pattern: 'vertical' })
    expect(REGION_PATTERNS).toContain(regionStyle(17, 18).pattern)
  })
})
