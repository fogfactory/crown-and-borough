import { describe, expect, it } from 'vitest'

import { parseChainDraft } from '@/lib/chain-parse'

describe('parseChainDraft', () => {
  it('parses a noble header and movement orders, ignoring the header', () => {
    const orders = parseChainDraft('HUG\nROS A BRI\nBRI J CHA')

    expect(orders).toHaveLength(2)
    expect(orders[0]).toMatchObject({
      type: 'attack',
      position: 'ROS',
      targets: ['BRI'],
      liaison: 'single',
    })
    expect(orders[1]).toMatchObject({
      type: 'join',
      position: 'BRI',
      targets: ['CHA'],
      liaison: 'single',
    })
  })

  it('parses loop liaisons and single-order lines', () => {
    const orders = parseChainDraft('HUG\n(BRI S ATL)')

    expect(orders).toHaveLength(1)
    expect(orders[0]).toMatchObject({
      type: 'support',
      position: 'BRI',
      targets: ['ATL'],
      liaison: 'loop',
    })
  })

  it('parses defensive and offensive support', () => {
    const orders = parseChainDraft('BRI S ROS\nBRI S ROS - CHA')

    expect(orders).toHaveLength(2)
    expect(orders[0]).toMatchObject({ targets: ['ROS'] })
    expect(orders[1]).toMatchObject({ targets: ['ROS', 'CHA'] })
  })

  it('parses disperse destinations with noble assignments', () => {
    const orders = parseChainDraft('BRI D ATL*HUG NOR*JEA')

    expect(orders).toHaveLength(1)
    expect(orders[0]).toMatchObject({
      type: 'disperse',
      position: 'BRI',
      targets: ['ATL', 'NOR'],
    })
    expect(orders[0].nobleAssignments).toEqual({
      ATL: ['HUG'],
      NOR: ['JEA'],
    })
  })

  it('keeps parsing the remaining lines after a malformed line', () => {
    const orders = parseChainDraft('HUG\n(This is broken\nROS A BRI')

    expect(orders).toHaveLength(1)
    expect(orders[0]).toMatchObject({ position: 'ROS', type: 'attack' })
  })

  it('ignores empty and comment-only lines', () => {
    const orders = parseChainDraft('# a comment\n\nROS A BRI # trailing')

    expect(orders).toHaveLength(1)
    expect(orders[0]).toMatchObject({ position: 'ROS', type: 'attack' })
  })

  it('ignores unknown symbols and incomplete orders', () => {
    const orders = parseChainDraft('ROS Z BRI\nROS A\nROS')

    expect(orders).toHaveLength(0)
  })

  it('ignores hold and pillage orders without an exact position token', () => {
    const orders = parseChainDraft('H \nP')

    expect(orders).toHaveLength(0)
  })
})
