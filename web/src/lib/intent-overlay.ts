import { parseChainDraft } from '@/lib/chain-parse'
import { formatOrderLabel } from '@/lib/order-label'
import type {
  MapData,
  Order,
  OrderType,
  PlayerId,
  Point,
  StateData,
  Territory,
} from '@/types'

const SYMBOLS: Record<OrderType, string> = {
  attack: 'A',
  support: 'S',
  hold: 'H',
  join: 'J',
  pillage: 'P',
  disperse: 'D',
}

export type IntentSegmentKind =
  'attack' | 'movement' | 'support-defensive' | 'support-offensive' | 'loop'

export interface IntentSegment {
  from: Point
  to: Point
  kind: IntentSegmentKind
}

export interface Intention {
  armyTerritory: string
  from: Point
  symbol: string
  type: OrderType
  turn: number
  nobleCode?: string
  label: string
  segments: IntentSegment[]
}

function centroid(points: Point[]): Point {
  if (points.length === 0) return [0, 0]
  const total = points.reduce((sum, [x, y]) => ({ x: sum.x + x, y: sum.y + y }), {
    x: 0,
    y: 0,
  })
  return [total.x / points.length, total.y / points.length]
}

function pointKey([x, y]: Point): string {
  return `${x},${y}`
}

function edgeKey(from: Point, to: Point): string {
  const fromKey = pointKey(from)
  const toKey = pointKey(to)
  return fromKey < toKey ? `${fromKey}|${toKey}` : `${toKey}|${fromKey}`
}

function polygonPoints(territory: Territory): Point[] {
  return territory.points
}

function territoryCentroid(map: MapData, id: string): Point | null {
  const territory = map.territories.find((candidate) => candidate.id === id)
  return territory ? centroid(polygonPoints(territory)) : null
}

function sharedBorderMidpoint(map: MapData, first: string, second: string): Point | null {
  const firstTerritory = map.territories.find((candidate) => candidate.id === first)
  const secondTerritory = map.territories.find((candidate) => candidate.id === second)
  if (!firstTerritory || !secondTerritory) return null

  const secondEdges = new Set(
    polygonPoints(secondTerritory).map((point, index) =>
      edgeKey(
        point,
        polygonPoints(secondTerritory)[
          (index + 1) % polygonPoints(secondTerritory).length
        ],
      ),
    ),
  )

  let count = 0
  let sumX = 0
  let sumY = 0
  const firstPoints = polygonPoints(firstTerritory)
  for (let index = 0; index < firstPoints.length; index += 1) {
    const from = firstPoints[index]
    const to = firstPoints[(index + 1) % firstPoints.length]
    if (!secondEdges.has(edgeKey(from, to))) continue
    sumX += (from[0] + to[0]) / 2
    sumY += (from[1] + to[1]) / 2
    count += 1
  }

  if (count === 0) return null
  return [sumX / count, sumY / count]
}

function isAdjacent(map: MapData, from: string, to: string): boolean {
  const territory = map.territories.find((candidate) => candidate.id === from)
  return territory?.adjacencies.includes(to) ?? false
}

function known(map: MapData, id: string): boolean {
  return map.territories.some((candidate) => candidate.id === id)
}

function isInvalidLoop(map: MapData, order: Order): boolean {
  for (const target of order.targets ?? []) {
    if (known(map, target) && target === order.position) return true
  }
  return false
}

function makeIntention(
  map: MapData,
  order: Order,
  territoryState: { id: string; size: number },
  turn: number,
  nobleCode?: string,
): Intention | null {
  if (order.liaison === 'loop' && order.type !== 'disperse') return null

  const position = order.position
  const from = territoryCentroid(map, position)
  if (!from) return null

  const base: Omit<Intention, 'type' | 'segments'> = {
    armyTerritory: territoryState.id,
    from,
    symbol: SYMBOLS[order.type],
    turn,
    nobleCode,
    label: formatOrderLabel({
      type: order.type,
      position: order.position,
      targets: order.targets,
      nobleAssignments: order.nobleAssignments,
      liaison: order.liaison,
    }),
  }

  switch (order.type) {
    case 'attack': {
      const target = order.targets?.[0]
      if (!target || !known(map, target) || !isAdjacent(map, position, target)) {
        return null
      }
      if (isInvalidLoop(map, order)) return null
      const to = territoryCentroid(map, target)
      if (!to) return null
      return { ...base, type: order.type, segments: [{ from, to, kind: 'attack' }] }
    }
    case 'join': {
      const target = order.targets?.[0]
      if (!target || !known(map, target) || !isAdjacent(map, position, target)) {
        return null
      }
      if (isInvalidLoop(map, order)) return null
      const to = territoryCentroid(map, target)
      if (!to) return null
      return { ...base, type: order.type, segments: [{ from, to, kind: 'movement' }] }
    }
    case 'support': {
      const supported = order.targets?.[0]
      const destination = order.targets?.[1]
      if (!supported || !known(map, supported)) return null
      if (supported === position || !isAdjacent(map, position, supported)) return null
      if (destination) {
        if (!known(map, destination) || destination === position) return null
        if (!isAdjacent(map, position, destination)) return null
        if (!isAdjacent(map, supported, destination)) return null
        const to =
          sharedBorderMidpoint(map, supported, destination) ??
          territoryCentroid(map, destination)
        if (!to) return null
        return {
          ...base,
          type: order.type,
          segments: [{ from, to, kind: 'support-offensive' }],
        }
      }
      const to = territoryCentroid(map, supported)
      if (!to) return null
      return {
        ...base,
        type: order.type,
        segments: [{ from, to, kind: 'support-defensive' }],
      }
    }
    case 'disperse': {
      const targets = order.targets ?? []
      for (const target of targets) {
        if (!known(map, target)) return null
        if (target !== position && !isAdjacent(map, position, target)) return null
      }
      const segments: IntentSegment[] = []
      const seen = new Set<string>()
      const holdsGround =
        targets.includes(position) || territoryState.size > targets.length
      if (holdsGround) {
        segments.push({ from, to: from, kind: 'loop' })
        seen.add(position)
      }
      for (const target of targets) {
        if (seen.has(target)) continue
        seen.add(target)
        const to = territoryCentroid(map, target)
        if (!to) return null
        segments.push({ from, to, kind: 'movement' })
      }
      if (segments.length === 0) return null
      return { ...base, type: order.type, segments }
    }
    case 'hold':
    case 'pillage': {
      if (order.liaison === 'loop') return null
      return { ...base, type: order.type, segments: [] }
    }
    default:
      return null
  }
}

function appendChainIntentions(
  map: MapData,
  orders: Order[],
  startIndex: number,
  army: { id: string; size: number },
  intentions: Intention[],
  nobleCode?: string,
): void {
  for (let orderIndex = startIndex; orderIndex < orders.length; orderIndex += 1) {
    const order = orders[orderIndex]
    if (order.liaison === 'loop' && order.type !== 'disperse') continue
    const intention = makeIntention(
      map,
      order,
      army,
      orderIndex - startIndex + 1,
      nobleCode,
    )
    if (intention) intentions.push(intention)
  }
}

export function buildIntentions(
  map: MapData,
  state: StateData,
  player: PlayerId,
  chainDrafts: Record<string, string>,
): Intention[] {
  if (state.season === 'winter') return []

  const intentions: Intention[] = []

  for (const territoryState of state.territories) {
    const army = territoryState.army
    if (!army || army.owner !== player) continue
    const chain = army.chain
    if (!chain || (chain.visibility ?? 'known') === 'hidden') continue
    const orders = chain.orders ?? []
    appendChainIntentions(
      map,
      orders,
      Math.max(0, chain.currentIndex ?? 0),
      { id: territoryState.id, size: army.size },
      intentions,
    )
  }

  const ownedNobleCodes = new Set(
    state.nobles
      .filter((noble) => noble.owner === player && noble.status !== 'dungeon')
      .map((noble) => noble.code),
  )

  for (const [nobleCode, text] of Object.entries(chainDrafts)) {
    if (!ownedNobleCodes.has(nobleCode) || !text.trim()) continue
    const parsedOrders = parseChainDraft(text)
    const firstOrder = parsedOrders[0]
    const territoryState = firstOrder
      ? state.territories.find(
          (candidate) =>
            candidate.id === firstOrder.position && candidate.army?.owner === player,
        )
      : null
    if (!territoryState?.army) continue
    appendChainIntentions(
      map,
      parsedOrders,
      0,
      { id: territoryState.id, size: territoryState.army.size },
      intentions,
      nobleCode,
    )
  }

  return intentions
}
