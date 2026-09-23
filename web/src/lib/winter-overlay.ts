import {
  parseWinterDraftDetailed,
  type ParsedWinterOrder,
  type WinterParseError,
} from '@/lib/winter-parse'
import type {
  Infrastructure,
  MapData,
  NobleStatus,
  PlayerId,
  StateData,
  WinterCosts,
} from '@/types'

export type WinterIntentionKind =
  | 'build'
  | 'recruit_troop'
  | 'recruit_noble'
  | 'liberate'
  | 'hostage'
  | 'dungeon'
  | 'capital'
  | 'transfer'
  | 'error'

export type WinterIntentionSource = 'draft' | 'submitted'

export interface WinterIntention {
  kind: WinterIntentionKind
  line: number
  valid: boolean
  source: WinterIntentionSource
  color?: string
  territory?: string
  sourceTerritory?: string
  targetTerritory?: string
  amount?: number
  noble?: string
  infrastructure?: 'mill' | 'castle' | 'supply_depot'
  level?: number
  reason?: string
  reasonValues?: Record<string, string | number>
  warning?: boolean
  label: string
}

export interface WinterSimulationOutcome {
  line: number
  order?: ParsedWinterOrder
  valid: boolean
  reason?: string
  reasonValues?: Record<string, string | number>
  territory?: string
  level?: number
  warning?: boolean
}

export interface WinterSimulationResult {
  outcomes: WinterSimulationOutcome[]
}

export interface WinterOverlayOptions {
  source?: WinterIntentionSource
  color?: string
  costs?: WinterCosts | null
}

interface SimulatedArmy {
  owner: PlayerId
  size: number
}

interface SimulatedNoble {
  owner: PlayerId
  status: NobleStatus
  location: string
}

interface SimulationContext {
  state: StateData
  player: PlayerId
  map: MapData
  costs?: WinterCosts | null
  infrastructure: Map<string, Infrastructure>
  armies: Map<string, SimulatedArmy>
  nobles: Map<string, SimulatedNoble>
  resources: Map<string, number>
  capitals: Map<PlayerId, string | undefined>
}

function hasSettlement(infrastructure: Infrastructure | undefined): boolean {
  return infrastructure?.type === 'castle' || infrastructure?.type === 'village'
}

function territoryState(context: SimulationContext, id: string | undefined) {
  return id
    ? context.state.territories.find((territory) => territory.id === id)
    : undefined
}

function territoryExists(context: SimulationContext, id: string | undefined): boolean {
  return Boolean(id && territoryState(context, id))
}

function controlsTerritory(context: SimulationContext, id: string | undefined): boolean {
  return territoryState(context, id)?.owner === context.player
}

function isAdjacent(context: SimulationContext, first: string, second: string): boolean {
  return (
    context.map.territories
      .find((territory) => territory.id === first)
      ?.adjacencies.includes(second) ?? false
  )
}

function hasEligibleTroopNoble(context: SimulationContext, target: string): boolean {
  return Array.from(context.nobles.values()).some(
    (noble) =>
      noble.owner === context.player &&
      noble.status === 'free' &&
      (noble.location === target || isAdjacent(context, noble.location, target)),
  )
}

function paymentSources(context: SimulationContext): string[] {
  return context.state.territories
    .filter(
      (territory) =>
        territory.owner === context.player &&
        hasSettlement(context.infrastructure.get(territory.id)),
    )
    .map((territory) => territory.id)
}

function pay(context: SimulationContext, amount: number): boolean {
  if (amount <= 0) return true
  const sources = paymentSources(context)
  const available = sources.reduce(
    (total, territoryID) => total + Math.max(0, context.resources.get(territoryID) ?? 0),
    0,
  )
  if (available < amount) return false

  let remaining = amount
  for (const territoryID of sources) {
    if (remaining === 0) break
    const stock = Math.max(0, context.resources.get(territoryID) ?? 0)
    const paid = Math.min(stock, remaining)
    context.resources.set(territoryID, stock - paid)
    remaining -= paid
  }
  return true
}

function payOrWarn(context: SimulationContext, amount: number | null): boolean {
  if (amount === null || amount <= 0) return false
  return !pay(context, amount)
}

function costFor(
  context: SimulationContext,
  order: ParsedWinterOrder,
  nextMillLevel?: number,
): number | null {
  const costs = context.costs
  if (!costs) return null

  switch (order.type) {
    case 'recruit_noble':
      return costs.noble
    case 'recruit_troop':
      return costs.troop
    case 'liberate_noble':
      return costs.liberation
    case 'transfer':
      return order.amount ?? 0
    case 'build':
      if (order.infrastructure === 'castle') return costs.castle
      if (order.infrastructure === 'supply_depot') return costs.supplyDepot
      if (nextMillLevel === undefined) return null
      return costs.millLevels[nextMillLevel - 1] ?? null
    case 'elect_capital':
    case 'hostage':
    case 'dungeon':
      return 0
  }
}

function syntaxTerritory(map: MapData, text: string, line: number): string | undefined {
  const rawLine = text.replace(/\r\n/g, '\n').split('\n')[line - 1] ?? ''
  const comment = rawLine.indexOf('#')
  const fields = (comment >= 0 ? rawLine.slice(0, comment) : rawLine)
    .trim()
    .toUpperCase()
    .split(/\s+/)
  return fields.find((field) =>
    map.territories.some((territory) => territory.id === field),
  )
}

function orderLabel(order: ParsedWinterOrder): string {
  switch (order.type) {
    case 'recruit_noble':
      return `R N ${order.territory ?? '???'}`
    case 'recruit_troop':
      return `R T ${order.territory ?? '???'}`
    case 'build':
      return `C ${order.infrastructure === 'mill' ? 'M' : order.infrastructure === 'castle' ? 'C' : 'D'} ${order.territory ?? '???'}`
    case 'elect_capital':
      return `E C ${order.territory ?? '???'}`
    case 'liberate_noble':
      return `L N ${order.noble ?? '???'}`
    case 'hostage':
      return `O N ${order.noble ?? '???'}`
    case 'dungeon':
      return `P N ${order.noble ?? '???'}`
    case 'transfer':
      return `G ${order.source ?? '???'} ${order.target ?? '???'} ${order.amount ?? '???'}`
  }
}

function parseErrorLabel(text: string, line: number): string {
  return text.replace(/\r\n/g, '\n').split('\n')[line - 1]?.trim() || `Line ${line}`
}

function contextFromState(
  state: StateData,
  player: PlayerId,
  map: MapData,
  costs?: WinterCosts | null,
): SimulationContext {
  const infrastructure = new Map<string, Infrastructure>()
  const armies = new Map<string, SimulatedArmy>()
  const resources = new Map<string, number>()
  for (const territory of state.territories) {
    if (territory.infrastructures[0]) {
      infrastructure.set(territory.id, { ...territory.infrastructures[0] })
    }
    if (territory.army) {
      armies.set(territory.id, { owner: territory.army.owner, size: territory.army.size })
    }
    resources.set(territory.id, territory.resources)
  }

  const nobles = new Map<string, SimulatedNoble>()
  for (const noble of state.nobles) {
    nobles.set(noble.code, {
      owner: noble.owner,
      status: noble.status,
      location: noble.location,
    })
  }

  return {
    state,
    player,
    map,
    costs,
    infrastructure,
    armies,
    nobles,
    resources,
    capitals: new Map(
      state.players.map((candidate) => [candidate.id, candidate.capitalTerritory]),
    ),
  }
}

function outcomeError(
  order: ParsedWinterOrder,
  reason: string,
  territory?: string,
): WinterSimulationOutcome {
  return { line: order.line, order, valid: false, reason, territory }
}

function resolveWinterOrder(
  context: SimulationContext,
  order: ParsedWinterOrder,
): WinterSimulationOutcome {
  const territory = order.territory

  switch (order.type) {
    case 'recruit_noble': {
      if (!territory || !territoryExists(context, territory)) {
        return outcomeError(order, 'unknown_territory', territory)
      }
      if (!controlsTerritory(context, territory)) {
        return outcomeError(order, 'territory_not_controlled', territory)
      }
      if (!hasSettlement(context.infrastructure.get(territory))) {
        return outcomeError(order, 'noble_requires_settlement', territory)
      }
      if (context.armies.get(territory)?.owner !== context.player) {
        return outcomeError(order, 'noble_requires_owned_army', territory)
      }
      const cost = costFor(context, order)
      const warning = payOrWarn(context, cost)
      context.nobles.set(`draft-${order.line}`, {
        owner: context.player,
        status: 'free',
        location: territory,
      })
      return {
        line: order.line,
        order,
        valid: true,
        warning,
        reason: warning ? 'insufficient_resources' : undefined,
        territory,
      }
    }
    case 'recruit_troop': {
      if (!territory || !territoryExists(context, territory)) {
        return outcomeError(order, 'unknown_territory', territory)
      }
      if (!controlsTerritory(context, territory)) {
        return outcomeError(order, 'territory_not_controlled', territory)
      }
      const army = context.armies.get(territory)
      if (army && army.owner !== context.player) {
        return outcomeError(order, 'territory_occupied_by_other_player', territory)
      }
      if (!hasEligibleTroopNoble(context, territory)) {
        return outcomeError(order, 'troop_requires_adjacent_noble', territory)
      }
      const cost = costFor(context, order)
      const warning = payOrWarn(context, cost)
      if (army) {
        army.size += 1
      } else {
        context.armies.set(territory, { owner: context.player, size: 1 })
      }
      return {
        line: order.line,
        order,
        valid: true,
        warning,
        reason: warning ? 'insufficient_resources' : undefined,
        territory,
      }
    }
    case 'build': {
      if (!territory || !territoryExists(context, territory)) {
        return outcomeError(order, 'unknown_territory', territory)
      }
      if (!controlsTerritory(context, territory)) {
        return outcomeError(order, 'territory_not_controlled', territory)
      }
      const existing = context.infrastructure.get(territory)
      let nextLevel: number | undefined
      if (order.infrastructure === 'mill') {
        const territoryData = context.map.territories.find(
          (candidate) => candidate.id === territory,
        )
        const productiveNeighbor =
          hasSettlement(existing) ||
          Boolean(
            territoryData?.adjacencies.some((neighbor) =>
              hasSettlement(context.infrastructure.get(neighbor)),
            ),
          )
        if (!productiveNeighbor) {
          return outcomeError(order, 'mill_requires_productive_neighbor', territory)
        }
        if (existing && existing.type !== 'mill') {
          return outcomeError(order, 'structure_present', territory)
        }
        nextLevel = existing ? existing.level + 1 : 1
        const maximum = context.costs?.millLevels.length ?? 3
        if (nextLevel > maximum) {
          return outcomeError(order, 'mill_max_level_reached', territory)
        }
      } else if (order.infrastructure === 'castle') {
        if (existing && existing.type !== 'village') {
          return outcomeError(order, 'structure_present', territory)
        }
      } else if (existing) {
        return outcomeError(order, 'structure_present', territory)
      }

      const cost = costFor(context, order, nextLevel)
      const warning = payOrWarn(context, cost)
      if (order.infrastructure === 'mill') {
        context.infrastructure.set(territory, { type: 'mill', level: nextLevel ?? 1 })
      } else if (order.infrastructure === 'castle') {
        context.infrastructure.set(territory, { type: 'castle', level: 1 })
        if (!context.capitals.get(context.player)) {
          context.capitals.set(context.player, territory)
        }
      } else {
        context.infrastructure.set(territory, { type: 'supply_depot', level: 1 })
      }
      return {
        line: order.line,
        order,
        valid: true,
        warning,
        reason: warning ? 'insufficient_resources' : undefined,
        territory,
        level: nextLevel ?? 1,
      }
    }
    case 'elect_capital': {
      if (!territory || !territoryExists(context, territory)) {
        return outcomeError(order, 'unknown_territory', territory)
      }
      if (!controlsTerritory(context, territory)) {
        return outcomeError(order, 'territory_not_controlled', territory)
      }
      if (context.infrastructure.get(territory)?.type !== 'castle') {
        return outcomeError(order, 'capital_requires_controlled_castle', territory)
      }
      context.capitals.set(context.player, territory)
      return { line: order.line, order, valid: true, territory }
    }
    case 'liberate_noble': {
      const noble = context.nobles.get(order.noble ?? '')
      if (!noble) return outcomeError(order, 'unknown_noble')
      if (noble.status === 'free')
        return outcomeError(order, 'noble_not_prisoner', noble.location)
      if (context.armies.get(noble.location)?.owner !== context.player) {
        return outcomeError(order, 'noble_not_held', noble.location)
      }
      const capital = context.capitals.get(noble.owner)
      if (!capital) return outcomeError(order, 'no_capital', noble.location)
      if (context.armies.get(capital)?.owner !== noble.owner) {
        return outcomeError(order, 'no_army_at_capital', noble.location)
      }
      const cost = costFor(context, order)
      const warning = payOrWarn(context, cost)
      noble.status = 'free'
      noble.location = capital
      return {
        line: order.line,
        order,
        valid: true,
        warning,
        reason: warning ? 'insufficient_resources' : undefined,
        territory: capital,
      }
    }
    case 'hostage':
    case 'dungeon': {
      const noble = context.nobles.get(order.noble ?? '')
      if (!noble) return outcomeError(order, 'unknown_noble')
      if (noble.status === 'free')
        return outcomeError(order, 'noble_not_prisoner', noble.location)
      if (
        context.armies.get(noble.location)?.owner !== context.player ||
        noble.owner === context.player
      ) {
        return outcomeError(order, 'noble_not_held', noble.location)
      }
      noble.status = order.type === 'hostage' ? 'hostage' : 'dungeon'
      return { line: order.line, order, valid: true, territory: noble.location }
    }
    case 'transfer': {
      const source = order.source
      const target = order.target
      if (
        !source ||
        !target ||
        !territoryExists(context, source) ||
        !territoryExists(context, target)
      ) {
        return outcomeError(order, 'unknown_territory', source ?? target)
      }
      if (source === target) {
        return outcomeError(order, 'transfer_same_territory', source)
      }
      if (!order.amount || order.amount < 1) {
        return outcomeError(order, 'invalid_transfer_amount', source)
      }
      if (
        !controlsTerritory(context, source) ||
        !hasSettlement(context.infrastructure.get(source))
      ) {
        return outcomeError(order, 'transfer_source_not_settlement', source)
      }
      const targetState = territoryState(context, target)
      if (
        !targetState?.owner ||
        targetState.owner === context.player ||
        !hasSettlement(context.infrastructure.get(target))
      ) {
        return outcomeError(order, 'transfer_target_not_settlement', target)
      }
      const warning = payOrWarn(context, order.amount)
      context.resources.set(target, (context.resources.get(target) ?? 0) + order.amount)
      return {
        line: order.line,
        order,
        valid: true,
        warning,
        reason: warning ? 'insufficient_resources' : undefined,
        territory: source,
      }
    }
  }
}

export function simulateWinterDraft(
  state: StateData,
  player: PlayerId,
  draft: string,
  map: MapData,
  costs?: WinterCosts | null,
): WinterSimulationResult {
  const parsed = parseWinterDraftDetailed(draft, { map, nobles: state.nobles })
  const context = contextFromState(state, player, map, costs)
  const outcomes: WinterSimulationOutcome[] = parsed.errors.map(
    (error: WinterParseError) => ({
      line: error.line,
      valid: false,
      reason: error.key,
      reasonValues: error.values,
      territory: syntaxTerritory(map, draft, error.line),
    }),
  )

  for (const order of parsed.orders) {
    outcomes.push(resolveWinterOrder(context, order))
  }

  outcomes.sort((first, second) => first.line - second.line)
  return { outcomes }
}

function intentionForOrder(
  outcome: WinterSimulationOutcome,
  source: WinterIntentionSource,
  color: string | undefined,
): WinterIntention {
  const order = outcome.order!
  const common = {
    line: outcome.line,
    valid: true,
    warning: outcome.warning,
    reason: outcome.reason,
    source,
    color,
    label: orderLabel(order),
  }
  switch (order.type) {
    case 'build':
      return {
        ...common,
        kind: 'build',
        territory: order.territory,
        infrastructure: order.infrastructure,
        level: outcome.level,
      }
    case 'recruit_troop':
      return { ...common, kind: 'recruit_troop', territory: order.territory }
    case 'recruit_noble':
      return { ...common, kind: 'recruit_noble', territory: order.territory }
    case 'liberate_noble':
      return {
        ...common,
        kind: 'liberate',
        territory: outcome.territory,
        noble: order.noble,
      }
    case 'hostage':
      return {
        ...common,
        kind: 'hostage',
        territory: outcome.territory,
        noble: order.noble,
      }
    case 'dungeon':
      return {
        ...common,
        kind: 'dungeon',
        territory: outcome.territory,
        noble: order.noble,
      }
    case 'elect_capital':
      return { ...common, kind: 'capital', territory: order.territory }
    case 'transfer':
      return {
        ...common,
        kind: 'transfer',
        sourceTerritory: order.source,
        targetTerritory: order.target,
        amount: order.amount,
      }
  }
}

export function buildWinterIntentions(
  map: MapData,
  state: StateData,
  player: PlayerId,
  draft: string,
  options: WinterOverlayOptions = {},
): WinterIntention[] {
  if (state.season !== 'winter' || !draft.trim()) return []

  const source = options.source ?? 'draft'
  const simulation = simulateWinterDraft(state, player, draft, map, options.costs)
  return simulation.outcomes.map((outcome) => {
    if (!outcome.valid) {
      return {
        kind: 'error',
        line: outcome.line,
        valid: false,
        source,
        color: options.color,
        territory: outcome.territory,
        reason: outcome.reason,
        reasonValues: outcome.reasonValues,
        warning: outcome.warning,
        label: outcome.order
          ? orderLabel(outcome.order)
          : parseErrorLabel(draft, outcome.line),
      }
    }
    return intentionForOrder(outcome, source, options.color)
  })
}
