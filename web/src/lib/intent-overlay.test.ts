import { describe, expect, it } from 'vitest'

import {
  affluentCurve,
  bezierPoint,
  anchorIntentions,
  assignSupportSlots,
  buildIntentions as buildIntentionsFromOrders,
} from '@/lib/intent-overlay'
import { draftOrders } from '@/test/parse-orders'

/** Keeps the fixtures readable: drafts are written as order text. */
const buildIntentions = (
  map: MapData,
  state: StateData,
  player: string,
  drafts: Record<string, string>,
  options?: Parameters<typeof buildIntentionsFromOrders>[4],
) => buildIntentionsFromOrders(map, state, player, draftOrders(drafts), options)
import type { MapData, Order, StateData } from '@/types'

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
  it('renders submitted orders with the player color without installed chains', () => {
    const intentions = buildIntentions(
      map,
      stateWith(),
      'P1',
      { HUG: 'ROS A BRU' },
      { includeInstalled: false, source: 'submitted', color: '#123456' },
    )

    expect(intentions).toHaveLength(1)
    expect(intentions[0]).toMatchObject({ source: 'submitted', color: '#123456' })
  })

  it('colors installed chains when building a spectator view', () => {
    const state = stateWith({
      territories: stateWith().territories.map((territory) =>
        territory.id === 'ROS'
          ? {
              ...territory,
              army: {
                owner: 'P1',
                size: 3,
                chain: {
                  noble: 'HUG',
                  currentIndex: 0,
                  orders: [
                    {
                      type: 'attack',
                      position: 'ROS',
                      targets: ['BRU'],
                      liaison: 'single',
                    },
                  ],
                },
              },
            }
          : territory,
      ),
    })
    const intentions = buildIntentions(map, state, 'P1', {}, { color: '#654321' })

    expect(intentions[0]).toMatchObject({ source: 'chain', color: '#654321' })
  })

  it('renders an attack penetrating into the destination territory', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A BRU',
    })

    expect(intentions).toHaveLength(1)
    expect(intentions[0]).toMatchObject({
      armyTerritory: 'ROS',
      symbol: 'A',
      turn: 1,
      source: 'draft',
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
    expect(intentions.map(({ turnLabel }) => turnLabel)).toEqual(['1', '2'])
  })

  it('renders later draft orders from their declared positions', () => {
    const intentions = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\nROS A BRU\nBRU A CHA',
    })

    expect(intentions).toHaveLength(2)
    expect(intentions[1]).toMatchObject({ symbol: 'A', turn: 2 })
    expect(intentions[1].turnLabel).toBe('2')
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

  it('draws the legs the server plans for a maneuver', () => {
    const order: Order = {
      type: 'maneuver',
      position: 'ROS',
      targets: ['BRU', 'CHA'],
      counts: [1, -1],
      liaison: 'single',
      legs: [
        { target: 'ROS', troops: 1, nobles: ['HUG'] },
        { target: 'BRU', troops: 1 },
        { target: 'CHA', troops: 2, nobles: ['JEA'], carriesChain: true },
      ],
    }
    const [intention] = buildIntentionsFromOrders(map, stateWith(), 'P1', {
      HUG: [order],
    })

    expect(intention.segments.map((segment) => segment.kind)).toEqual([
      'loop',
      'movement',
      'movement',
    ])
    expect(intention.segments[0].nobles).toEqual(['HUG'])
    expect(intention.segments[1]).not.toHaveProperty('nobles')
    expect(intention.segments[2].nobles).toEqual(['JEA'])
    expect(intention.segments[2].carriesChain).toBe(true)
    expect(intention.segments[0].carriesChain).toBeUndefined()
  })

  it('shows a wildcard for nobles the server cannot place yet', () => {
    const order: Order = {
      type: 'maneuver',
      position: 'ROS',
      liaison: 'single',
      legs: [{ target: 'BRU', troops: 1, anyNobles: true, carriesChain: true }],
    }
    const [intention] = buildIntentionsFromOrders(map, stateWith(), 'P1', {
      HUG: [order],
    })

    expect(intention.segments[0].nobles).toEqual(['*'])
  })

  it('ignores a maneuver whose legs leave the neighbourhood', () => {
    const order: Order = {
      type: 'maneuver',
      position: 'ROS',
      liaison: 'single',
      legs: [{ target: 'FAR', troops: 1 }],
    }

    expect(buildIntentionsFromOrders(map, stateWith(), 'P1', { HUG: [order] })).toEqual(
      [],
    )
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

  it('renders loop liaisons and marks their turns as uncertain', () => {
    const loopedSupport = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(ROS S BRU)\nBRU A CHA',
    })
    const loopedHold = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(H ROS)\nROS A BRU',
    })
    const loopedDisperse = buildIntentions(map, stateWith(), 'P1', {
      HUG: 'HUG\n(ROS D BRU CHA)',
    })

    expect(loopedSupport).toHaveLength(2)
    expect(loopedSupport.map(({ turnLabel }) => turnLabel)).toEqual(['?', '?'])
    expect(loopedHold).toHaveLength(2)
    expect(loopedHold.map(({ turnLabel }) => turnLabel)).toEqual(['?', '?'])
    expect(loopedDisperse).toHaveLength(1)
    expect(loopedDisperse[0].turnLabel).toBe('?')
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

  it('renders installed chains during winter without rendering drafts', () => {
    const winterState = stateWith({
      season: 'winter',
      territories: baseState.territories.map((territory, index) =>
        index === 0
          ? {
              ...territory,
              army: {
                owner: 'P1',
                size: 3,
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
                  ],
                },
              },
            }
          : territory,
      ),
    })

    const intentions = buildIntentions(
      map,
      winterState,
      'P1',
      { HUG: 'HUG\nROS H' },
      { includeInstalledInWinter: true },
    )

    expect(intentions).toHaveLength(1)
    expect(intentions[0]).toMatchObject({ source: 'chain', symbol: 'A' })
  })

  it('shows only the remaining installed chain orders in winter after it has advanced', () => {
    const winterState = stateWith({
      season: 'winter',
      territories: baseState.territories.map((territory, index) =>
        index === 0
          ? {
              ...territory,
              army: {
                owner: 'P1',
                size: 3,
                chain: {
                  visibility: 'known',
                  currentIndex: 1,
                  orders: [
                    {
                      type: 'attack',
                      position: 'ROS',
                      targets: ['BRU'],
                      liaison: 'single',
                    },
                    {
                      type: 'hold',
                      position: 'BRU',
                      targets: [],
                      liaison: 'loop',
                    },
                  ],
                },
              },
            }
          : territory,
      ),
    })

    const intentions = buildIntentions(
      map,
      winterState,
      'P1',
      {},
      { includeInstalledInWinter: true },
    )

    expect(intentions.map(({ symbol, turn }) => ({ symbol, turn }))).toEqual([
      { symbol: 'H', turn: 1 },
    ])
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
    expect(intentionsWithChain[0]).toMatchObject({
      symbol: 'A',
      turn: 1,
      source: 'chain',
    })
    expect(intentionsWithChain[1]).toMatchObject({
      symbol: 'A',
      turn: 2,
      source: 'chain',
    })
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
    expect(intentions[0]).toMatchObject({ symbol: 'A', turn: 1, source: 'chain' })
    expect(intentions[1]).toMatchObject({ symbol: 'J', turn: 2, source: 'chain' })
    expect(intentions[1].from).toEqual([75, 25])
  })

  it('marks installed orders after a loop with uncertain turns', () => {
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
                        type: 'hold',
                        position: 'BRU',
                        targets: [],
                        liaison: 'loop',
                      },
                      {
                        type: 'attack',
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

    expect(intentions.map(({ turnLabel }) => turnLabel)).toEqual(['1', '?', '?'])
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

  describe('support geometry', () => {
    const offensive = () =>
      buildIntentions(supportMap, supportState(), 'P1', {
        HUG: 'HUG\nART H\nART S BOR - CID',
      })[0].segments[0]
    const defensive = () =>
      buildIntentions(supportMap, supportState(), 'P1', {
        HUG: 'HUG\nART H\nART S BOR',
      })[0].segments[0]

    it('values a support as the army size plus one for its noble', () => {
      expect(offensive().offensive?.value).toBe(5)
      expect(defensive().defensive?.value).toBe(5)
    })

    it('keeps the tributary inside the two territories it links and ends on the attack tip', () => {
      // ART is [0,50]x[0,50], CID is [50,100]x[50,100].
      const inside = ([x, y]: [number, number]) =>
        (x >= -1e-6 && x <= 50 + 1e-6 && y >= -1e-6 && y <= 50 + 1e-6) ||
        (x >= 50 - 1e-6 && x <= 100 + 1e-6 && y >= 50 - 1e-6 && y <= 100 + 1e-6)
      const support = offensive().offensive!
      const { curves, tail } = affluentCurve(support)
      expect(tail).toEqual(support.arrowEnd)
      for (const curve of curves) {
        for (let step = 0; step <= 20; step += 1) {
          expect(inside(bezierPoint(curve, step / 20))).toBe(true)
        }
      }
    })

    it('starts the tributary on the attacked territory boundary nearest the supporting army', () => {
      expect(offensive().offensive?.start).toEqual([50, 50])
    })

    it('lists every passable border of the held territory with an inward normal', () => {
      const palisade = defensive().defensive!
      expect(palisade.borders).toHaveLength(2)
      for (const border of palisade.borders) {
        const middle = [
          (border.from[0] + border.to[0]) / 2,
          (border.from[1] + border.to[1]) / 2,
        ]
        // BOR is the square [50..100] x [0..50]: inward points to its center.
        expect(
          (75 - middle[0]) * border.inward[0] + (25 - middle[1]) * border.inward[1],
        ).toBeGreaterThan(0)
      }
      expect(palisade.link.from).toEqual([50, 50])
    })

    describe('a supporter that does not touch the supported army', () => {
      // S and A both touch D, but not each other.
      const square = (id: string, x: number, y: number, adjacencies: string[]) => ({
        id,
        name: id,
        terrain: 'plain' as const,
        village: false,
        points: [
          [x, y],
          [x + 50, y],
          [x + 50, y + 50],
          [x, y + 50],
        ] as [number, number][],
        adjacencies,
        impassable: [],
      })
      const apartMap: MapData = {
        territories: [
          square('SUP', 0, 0, ['DST']),
          square('DST', 50, 0, ['SUP', 'AID']),
          square('AID', 50, 50, ['DST']),
        ],
      }
      const apartState: StateData = {
        ...supportState(),
        territories: [
          {
            id: 'SUP',
            owner: 'P1',
            resources: 0,
            army: { owner: 'P1', size: 2, chain: null },
            infrastructures: [],
          },
          { id: 'DST', owner: 'P2', resources: 0, army: null, infrastructures: [] },
          {
            id: 'AID',
            owner: 'P1',
            resources: 0,
            army: { owner: 'P1', size: 3, chain: null },
            infrastructures: [],
          },
        ],
        nobles: [
          {
            id: 'N1',
            code: 'HUG',
            name: 'Hugues',
            owner: 'P1',
            location: 'AID',
            status: 'free',
          },
        ],
      }
      const draw = () =>
        buildIntentions(apartMap, apartState, 'P1', { HUG: 'HUG\nAID S SUP - DST' })

      it('is still drawn', () => {
        expect(draw()).toHaveLength(1)
        expect(draw()[0].segments[0].offensive).toMatchObject({
          supported: 'SUP',
          destination: 'DST',
          value: 4,
        })
      })

      it('leaves the supporting army, crosses the shared border and joins the attack arrow inside the attacked territory', () => {
        const anchors = new Map<string, [number, number]>([
          ['AID', [70, 90]],
          ['SUP', [20, 30]],
        ])
        const [intention] = anchorIntentions(apartMap, draw(), anchors)
        const support = intention.segments[0].offensive!
        expect(intention.segments[0].from).toEqual([70, 90])
        expect(support.origin).toEqual([70, 90])
        // The border shared by AID and DST is y = 50.
        expect(support.start[1]).toBe(50)
        // The arrow runs from the supported army to the attacked territory's centre.
        expect(support.arrowEnd).toEqual([75, 25])
        expect(support.arrowEntry[0]).toBeCloseTo(50)
        const { curves, tail } = affluentCurve(support)
        const [[start, , near, crossing], [again, far, lead, end]] = curves
        expect(start).toEqual([70, 90])
        // The two halves meet on the shared border, smoothly.
        expect(crossing[1]).toBeCloseTo(50)
        expect(again).toEqual(crossing)
        const turn =
          (crossing[0] - near[0]) * (far[1] - crossing[1]) -
          (crossing[1] - near[1]) * (far[0] - crossing[0])
        expect(Math.abs(turn)).toBeLessThan(1e-6)
        // It ends exactly on the tip of the attack arrow, tangent to its vector.
        expect(tail).toEqual(support.arrowEnd)
        const onArrow = ([x, y]: [number, number]) => {
          const [ax, ay] = [20, 30]
          const [bx, by] = support.arrowEnd
          return Math.abs((x - ax) * (by - ay) - (y - ay) * (bx - ax)) < 1e-6
        }
        expect(onArrow(end)).toBe(true)
        expect(onArrow(lead)).toBe(true)
        const [tx, ty] = [tail[0] - end[0], tail[1] - end[1]]
        const [vx, vy] = [support.arrowEnd[0] - 20, support.arrowEnd[1] - 30]
        expect(tx * vx + ty * vy).toBeGreaterThan(0)
        // The last stretch is a third of the way from the border to the centre.
        expect(Math.hypot(tx, ty)).toBeCloseTo(
          Math.hypot(
            support.arrowEnd[0] - support.arrowEntry[0],
            support.arrowEnd[1] - support.arrowEntry[1],
          ) / 3,
        )
        // AID is [50,100]x[50,100] and DST [50,100]x[0,50].
        for (const curve of curves) {
          for (let step = 0; step <= 20; step += 1) {
            const [x, y] = bezierPoint(curve, step / 20)
            expect(x).toBeGreaterThanOrEqual(50 - 1e-6)
            expect(x).toBeLessThanOrEqual(100 + 1e-6)
            expect(y).toBeGreaterThanOrEqual(-1e-6)
            expect(y).toBeLessThanOrEqual(100 + 1e-6)
          }
        }
      })
    })

    it('starts a defensive link at the supporting army and ends it on the border', () => {
      const [intention] = anchorIntentions(
        supportMap,
        buildIntentions(supportMap, supportState(), 'P1', {
          HUG: 'HUG\nART H\nART S BOR',
        }),
        new Map([['ART', [10, 10]]]),
      )
      expect(intention.segments[0].from).toEqual([10, 10])
      const link = intention.segments[0].defensive!.link
      // The border BOR shares with ART, the supporter's own territory.
      expect([link.from[0], link.to[0]]).toEqual([50, 50])
      expect([link.from[1], link.to[1]].sort((a, b) => a - b)).toEqual([0, 50])
    })

    it('leaves the later orders of a chain where they start', () => {
      const intentions = anchorIntentions(
        supportMap,
        buildIntentions(supportMap, supportState(), 'P1', {
          HUG: 'HUG\nART A BOR\nBOR A ART',
        }),
        new Map([['ART', [10, 10]]]),
      )
      expect(intentions).toHaveLength(2)
      expect(intentions[0].segments[0].from).toEqual([10, 10])
      // The second order starts from BOR, the centre of a territory the army has yet to reach.
      expect(intentions[1].segments[0].from).toEqual([75, 25])
    })

    it('puts supports of the same territory side by side, colors alternating', () => {
      const base = defensive()
      const mk = (color: string) => ({
        armyTerritory: 'ART',
        from: [25, 25] as [number, number],
        symbol: 'S',
        type: 'support' as const,
        turn: 1,
        turnLabel: '1',
        source: 'chain' as const,
        color,
        label: 'S',
        segments: [base],
      })
      const placed = assignSupportSlots([mk('#111'), mk('#111'), mk('#222')])
      const slots = placed.map((intention) => [
        intention.color,
        intention.segments[0].defensive?.slot,
        intention.segments[0].defensive?.slots,
      ])
      expect(slots).toEqual([
        ['#111', 0, 3],
        ['#111', 2, 3],
        ['#222', 1, 3],
      ])
    })
  })
})
