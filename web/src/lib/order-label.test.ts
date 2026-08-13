import { describe, expect, it } from 'vitest'

import { formatOrderLabel } from '@/lib/order-label'

describe('formatOrderLabel', () => {
  it('renders territory-targeted orders with their complete syntax', () => {
    expect(
      formatOrderLabel({
        type: 'attack',
        position: 'ROS',
        targets: ['BRU'],
        liaison: 'single',
      }),
    ).toBe('ROS A BRU')
    expect(
      formatOrderLabel({
        type: 'support',
        position: 'ROS',
        targets: ['BRU', 'CHA'],
        liaison: 'single',
      }),
    ).toBe('ROS S BRU - CHA')
    expect(formatOrderLabel({ type: 'hold', position: 'ROS', liaison: 'single' })).toBe(
      'H ROS',
    )
    expect(
      formatOrderLabel({ type: 'pillage', position: 'ROS', liaison: 'single' }),
    ).toBe('P ROS')
    expect(
      formatOrderLabel({
        type: 'join',
        position: 'ROS',
        targets: ['BRU'],
        liaison: 'single',
      }),
    ).toBe('ROS J BRU')
  })

  it('renders dispersion assignments and loop liaison', () => {
    expect(
      formatOrderLabel({
        type: 'disperse',
        position: 'ROS',
        targets: ['ROS', 'BRU'],
        nobleAssignments: { ROS: ['JEA'], BRU: ['BOB'] },
        liaison: 'single',
      }),
    ).toBe('ROS D ROS*JEA BRU*BOB')
    expect(
      formatOrderLabel({
        type: 'disperse',
        position: 'ROS',
        targets: ['ROS', 'BRU'],
        nobleAssignments: { ROS: ['*'], BRU: ['BOB'] },
        liaison: 'single',
      }),
    ).toBe('ROS D ROS* BRU*BOB')
  })
})
