import { distanceBetween, midpointOf } from '@/lib/map-gestures'
import { pointInPolygon, polygonEdges } from '@/lib/map-svg-geometry'
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
  maneuver: 'M',
  transfer: 'T',
}

export type IntentSegmentKind =
  'attack' | 'movement' | 'support-defensive' | 'support-offensive' | 'transfer' | 'loop'

export type IntentionSource = 'chain' | 'draft' | 'submitted'

/** A border of a territory: its two ends and the unit vector pointing into the territory. */
export interface BorderEdge {
  from: Point
  to: Point
  inward: Point
}

/**
 * A defensive support: a palisade along the passable borders of the supported
 * territory, a value disc, and a thin link from the supporting army.
 */
export interface DefensiveSupport {
  supported: string
  value: number
  /** Every passable border of the supported territory. */
  borders: BorderEdge[]
  /** The border the supporting army is linked to. */
  link: BorderEdge
  /** Place among the supports of the same territory (see `assignSupportSlots`). */
  slot: number
  slots: number
}

/**
 * An offensive support: a thin tributary from the supporting army, across the
 * border it shares with the attacked territory, to the tip of the attack arrow.
 */
export interface OffensiveSupport {
  supported: string
  destination: string
  value: number
  /** On the border shared by the attacked territory and the supporting army. */
  start: Point
  /** Where the supporting army stands: the tributary starts here. */
  origin: Point
  /** Where the attack arrow enters the attacked territory. */
  arrowEntry: Point
  /** Centre of the attacked territory, where the arrow ends. */
  arrowEnd: Point
  /** The attacked territory's outline. */
  area: Point[]
  /** The supporting army's territory outline: the curve stays within it and `area`. */
  originArea: Point[]
}

export interface IntentSegment {
  from: Point
  to: Point
  kind: IntentSegmentKind
  defensive?: DefensiveSupport
  offensive?: OffensiveSupport
  /** Nobles riding this leg of a dispersion or maneuver (`*` stands for all the remaining ones). */
  nobles?: string[]
  /** The order sheet (the chain) follows the group of this leg: the last destination of a maneuver. */
  carriesChain?: boolean
}

export interface Intention {
  armyTerritory: string
  from: Point
  symbol: string
  type: OrderType
  turn: number
  turnLabel: string
  source: IntentionSource
  color?: string
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

function sharedEdges(a: Territory, b: Territory): Array<[Point, Point]> {
  const other = new Set(polygonEdges(b.points).map(([from, to]) => edgeKey(from, to)))
  return polygonEdges(a.points).filter(([from, to]) => other.has(edgeKey(from, to)))
}

function inwardNormal(from: Point, to: Point, center: Point): Point {
  const dx = to[0] - from[0]
  const dy = to[1] - from[1]
  const length = Math.hypot(dx, dy) || 1
  let normal: Point = [-dy / length, dx / length]
  const middle: Point = [(from[0] + to[0]) / 2, (from[1] + to[1]) / 2]
  if (normal[0] * (center[0] - middle[0]) + normal[1] * (center[1] - middle[1]) < 0) {
    normal = [-normal[0], -normal[1]]
  }
  return normal
}

/** The passable borders of a territory, each with its inward normal. */
function passableBorders(map: MapData, id: string): BorderEdge[] {
  const territory = map.territories.find((candidate) => candidate.id === id)
  if (!territory) return []
  const center = centroid(territory.points)
  const borders: BorderEdge[] = []
  for (const neighborId of territory.adjacencies) {
    if (territory.impassable.includes(neighborId)) continue
    const neighbor = map.territories.find((candidate) => candidate.id === neighborId)
    if (!neighbor) continue
    for (const [from, to] of sharedEdges(territory, neighbor)) {
      borders.push({ from, to, inward: inwardNormal(from, to, center) })
    }
  }
  return borders
}

function sharedBorders(map: MapData, id: string, other: string): BorderEdge[] {
  const territory = map.territories.find((candidate) => candidate.id === id)
  const neighbor = map.territories.find((candidate) => candidate.id === other)
  if (!territory || !neighbor) return []
  const center = centroid(territory.points)
  return sharedEdges(territory, neighbor).map(([from, to]) => ({
    from,
    to,
    inward: inwardNormal(from, to, center),
  }))
}

function middleOf(edge: BorderEdge): Point {
  return midpointOf(edge.from, edge.to)
}

/** The border whose middle lies nearest to `origin`. */
function nearestBorder(borders: BorderEdge[], origin: Point): BorderEdge | null {
  return borders.reduce<BorderEdge | null>(
    (best, border) =>
      !best ||
      distanceSquared(middleOf(border), origin) < distanceSquared(middleOf(best), origin)
        ? border
        : best,
    null,
  )
}

/** The point of a border for slot `slot` out of `slots` side by side. */
export function borderSlotPoint(edge: BorderEdge, slot: number, slots: number): Point {
  const t = (slot + 0.5) / Math.max(1, slots)
  return [
    edge.from[0] + (edge.to[0] - edge.from[0]) * t,
    edge.from[1] + (edge.to[1] - edge.from[1]) * t,
  ]
}

function distanceSquared(a: Point, b: Point): number {
  return (a[0] - b[0]) ** 2 + (a[1] - b[1]) ** 2
}

function lerp(a: Point, b: Point, t: number): Point {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]
}

/** Where the segment a-b first meets the polygon's boundary, walking from a. */
function boundaryEntry(polygon: Point[], a: Point, b: Point): Point | null {
  let best: { t: number; point: Point } | null = null
  polygon.forEach((from, index) => {
    const to = polygon[(index + 1) % polygon.length]
    const rx = b[0] - a[0]
    const ry = b[1] - a[1]
    const sx = to[0] - from[0]
    const sy = to[1] - from[1]
    const denominator = rx * sy - ry * sx
    if (Math.abs(denominator) < 1e-9) return
    const t = ((from[0] - a[0]) * sy - (from[1] - a[1]) * sx) / denominator
    const u = ((from[0] - a[0]) * ry - (from[1] - a[1]) * rx) / denominator
    if (t < 0 || t > 1 || u < 0 || u > 1) return
    if (!best || t < best.t) best = { t, point: [a[0] + rx * t, a[1] + ry * t] }
  })
  return (best as { point: Point } | null)?.point ?? null
}

/**
 * The tributary of an offensive support. It leaves the supporting army (at
 * `origin`), reaches the border it shares with the attacked territory, and
 * curves inside that territory to join the attack arrow, which runs from the
 * supported army (at `supportedFrom`) to the attacked territory's centre.
 */
function offensiveSupportGeometry(
  map: MapData,
  position: string,
  supported: string,
  destination: string,
  value: number,
  origin: Point,
  supportedFrom: Point,
): OffensiveSupport | null {
  const target = map.territories.find((candidate) => candidate.id === destination)
  const arrowEnd = territoryCentroid(map, destination)
  if (!target || !arrowEnd) return null
  const shared = sharedBorders(map, destination, position)
  const nearest = nearestBorder(shared, origin)
  const start: Point = nearest
    ? middleOf(nearest)
    : target.points.reduce((best, point) =>
        distanceSquared(point, origin) < distanceSquared(best, origin) ? point : best,
      )
  const arrowEntry =
    boundaryEntry(target.points, supportedFrom, arrowEnd) ??
    sharedBorderMidpoint(map, supported, destination) ??
    lerp(supportedFrom, arrowEnd, 0.5)
  return {
    supported,
    destination,
    value,
    start,
    origin,
    arrowEntry,
    arrowEnd,
    area: target.points,
    originArea:
      map.territories.find((candidate) => candidate.id === position)?.points ?? [],
  }
}

/** A palisade on the passable borders of the held territory, linked to the supporting army. */
function defensiveSupportGeometry(
  map: MapData,
  position: string,
  supported: string,
  value: number,
  origin: Point,
  place: { slot: number; slots: number } = { slot: 0, slots: 1 },
): DefensiveSupport | null {
  const borders = passableBorders(map, supported)
  const touching = new Set(
    sharedBorders(map, supported, position).map((edge) => edgeKey(edge.from, edge.to)),
  )
  const facing = borders.filter((border) => touching.has(edgeKey(border.from, border.to)))
  const link = nearestBorder(facing.length > 0 ? facing : borders, origin)
  return link ? { supported, value, borders, link, ...place } : null
}

type Cubic = [Point, Point, Point, Point]

/** A point of a cubic Bézier curve, at `t` in [0, 1]. */
export function bezierPoint([a, b, c, d]: Cubic, t: number): Point {
  const u = 1 - t
  return [
    u ** 3 * a[0] + 3 * u ** 2 * t * b[0] + 3 * u * t ** 2 * c[0] + t ** 3 * d[0],
    u ** 3 * a[1] + 3 * u ** 2 * t * b[1] + 3 * u * t ** 2 * c[1] + t ** 3 * d[1],
  ]
}

/**
 * The tributary of an offensive support: two cubics joined smoothly at the
 * border crossing, then a straight stretch to the tip of the attack arrow.
 */
export interface AffluentPath {
  /** Inside the supporting army's territory: from the army to the border crossing. */
  curves: [Cubic, Cubic]
  /** Tip of the affluent, exactly the tip of the attack arrow it backs. */
  tail: Point
}

/**
 * The tributary of an offensive support. It starts on the supporting army,
 * crosses the border it shares with the attacked territory, and ends exactly
 * on the tip of the attack it backs, tangent to that arrow's vector and with
 * a straight last third. Each half is a cubic Bézier whose control points lie
 * in one territory only; territories are convex, so the curve never strays
 * over a third one (it bends more, rather than overflow).
 */
export function affluentCurve(support: OffensiveSupport): AffluentPath {
  const tail = support.arrowEnd
  const crossing = support.start
  // The last stretch runs straight along the arrow, a third of the way from
  // the border to the centre.
  const end = lerp(tail, support.arrowEntry, 1 / 3)
  const lead = lerp(support.arrowEntry, end, 0.4)

  // First half, in the supporter's territory: the largest bow that fits.
  const chord: Point = [crossing[0] - support.origin[0], crossing[1] - support.origin[1]]
  const chordLength = distanceBetween(support.origin, crossing)
  const normal: Point =
    chordLength > 0 ? [-chord[1] / chordLength, chord[0] / chordLength] : [0, 0]
  const inOrigin = (point: Point) => pointInPolygon(point, support.originArea)
  const shape = (height: number): [Point, Point] => [
    [
      support.origin[0] + chord[0] * 0.3 + normal[0] * height,
      support.origin[1] + chord[1] * 0.3 + normal[1] * height,
    ],
    [
      support.origin[0] + chord[0] * 0.75 + normal[0] * height * 0.6,
      support.origin[1] + chord[1] * 0.75 + normal[1] * height * 0.6,
    ],
  ]
  let first: [Point, Point] = [
    lerp(support.origin, crossing, 0.33),
    lerp(support.origin, crossing, 0.67),
  ]
  for (const factor of [0.7, 0.5, 0.35, 0.2, 0.1]) {
    const fitting = [1, -1]
      .map((sign) => shape(sign * factor * chordLength))
      .filter(([near, far]) => inOrigin(near) && inOrigin(far))
    if (fitting.length === 0) continue
    // Between two sides, the one away from the attacked centre reads best.
    const away = fitting.find(
      ([near]) =>
        (near[0] - support.origin[0]) * (support.origin[0] - tail[0]) +
          (near[1] - support.origin[1]) * (support.origin[1] - tail[1]) >=
        0,
    )
    first = away ?? fitting[0]!
    break
  }

  // Second half, in the attacked territory: leaves the border along the
  // first half's direction (a smooth join), then runs into the arrow.
  const heading: Point = [crossing[0] - first[1][0], crossing[1] - first[1][1]]
  const headingLength = Math.hypot(heading[0], heading[1]) || 1
  const reach = distanceBetween(lead, crossing)
  let departure: Point = crossing
  for (const factor of [0.4, 0.3, 0.2, 0.1, 0.05]) {
    const candidate: Point = [
      crossing[0] + (heading[0] / headingLength) * reach * factor,
      crossing[1] + (heading[1] / headingLength) * reach * factor,
    ]
    if (pointInPolygon(candidate, support.area)) {
      departure = candidate
      break
    }
  }

  return {
    curves: [
      [support.origin, first[0], first[1], crossing],
      [crossing, departure, lead, end],
    ],
    tail,
  }
}

/**
 * Redraws the intentions from where the armies stand: every leg leaves its
 * army's marker instead of the centre of its territory, and the supports are
 * rebuilt around those positions. `anchors` maps a territory to its army.
 */
export function anchorIntentions(
  map: MapData,
  intentions: Intention[],
  anchors: Map<string, Point>,
): Intention[] {
  return intentions.map((intention) => {
    const anchor = anchors.get(intention.armyTerritory)
    if (!anchor) return intention
    return {
      ...intention,
      from: anchor,
      segments: intention.segments.map((segment) => {
        const moved: IntentSegment = {
          ...segment,
          from: anchor,
          to: segment.kind === 'loop' ? anchor : segment.to,
        }
        const { defensive, offensive } = segment
        if (defensive) {
          const rebuilt = defensiveSupportGeometry(
            map,
            intention.armyTerritory,
            defensive.supported,
            defensive.value,
            anchor,
            { slot: defensive.slot, slots: defensive.slots },
          )
          return { ...moved, defensive: rebuilt ?? defensive }
        }
        if (offensive) {
          const supportedFrom =
            anchors.get(offensive.supported) ??
            territoryCentroid(map, offensive.supported) ??
            offensive.arrowEntry
          const rebuilt = offensiveSupportGeometry(
            map,
            intention.armyTerritory,
            offensive.supported,
            offensive.destination,
            offensive.value,
            anchor,
            supportedFrom,
          )
          return { ...moved, offensive: rebuilt ?? offensive }
        }
        return moved
      }),
    }
  })
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

/**
 * The legs of an order that splits an army: a loop for the troops that stay on
 * the origin, then one arrow per distinct destination, in order. `extras`
 * adds what a leg carries. Null when a destination is unknown or not adjacent.
 */
function splitSegments(
  map: MapData,
  position: string,
  targets: string[],
  holdsGround: boolean,
  extras: (target: string) => Partial<IntentSegment>,
): IntentSegment[] | null {
  const from = territoryCentroid(map, position)
  if (!from) return null
  for (const target of targets) {
    if (!known(map, target)) return null
    if (target !== position && !isAdjacent(map, position, target)) return null
  }
  const segments: IntentSegment[] = []
  const seen = new Set<string>()
  if (holdsGround) {
    segments.push({ from, to: from, kind: 'loop', ...extras(position) })
    seen.add(position)
  }
  for (const target of targets) {
    if (seen.has(target)) continue
    seen.add(target)
    const to = territoryCentroid(map, target)
    if (!to) return null
    segments.push({ from, to, kind: 'movement', ...extras(target) })
  }
  return segments.length > 0 ? segments : null
}

function makeIntention(
  map: MapData,
  order: Order,
  territoryState: { id: string; size: number },
  turn: number,
  turnUncertain: boolean,
  source: IntentionSource,
  nobleCode?: string,
  color?: string,
  commanded = false,
): Intention | null {
  const position = order.position
  const from = territoryCentroid(map, position)
  if (!from) return null

  const base: Omit<Intention, 'type' | 'segments'> = {
    armyTerritory: territoryState.id,
    from,
    symbol: SYMBOLS[order.type],
    turn,
    turnLabel: turnUncertain || order.liaison === 'loop' ? '?' : String(turn),
    source,
    color,
    nobleCode,
    label: formatOrderLabel({
      type: order.type,
      position: order.position,
      targets: order.targets,
      counts: order.counts,
      nobleAssignments: order.nobleAssignments,
      liaison: order.liaison,
      amount: order.amount,
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
      return {
        ...base,
        type: order.type,
        segments: [{ from, to, kind: 'attack' }],
      }
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
      const supportValue = territoryState.size + (commanded ? 1 : 0)
      const supported = order.targets?.[0]
      const destination = order.targets?.[1]
      if (!supported || !known(map, supported)) return null
      if (supported === position) return null
      if (destination) {
        // The engine asks the supporter and the supported army to touch the
        // destination, not to touch each other.
        if (!known(map, destination) || destination === position) return null
        if (!isAdjacent(map, position, destination)) return null
        if (!isAdjacent(map, supported, destination)) return null
        const to =
          sharedBorderMidpoint(map, supported, destination) ??
          territoryCentroid(map, destination)
        const supportedFrom = territoryCentroid(map, supported)
        const offensive =
          supportedFrom &&
          offensiveSupportGeometry(
            map,
            position,
            supported,
            destination,
            supportValue,
            from,
            supportedFrom,
          )
        if (!to || !offensive) return null
        return {
          ...base,
          type: order.type,
          segments: [{ from, to, kind: 'support-offensive', offensive }],
        }
      }
      if (!isAdjacent(map, position, supported)) return null
      const to = territoryCentroid(map, supported)
      if (!to) return null
      const defensive = defensiveSupportGeometry(
        map,
        position,
        supported,
        supportValue,
        from,
      )
      return {
        ...base,
        type: order.type,
        segments: [
          { from, to, kind: 'support-defensive', ...(defensive && { defensive }) },
        ],
      }
    }
    case 'disperse': {
      const targets = order.targets ?? []
      const segments = splitSegments(
        map,
        position,
        targets,
        targets.includes(position) || territoryState.size > targets.length,
        (target) => {
          const nobles = order.nobleAssignments?.[target]
          return nobles && nobles.length > 0 ? { nobles } : {}
        },
      )
      return segments && { ...base, type: order.type, segments }
    }
    case 'maneuver': {
      // The server plans where troops, nobles and the order sheet go.
      const segments: IntentSegment[] = []
      for (const leg of order.legs ?? []) {
        const stays = leg.target === position
        if (
          !known(map, leg.target) ||
          (!stays && !isAdjacent(map, position, leg.target))
        ) {
          return null
        }
        const to = stays ? from : territoryCentroid(map, leg.target)
        if (!to) return null
        segments.push({
          from,
          to,
          kind: stays ? 'loop' : 'movement',
          ...((leg.nobles?.length || leg.anyNobles) && {
            nobles: [...(leg.nobles ?? []), ...(leg.anyNobles ? ['*'] : [])],
          }),
          ...(leg.carriesChain && { carriesChain: true }),
        })
      }
      return segments.length > 0 ? { ...base, type: order.type, segments } : null
    }
    case 'transfer': {
      const target = order.targets?.[0]
      if (!target || !known(map, target) || target === position) return null
      const to = territoryCentroid(map, target)
      if (!to) return null
      return { ...base, type: order.type, segments: [{ from, to, kind: 'transfer' }] }
    }
    case 'hold':
    case 'pillage': {
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
  source: IntentionSource,
  nobleCode?: string,
  color?: string,
  commanded = false,
): void {
  let loopPending = false
  for (let orderIndex = startIndex; orderIndex < orders.length; orderIndex += 1) {
    const order = orders[orderIndex]
    const intention = makeIntention(
      map,
      order,
      army,
      orderIndex - startIndex + 1,
      loopPending,
      source,
      nobleCode,
      color,
      commanded,
    )
    if (intention) intentions.push(intention)
    if (order.liaison === 'loop') loopPending = true
  }
}

export function buildIntentions(
  map: MapData,
  state: StateData,
  player: PlayerId,
  /** Drafted orders per noble code, as parsed by the server preview. */
  draftOrders: Record<string, Order[]>,
  options: {
    includeInstalled?: boolean
    includeInstalledInWinter?: boolean
    source?: IntentionSource
    color?: string
  } = {},
): Intention[] {
  const winterInstalledOnly =
    state.season === 'winter' && options.includeInstalledInWinter === true
  if (state.season === 'winter' && !winterInstalledOnly) return []

  const intentions: Intention[] = []
  if (options.includeInstalled !== false) {
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
        'chain',
        undefined,
        options.color,
        Boolean(chain.noble),
      )
    }
  }

  const ownedNobleCodes = new Set(
    state.nobles
      .filter((noble) => noble.owner === player && noble.status !== 'dungeon')
      .map((noble) => noble.code),
  )

  if (!winterInstalledOnly) {
    for (const [nobleCode, parsedOrders] of Object.entries(draftOrders)) {
      if (!ownedNobleCodes.has(nobleCode) || parsedOrders.length === 0) continue
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
        options.source ?? 'draft',
        nobleCode,
        options.color,
        true,
      )
    }
  }

  return assignSupportSlots(intentions)
}

/**
 * Places the defensive supports that share a territory side by side, their
 * colours alternating. Pure and idempotent, so it can run again on intentions merged from several players.
 */
export function assignSupportSlots(intentions: Intention[]): Intention[] {
  const groups = new Map<string, Array<{ intention: number; segment: number }>>()
  intentions.forEach((intention, i) => {
    intention.segments.forEach((segment, j) => {
      if (!segment.defensive) return
      const key = segment.defensive.supported
      groups.set(key, [...(groups.get(key) ?? []), { intention: i, segment: j }])
    })
  })

  const placed = new Map<string, { slot: number; slots: number }>()
  for (const members of groups.values()) {
    const byColor = new Map<string, typeof members>()
    for (const member of members) {
      const color = intentions[member.intention].color ?? ''
      byColor.set(color, [...(byColor.get(color) ?? []), member])
    }
    const queues = [...byColor.values()]
    const ordered: typeof members = []
    while (queues.some((queue) => queue.length > 0)) {
      for (const queue of queues) {
        const next = queue.shift()
        if (next) ordered.push(next)
      }
    }
    ordered.forEach((member, slot) =>
      placed.set(`${member.intention}:${member.segment}`, {
        slot,
        slots: ordered.length,
      }),
    )
  }
  if (placed.size === 0) return intentions

  return intentions.map((intention, i) => ({
    ...intention,
    segments: intention.segments.map((segment, j) => {
      const place = placed.get(`${i}:${j}`)
      if (!place || !segment.defensive) return segment
      return { ...segment, defensive: { ...segment.defensive, ...place } }
    }),
  }))
}
