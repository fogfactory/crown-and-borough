import { describe, expect, it } from 'vitest'

import { buildWinterIntentions } from '@/lib/winter-overlay'
import type { WinterLinePreview } from '@/types'

const draft = [
  'C M ROS',
  'R T BRU',
  'NOT A LINE',
  'C C ROS',
  'G ROS ATL 3',
  'D C GR',
].join('\n')

const lines: WinterLinePreview[] = [
  {
    line: 1,
    status: 'applied',
    type: 'build',
    territory: 'ROS',
    infrastructure: 'mill',
    level: 2,
    cost: 5,
  },
  {
    line: 2,
    status: 'rejected',
    type: 'recruit_troop',
    territory: 'BRU',
    reason: 'insufficient_resources',
  },
  { line: 3, status: 'invalid', message: 'Ligne mal formée' },
  {
    line: 4,
    status: 'rejected',
    type: 'build',
    territory: 'ROS',
    infrastructure: 'castle',
    reason: 'structure_present',
  },
  {
    line: 5,
    status: 'applied',
    type: 'transfer',
    territory: 'ROS',
    source: 'ROS',
    target: 'ATL',
    amount: 3,
    cost: 3,
  },
  { line: 6, status: 'discard' },
]

describe('buildWinterIntentions', () => {
  const intentions = buildWinterIntentions(lines, draft, { color: '#123456' })

  it('draws applied lines as their order kind, labelled with the draft line', () => {
    expect(intentions[0]).toMatchObject({
      kind: 'build',
      line: 1,
      valid: true,
      territory: 'ROS',
      infrastructure: 'mill',
      level: 2,
      label: 'C M ROS',
      color: '#123456',
      source: 'draft',
    })
  })

  it('keeps a line refused only for lack of resources as a warning', () => {
    expect(intentions[1]).toMatchObject({
      kind: 'recruit_troop',
      valid: true,
      warning: true,
      reason: 'insufficient_resources',
      territory: 'BRU',
    })
  })

  it('turns malformed lines and other refusals into errors', () => {
    expect(intentions[2]).toMatchObject({
      kind: 'error',
      valid: false,
      message: 'Ligne mal formée',
      label: 'NOT A LINE',
    })
    expect(intentions[3]).toMatchObject({
      kind: 'error',
      valid: false,
      reason: 'structure_present',
      territory: 'ROS',
    })
  })

  it('draws transfers between their source and target and skips discards', () => {
    expect(intentions[4]).toMatchObject({
      kind: 'transfer',
      sourceTerritory: 'ROS',
      targetTerritory: 'ATL',
      amount: 3,
      territory: undefined,
    })
    expect(intentions).toHaveLength(5)
  })

  it('marks submitted lines for the observer overlay', () => {
    expect(
      buildWinterIntentions([lines[0]], draft, { source: 'submitted' })[0].source,
    ).toBe('submitted')
  })
})
