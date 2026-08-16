export type Terrain = 'plain' | 'forest' | 'hill' | 'mountain' | 'swamp'

export type Season = 'spring' | 'summer' | 'autumn' | 'winter'

export type InfraType = 'mill' | 'supply_depot' | 'castle' | 'village'

export type NobleStatus = 'free' | 'hostage' | 'dungeon'

export type OrderType = 'attack' | 'support' | 'hold' | 'join' | 'pillage' | 'disperse'

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

export type PlayerId = string

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
  players: GameSlot[]
  turn: number
  season: Season
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

export interface MapData {
  territories: Territory[]
}

export interface Player {
  id: PlayerId
  name: string
  color: string
  capitalTerritory?: string
}

export interface TerritoryState {
  id: string
  owner: PlayerId | null
  resources: number
  army: Army | null
  infrastructures: Infrastructure[]
}

export interface StateData {
  turn: number
  season: Season
  players: Player[]
  territories: TerritoryState[]
  nobles: Noble[]
}

export interface SupplyLine {
  kind: 'army' | 'source'
  territory: string
  armyOwner: PlayerId
  armySize: number
  rations: number
  demand: number
  source: string | null
  distance: number
  path: string[]
  reachable: string[]
  selfSupplied: boolean
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

export interface OrdersInput {
  chains: ChainSubmission[]
  winter: WinterSubmission[]
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
  resourcesBefore: number
  resourcesAfter: number
  controlledBefore: number
  controlledAfter: number
  armies: ReportArmy[]
  nobles: ReportNoble[]
  infrastructures: ReportInfrastructure[]
}

export interface SupplyReport {
  source: string
  owner: PlayerId
  production: number
  demand: number
  rations: Record<string, number>
  stockConsumed: number
  stockAfter: number
}

export interface FamineReport {
  army: string
  owner: PlayerId
  territory: string
  source: string
  troops: number
  troopsLost?: number
  savedByPillage: boolean
  infrastructure?: string
  infrastructureType?: InfraType
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
  previousOwner?: PlayerId
  owner?: PlayerId
}

export interface WinterInvestmentReport {
  kind: string
  player: PlayerId
  outcome: Outcome
  cost: number
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

export interface WinterOrder {
  id?: string
  type: WinterOrderType
  territory?: string
  infrastructureType?: InfraType
  nobleCode?: string
}

export interface WinterStockReport {
  territory: string
  owner?: PlayerId
  stockBefore: number
  stockAfter: number
}

export interface WinterReport {
  investments: WinterInvestmentReport[]
  stocks: WinterStockReport[]
}

export interface TurnReport {
  header: ReportHeader
  players: PlayerReport[]
  receptions: ReceptionReport[]
  supply: SupplyReport[]
  famines: FamineReport[]
  combats: CombatReport[]
  orders: OrderReport[]
  moves: MoveReport[]
  nobles: ReportNoble[]
  winter?: WinterReport
}
