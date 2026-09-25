export type Terrain = 'plain' | 'forest' | 'hill' | 'mountain' | 'swamp'

export type Season = 'spring' | 'summer' | 'autumn' | 'winter'

export type InfraType = 'mill' | 'supply_depot' | 'castle' | 'village'

export type NobleStatus = 'free' | 'hostage' | 'dungeon'

export type CardKind =
  'fair_weather' | 'abundant_harvest' | 'revolt' | 'plague' | 'bad_weather' | 'famine'

export type OrderType =
  'attack' | 'support' | 'hold' | 'join' | 'pillage' | 'disperse' | 'transfer'

export type LiaisonMode = 'single' | 'loop'

export type Outcome = 'success' | 'failure' | 'invalid'

export type Progression = 'advanced' | 'retried' | 'broken' | 'consumed'

export type EventType =
  | 'movement'
  | 'fusion'
  | 'dispersion'
  | 'pillage'
  | 'retreat'
  | 'army_destroyed'
  | 'control_changed'
  | 'noble_movement'
  | 'capture'
  | 'liberation'
  | 'transfer'
  | 'deck_draw'
  | 'deck_discard'
  | 'deck_restore'
  | 'calamity_scheduled'
  | 'deck_order_played'
  | 'calamity_applied'
  | 'calamity_canceled'
  | 'bonus_effect'
  | 'neutral_army_created'
  | 'plague_noble_death'
  | 'plague_noble_survived'
  | 'bad_weather_blocked'
  | 'famine_loss'
  | 'bad_weather_loss'
  | 'famine'
  | 'card_canceled'
  | 'rumor'

export type PlayerId = string

export const NEUTRAL_PLAYER_ID = 'NEUTRAL'

export type GameStatus = 'playing' | 'finished'

export interface ProfileData {
  uid: string
  email: string
  displayName: string
}

export interface GameSlot {
  id: PlayerId
  name: string
  color: string
  submitted: boolean
  /** Whether the current turn is still awaiting this player, per the server
   * (an eliminated player, or one with nothing left to submit this turn, is
   * never required even before they act). */
  required: boolean
  actorId?: string
}

export interface GameSummary {
  id: string
  name: string
  seed: string
  status: GameStatus
  winner?: PlayerId | null
  currentPlayer?: PlayerId
  canInvite?: boolean
  inviteAvailable?: boolean
  spectator?: boolean
  players: GameSlot[]
  turn: number
  season: Season
  yearCount?: number
  scores?: Record<PlayerId, ScoreBreakdown>
  revision: number
  updatedAt?: string
}

export interface GameViewDocument {
  gameId: string
  uid: string
  revision: number
  turn: number
  season: Season
  state: StateData
  updatedAt?: string
}

export type Point = [number, number]

export interface Army {
  owner: PlayerId
  size: number
  chain: Chain | null
}

export interface Order {
  type: OrderType
  position: string
  targets?: string[]
  nobleAssignments?: Record<string, string[]>
  liaison: LiaisonMode
  amount?: number
}

export interface Chain {
  visibility?: 'known' | 'hidden'
  noble?: string
  currentIndex?: number
  orders?: Order[]
}

export interface Infrastructure {
  type: InfraType
  level: number
}

export interface Noble {
  id: string
  code: string
  name: string
  owner: PlayerId
  location: string
  status: NobleStatus
}

export interface Territory {
  id: string
  name: string
  terrain: Terrain
  village: boolean
  points: Point[]
  adjacencies: string[]
  impassable: string[]
}

export interface Region {
  id: string
  seed: string
  territories: string[]
}

export interface MapData {
  territories: Territory[]
  regions?: Region[]
}

export interface Player {
  id: PlayerId
  name: string
  color: string
  capitalTerritory?: string
  /** Territory income projected for the next action turn (never in winter). */
  projectedIncome?: number
  /**
   * Mill production projected for the next action turn, credited to the
   * player's own settlements and self-supplied mills (never in winter).
   * Since issue #195, each mill credits exactly one destination (its
   * adjacent castle under the same control, else its adjacent village, else
   * itself), so this is the sum of what the player's own territories will
   * receive.
   */
  projectedMillIncome?: number
  /**
   * Net rations every army the player controls will draw from stock or the
   * supply network beyond what its own territory already produces for it,
   * projected for the next action turn (never populated in winter).
   */
  projectedConsumption?: number
  /**
   * Armies that would starve next action turn if nothing changes before
   * resolution: an estimate only because orders aren't submitted yet and an
   * undrawn calamity card is never reflected (see ArmyRisk).
   */
  armiesAtRisk?: ArmyRisk[]
}

/**
 * One army the famine risk forecast flags as starving, addressed by its
 * territory like the rest of the app addresses armies. Deficit is the
 * ration shortfall that goes unmet, not a troop count (an actual famine
 * costs 1 troop, regardless of the deficit's size).
 */
export interface ArmyRisk {
  territoryId: string
  size: number
  deficit: number
}

export interface ScoreBreakdown {
  territories: number
  villages: number
  mills: number
  castles: number
  nobles: number
  troops: number
  resources: number
  total: number
}

export interface TerritoryState {
  id: string
  owner: PlayerId | null
  resources: number
  army: Army | null
  infrastructures: Infrastructure[]
  /** Territory income this territory would yield next action turn. */
  projectedIncome?: number
  /** Where that income would land: the owner's capital or its fallback. */
  incomeDestination?: string
  /** Present only on a mill's own territory: its projected production. */
  millProduction?: number
  /**
   * Present only on a mill's own territory: where that production would
   * land (its adjacent castle under the same control, else its adjacent
   * village, else itself, in which case this equals `id`).
   */
  millDestination?: string
}

export interface StateData {
  turn: number
  season: Season
  year?: number
  yearCount?: number
  scores?: Record<PlayerId, ScoreBreakdown>
  finished?: boolean
  winner?: PlayerId | null
  players: Player[]
  territories: TerritoryState[]
  nobles: Noble[]
  specialHand?: CardKind[]
  activeRegionEffects?: ActiveRegionEffect[]
  announcements?: AnnouncementReport[]
}

export interface ActiveRegionEffect {
  kind: CardKind
  regionSeed: string
  season: Season
  year: number
}

export interface AnnouncementReport {
  kind: CardKind
  season: Season
  region: string
  year: number
}

export interface WinterCosts {
  castle: number
  millLevels: number[]
  troop: number
  noble: number
  supplyDepot: number
  liberation: number
}

export interface SupplyLine {
  kind: 'army' | 'source'
  territory: string
  armyOwner: PlayerId
  armySize: number
  terrainProduction: number
  /** Terrain rations removed by the current season's famine. */
  famineRations?: number
  /** Rations added by a regional bonus card. */
  bonusRations?: number
  localProduction: number
  rations: number
  totalDemand: number
  demand: number
  source: string | null
  distance: number
  path: string[]
  reachable: string[]
  selfSupplied: boolean
}

export interface TransferLine {
  kind: 'transfer'
  source: string
  target: string
  armyOwner: PlayerId
  recipientArmy?: string
  path: string[]
  reachable: boolean
  distance?: number
  reachableTerritories: string[]
}

export interface ChainSubmission {
  player: PlayerId
  noble: string
  text: string
}

export interface WinterSubmission {
  player: PlayerId
  lines: string
}

export interface DeckSubmission {
  text: string
}

export interface OrdersInput {
  chains: ChainSubmission[]
  winter: WinterSubmission[]
  special: DeckSubmission[]
}

export interface SubmittedChain {
  noble: string
  text: string
}

export interface SubmittedWinter {
  lines: string
}

export interface MySubmissionResponse {
  turn: number
  season: Season
  submitted: boolean
  chains: SubmittedChain[]
  winter?: SubmittedWinter
}

export interface SubmittedPlayerOrders {
  player: PlayerId
  chains: SubmittedChain[]
  winter?: SubmittedWinter
  /** Server dry run of this submission, used to draw the observer overlay. */
  preview?: OrdersPreview
}

/** Error found by the server in a draft; line numbers include the noble header. */
export interface OrdersPreviewError {
  player?: PlayerId
  noble?: string
  line?: number
  code: string
  message: string
}

/** Order lines of one drafted chain that the server could parse. */
export interface ChainPreview {
  noble: string
  orders: Order[]
}

export type WinterLineStatus = 'applied' | 'rejected' | 'invalid' | 'discard'

/**
 * Simulated outcome of one winter line. `reason` is an engine rejection
 * reason; `message` explains a line that does not parse.
 */
export interface WinterLinePreview {
  line: number
  status: WinterLineStatus
  type?: WinterOrderType
  territory?: string
  source?: string
  target?: string
  amount?: number
  infrastructure?: 'mill' | 'castle' | 'supply_depot'
  level?: number
  noble?: string
  cost?: number
  reason?: string
  message?: string
}

/** Server dry run of the current player's draft (`POST .../orders/preview`). */
export interface OrdersPreview {
  errors: OrdersPreviewError[]
  chains: ChainPreview[]
  winter: WinterLinePreview[]
  winterCost?: { spent: number; available: number }
}

export interface SubmittedOrdersResponse {
  turn: number
  season: Season
  submissions: SubmittedPlayerOrders[]
}

export interface OrdersResponse {
  status: 'pending' | 'resolved'
  player?: PlayerId
  submitted: PlayerId[]
  remaining: PlayerId[]
  report?: TurnReport
  state: StateData
  revision?: number
  resolved?: boolean
  forced?: boolean
}

export interface ReceptionReport {
  player: PlayerId
  noble: string
  received: boolean
  reason?: string
  reasonKey?: string
  reasonArgs?: unknown[]
}

export interface ReportHeader {
  year: number
  season: Season
  turn: number
}

export interface ReportArmy {
  id: string
  owner: PlayerId
  territory: string
  size: number
}

export interface ReportNoble {
  kind?: EventType
  noble: string
  code?: string
  name?: string
  owner?: PlayerId
  army?: string
  territory?: string
  source?: string
  destination?: string
  previousStatus?: NobleStatus
  status?: NobleStatus
  captor?: PlayerId
}

export interface ReportInfrastructure {
  id: string
  type: InfraType
  level: number
  territory: string
}

export interface PlayerReport {
  id: PlayerId
  name: string
  scoreBefore?: ScoreBreakdown
  scoreAfter?: ScoreBreakdown
  resourcesBefore: number
  resourcesAfter: number
  controlledBefore: number
  controlledAfter: number
  armies: ReportArmy[]
  nobles: ReportNoble[]
  infrastructures: ReportInfrastructure[]
}

export interface IncomeReport {
  owner: PlayerId
  destination?: string
  territories: number
  villages: number
  base: number
  bonus?: number
  suppressed?: number
  credited: number
  stockAfter?: number
  lost?: boolean
}

/**
 * One mill's harvest-and-weather-adjusted production and its single
 * beneficiary this turn (see issue #195): the adjacent castle or village
 * under the mill's own control, or the mill's own territory when none
 * qualifies (in which case `destination` equals `territory`). `suppressed`
 * is the production lost to bad weather instead.
 */
export interface MillReport {
  territory: string
  owner?: PlayerId
  level: number
  destination: string
  production: number
  bonus?: number
  suppressed?: number
}

export interface ProductionReport {
  territory: string
  region?: string
  owner?: PlayerId
  terrainRations: number
  bonusRations?: number
  suppressedRations?: number
  baseProduction?: number
  millProduction?: number
  bonusProduction?: number
  suppressedProduction?: number
  produced: number
  sentToRations?: Record<string, number>
  stockBefore?: number
  stockConsumed?: number
  stockAfter?: number
}

export interface ConsumptionReport {
  army: string
  owner: PlayerId
  territory: string
  source?: string
  size: number
  demand: number
  receivedLocal: number
  receivedTransfer: number
  totalReceived: number
  missing: number
  famine?: boolean
  savedByPillage?: boolean
  troopsLost?: number
  pillageInfrastructure?: InfraType
  resourceCredit?: number
  creditTerritory?: string
}

export interface CombatContender {
  army?: string
  owner?: PlayerId
  force?: number
  nobleBonus?: number
  defender?: boolean
}

export interface CombatReport {
  visibility?: 'exact' | 'general'
  territory: string
  baseDefense?: number
  defense?: number
  castleBonus?: number
  contenders?: CombatContender[]
  supporters?: string[]
  winner?: string
  dislodged?: string
  cutSupporters?: string[]
  reason?: string
  standoff?: boolean
  outcome?: string
  summary?: string
}

export interface OrderReport {
  visibility?: 'known' | 'hidden'
  army?: string
  chain?: string
  order?: string
  owner?: PlayerId
  noble?: string
  type?: OrderType
  source?: string
  target?: string
  targets?: string[]
  amount?: number
  nobleAssignments?: Record<string, string[]>
  liaison?: LiaisonMode
  outcome: Outcome
  reason?: string
  progression?: Progression
  indexBefore?: number
  indexAfter?: number
}

export interface MoveReport {
  kind: EventType
  army?: string
  otherArmy?: string
  armies?: string[]
  territory?: string
  source?: string
  target?: string
  destination?: string
  orderType?: OrderType
  outcome?: Outcome
  reason?: string
  resolved?: boolean
  infrastructure?: string
  infrastructureType?: InfraType
  resourceCredit?: number
  creditTerritory?: string
  resourceAmount?: number
  partial?: boolean
  previousOwner?: PlayerId
  owner?: PlayerId
}

export interface WinterInvestmentReport {
  kind: string
  player: PlayerId
  outcome: Outcome
  cost: number
  source?: string
  target?: string
  amount?: number
  territory?: string
  infrastructure?: string
  type?: InfraType
  level?: number
  noble?: string
  nobleCode?: string
  nobleName?: string
  reason?: string
  order?: WinterOrder
}

export type WinterOrderType =
  | 'recruit_noble'
  | 'recruit_troop'
  | 'build'
  | 'elect_capital'
  | 'liberate_noble'
  | 'hostage'
  | 'dungeon'
  | 'transfer'

export interface WinterOrder {
  id?: string
  type: WinterOrderType
  territory?: string
  infrastructureType?: InfraType
  nobleCode?: string
  source?: string
  target?: string
  amount?: number
}

export interface WinterStockReport {
  territory: string
  owner?: PlayerId
  stockBefore: number
  stockAfter: number
}

export interface CardReport {
  kind: CardKind
  eventType: EventType
  player?: PlayerId
  region?: string
  season?: Season
  outcome: Outcome
  reason?: string
}

export interface RumorReport {
  kind: CardKind
  key: string
  level?: number
}

export interface SeasonEffectReport {
  kind: EventType
  cardKind?: CardKind
  region?: string
  season?: Season
  owner?: PlayerId
  army?: string
  noble?: string
  territory?: string
  target?: string
  troops?: number
  sizeBefore?: number
  sizeAfter?: number
  productionLost?: number
  rationsLost?: number
  reason?: string
}

export interface AuguryReport {
  year: number
  capacities: Partial<Record<Season, number>>
  calamities: Array<{ kind: CardKind; season: Season; region: string }>
}

export interface WinterReport {
  investments: WinterInvestmentReport[]
  stocks: WinterStockReport[]
  cards?: CardReport[]
  rumors?: RumorReport[]
}

export interface TurnReport {
  header: ReportHeader
  players: PlayerReport[]
  receptions: ReceptionReport[]
  income?: IncomeReport[]
  mills?: MillReport[]
  production: ProductionReport[]
  consumption: ConsumptionReport[]
  combats: CombatReport[]
  orders: OrderReport[]
  moves: MoveReport[]
  nobles: ReportNoble[]
  seasonEffects?: SeasonEffectReport[]
  rumors?: RumorReport[]
  cards?: CardReport[]
  announcements?: AnnouncementReport[]
  augury?: AuguryReport
  winter?: WinterReport
}
