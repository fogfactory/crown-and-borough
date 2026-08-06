export type Terrain = 'plain' | 'forest' | 'hill' | 'mountain' | 'swamp'

export type Season = 'spring' | 'summer' | 'autumn' | 'winter'

export type InfraType =
  'mill' | 'post_relay' | 'watchtower' | 'supply_depot' | 'castle' | 'village'

export type NobleStatus = 'free' | 'hostage' | 'dungeon'

export type OrderType =
  | 'attack'
  | 'support'
  | 'hold'
  | 'join'
  | 'pillage'
  | 'disperse'
  | 'hostage'
  | 'dungeon'

export type LiaisonMode = 'single' | 'loop'

export type PlayerId = string

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
  nobleTargets?: string[]
  nobleAssignments?: Record<string, string[]>
  liaison: LiaisonMode
}

export interface Chain {
  noble: string
  currentIndex: number
  orders: Order[]
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
  code: string
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
  asOf: Record<string, number>
  territories: TerritoryState[]
  nobles: Noble[]
}
