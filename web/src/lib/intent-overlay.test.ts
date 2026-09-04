import { describe, expect, it } from 'vitest'

import { buildIntentions } from '@/lib/intent-overlay'
import type { MapData, StateData } from '@/types'

const map: MapData = {
  territories: [
    {
      id: 'ROS',
      name: 'Rosemont',
      terrain: 'plain',
      village: false,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['BRU', 'CHA'],
      impassable: [],
    },
    {
      id: 'BRU',
      name: 'Brisecote',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['ROS', 'CHA'],
      impassable: [],
    },
    {
      id: 'CHA',
      name: 'Champborne',
      terrain: 'hill',
      village: false,
      points: [
        [0, 50],
        [50, 50],
        [50, 100],
        [0, 100],
      ],
      adjacencies: ['ROS', 'BRU'],
      impassable: [],
    },
  ],
}

function armyAt(id: string) {
  return { id, owner: 'P1', resources: 0, army: null, infrastructures: [] }
}

const baseState: StateData = {
  turn: 1,
  season: 'spring',
  players: [{ id: 'P1', name: 'One', color: '#a84632' }],
  territories: [
    {
      id: 'ROS',
      owner: 'P1',
      resources: 0,
      army: { owner: 'P1', size: 3, chain: null },
      infrastructures: [],
    },
    armyAt('BRU'),
    armyAt('CHA'),
  ],
  nobles: [
    {
      id: 'N1',
      code: 'HUG',
      name: 'Hugues',
      owner: 'P1',
      location: 'ROS',
      status: 'free',
    },
  ],
}

const ROS_CENTROID = [25, 25] as const

const supportMap: MapData = {
  territories: [
    {
      id: 'ART',
      name: 'Atlas',
      terrain: 'plain',
      village: false,
      points: [
        [0, 0],
        [50, 0],
        [50, 50],
        [0, 50],
      ],
      adjacencies: ['BOR', 'CID'],
      impassable: [],
    },
    {
      id: 'BOR',
      name: 'Borie',
      terrain: 'forest',
      village: false,
      points: [
        [50, 0],
        [100, 0],
        [100, 50],
        [50, 50],
      ],
      adjacencies: ['ART', 'CID'],
      impassable: [],
    },
    {
      id: 'CID',
      name: 'Cider',
      terrain: 'hill',
      village: false,
      points: [
        [50, 50],
        [100, 50],
        [100, 100],
        [50, 100],
      ],
      adjacencies: ['ART', 'BOR'],
      impassable: [],
    },
  ],
}

function stateWith(overrides: Partial<StateData> = {}): StateData {
  return { ...baseState, territories: baseState.territories, ...overrides }
}

function supportState(): StateData {
  return {
    ...baseState,
    players: [{ id: 'P1', name: 'One', color: '#a84632' }],
    territories: [
      {
        id: 'ART',
        owner: 'P1',
        resources: 0,
        army: { owner: 'P1', size: 4, chain: null },
        infrastructures: [],
      },
      { id: 'BOR', owner: 'P2', resources: 0, army: null, infrastructures: [] },
      { id: 'CID', owner: 'P2', resources: 0, army: null, infrastructures: [] },
    ],
    nobles: [
      {
        id: 'N1',
        code: 'HUG',
        name: 'Hugues',
        owner: 'P1',
        location: 'ART',
        status: 'free',
      },
    ],
  }
}

describe('buildIntentions', () => {
  it('renders an attack penetrating into the destination territory', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A BRU',
    })

    expect(intentions).toHaveLength(1)
    expect(intentions[0]).toMatchObject({
      armyTerritory: 'ROS',
      symbol: 'A',
      turn: 1,
      nobleCode: 'HUG',
    })
    expect(intentions[0].from).toEqual([...ROS_CENTROID])
    expect(intentions[0].segments[0]).toMatchObject({ kind: 'attack' })
    expect(intentions[0].segments[0].to).toEqual([75, 25])
  })

  it('renders every order in a draft, including the first order', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nH ROS\nROS A BRU',
    })

    expect(intentions).toHaveLength(2)
    expect(intentions.map(({ symbol, turn }) => ({ symbol, turn }))).toEqual([
      { symbol: 'H', turn: 1 },
      { symbol: 'A', turn: 2 },
    ])
  })

  it('renders later draft orders from their declared positions', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A BRU\nBRU A CHA',
    })

    expect(intentions).toHaveLength(2)
    expect(intentions[1]).toMatchObject({ symbol: 'A', turn: 2 })
    expect(intentions[1].from).toEqual([75, 25])
    expect(intentions[1].segments[0].to).toEqual([25, 75])
  })

  it('renders a join toward the destination center', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS J BRU',
    })

    expect(intentions[0].turn).toBe(1)
    expect(intentions[0].segments[0]).toMatchObject({ kind: 'movement' })
    expect(intentions[0].segments[0].to).toEqual([75, 25])
  })

  it('renders defensive support toward the supported territory center', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS S BRU',
    })

    expect(intentions[0].turn).toBe(1)
    expect(intentions[0].segments[0]).toMatchObject({ kind: 'support-defensive' })
    expect(intentions[0].segments[0].to).toEqual([75, 25])
  })

  it('renders offensive support toward the frontier where the attack happens', () => {
    const intentions = buildIntentions(supportMap, supportState(), 'P1', {
      HUG: 'HUG\nART H\nART S BOR - CID',
    })

    expect(intentions).toHaveLength(1)
    expect(intentions[0].turn).toBe(1)
    expect(intentions[0].segments[0]).toMatchObject({ kind: 'support-offensive' })
    expect(intentions[0].segments[0].to).toEqual([75, 50])
  })

  it('renders one arrow per disperse destination and a loop for the origin', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS D BRU CHA ROS',
    })

    expect(intentions).toHaveLength(1)
    const { segments } = intentions[0]
    expect(segments.map((segment) => segment.kind)).toEqual([
      'loop',
      'movement',
      'movement',
    ])
    expect(segments[0].from).toEqual([...ROS_CENTROID])
    expect(segments[0].to).toEqual([...ROS_CENTROID])
  })

  it('numbers the following turn for a disperse draft order', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS D BRU ROS',
    })

    expect(intentions[0].turn).toBe(1)
  })

  it('renders the residual loop when a disperse leaves troops at the origin', () => {
    const implicit = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS D BRU',
    })
    const explicit = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS D ROS BRU',
    })

    expect(implicit[0].segments.map((segment) => segment.kind)).toEqual([
      'loop',
      'movement',
    ])
    expect(explicit[0].segments.map((segment) => segment.kind)).toEqual([
      'loop',
      'movement',
    ])
    expect(implicit[0].segments).toEqual(explicit[0].segments)
  })

  it('does not render a loop when every troop is assigned to a destination', () => {
    const emptied = buildIntentions(
      map,
      stateWith({
        territories: baseState.territories.map((territory, index) =>
          index === 0
            ? { ...territory, army: { owner: 'P1', size: 2, chain: null } }
            : territory,
        ),
      }),
      'P1',
      { HUG: 'HUG\nROS D BRU CHA' },
    )

    expect(emptied[0].segments.map((segment) => segment.kind)).toEqual([
      'movement',
      'movement',
    ])
  })

  it('renders hold and pillage as stationary icons without segments', () => {
    const held = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nH ROS',
    })
    const pillaged = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nP ROS',
    })

    expect(held[0]).toMatchObject({ symbol: 'H', turn: 1, segments: [] })
    expect(pillaged[0]).toMatchObject({ symbol: 'P', turn: 1, segments: [] })
  })

  it('ignores loop liaisons except for disperse', () => {
    const loopedSupport = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(ROS S BRU)',
    })
    const loopedHold = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(H ROS)',
    })
    const loopedDisperse = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(ROS D BRU CHA)',
    })

    expect(loopedSupport).toHaveLength(0)
    expect(loopedHold).toHaveLength(0)
    expect(loopedDisperse).toHaveLength(1)
  })

  it('ignores invalid orders without breaking the rest of the parse', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A UNKNOWN\nROS sELF\nROS A BRU',
    })

    expect(intentions).toHaveLength(1)
    expect(intentions[0].symbol).toBe('A')
    expect(intentions[0].turn).toBe(2)
    expect(intentions[0].segments[0].to).toEqual([75, 25])
  })

  it('ignores self-support and self-attack', () => {
    const selfSupport = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS S ROS',
    })
    const selfAttack = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A ROS',
    })

    expect(selfSupport).toHaveLength(0)
    expect(selfAttack).toHaveLength(0)
  })

  it('does not render anything during winter', () => {
    const intentions = buildIntentions(map, stateWith({ season: 'winter' }), 'P1', {
      HUG: 'HUG\nH ROS\nROS A BRU',
    })

    expect(intentions).toHaveLength(0)
  })

  it('renders visible installed chains including the current order', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {})

    const intentionsWithChain = buildIntentions(
      map,
      stateWith({
        territories: baseState.territories.map((territory, index) =>
          index === 0
            ? {
                ...territory,
                army: {
                  owner: 'P1',
                  size: 4,
                  chain: {
                    visibility: 'known',
                    currentIndex: 0,
                    orders: [
                      {
                        type: 'attack',
                        position: 'ROS',
                        targets: ['BRU'],
                        liaison: 'single',
                      },
                      {
                        type: 'attack',
                        position: 'BRU',
                        targets: ['CHA'],
                        liaison: 'single',
                      },
                      {
                        type: 'hold',
                        position: 'CHA',
                        targets: [],
                        liaison: 'single',
                      },
                    ],
                  },
                },
              }
            : territory,
        ),
      }),
      'P1',
      {},
    )

    expect(intentions).toHaveLength(0)
    expect(intentionsWithChain).toHaveLength(3)
    expect(intentionsWithChain[0]).toMatchObject({ symbol: 'A', turn: 1 })
    expect(intentionsWithChain[1]).toMatchObject({ symbol: 'A', turn: 2 })
    expect(intentionsWithChain[1].from).toEqual([75, 25])
    expect(intentionsWithChain[2]).toMatchObject({ symbol: 'H', turn: 3 })
  })

  it('renders the current order for an installed chain', () => {
    const intentions = buildIntentions(
      map,
      stateWith({
        territories: baseState.territories.map((territory, index) =>
          index === 0
            ? {
                ...territory,
                army: {
                  owner: 'P1',
                  size: 2,
                  chain: {
                    visibility: 'known',
                    currentIndex: 0,
                    orders: [
                      {
                        type: 'attack',
                        position: 'ROS',
                        targets: ['BRU'],
                        liaison: 'single',
                      },
                      {
                        type: 'join',
                        position: 'BRU',
                        targets: ['CHA'],
                        liaison: 'single',
                      },
                    ],
                  },
                },
              }
            : territory,
        ),
      }),
      'P1',
      {},
    )

    expect(intentions).toHaveLength(2)
    expect(intentions[0]).toMatchObject({ symbol: 'A', turn: 1 })
    expect(intentions[1]).toMatchObject({ symbol: 'J', turn: 2 })
    expect(intentions[1].from).toEqual([75, 25])
  })

  it('skips hidden installed chains', () => {
    const intentions = buildIntentions(
      map,
      stateWith({
        territories: baseState.territories.map((territory, index) =>
          index === 0
            ? {
                ...territory,
                army: {
                  owner: 'P1',
                  size: 2,
                  chain: {
                    visibility: 'hidden',
                    orders: [
                      {
                        type: 'attack',
                        position: 'ROS',
                        targets: ['BRU'],
                        liaison: 'single',
                      },
                      {
                        type: 'join',
                        position: 'ROS',
                        targets: ['CHA'],
                        liaison: 'single',
                      },
                    ],
                  },
                },
              }
            : territory,
        ),
      }),
      'P1',
      {},
    )

    expect(intentions).toHaveLength(0)
  })

  it('ignores drafts from non-owned or dungeon nobles', () => {
    const dungeonNoble = buildIntentions(map, stateWith(), 'P1', {
      ENN: 'ENN\nH ROS\nROS A BRU',
    })
    const hostageNoble = buildIntentions(
      map,
      stateWith({
        nobles: [
          ...baseState.nobles,
          {
            id: 'N2',
            code: 'OTG',
            name: 'Otto',
            owner: 'P2',
            location: 'BRU',
            status: 'hostage',
          },
        ],
      }),
      'P1',
      { OTG: 'OTG\nH BRU\nBRU S ROS' },
    )

    expect(dungeonNoble).toHaveLength(0)
    expect(hostageNoble).toHaveLength(0)
  })

  it('drops a draft order when no owned army is at the order position', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nH BRU\nBRU A ROS',
    })

    expect(intentions).toHaveLength(0)
  })
})
