export type Terrain = 'plain' | 'forest' | 'hill' | 'mountain' | 'swamp'

export type Season = 'spring' | 'summer' | 'autumn' | 'winter'

export type InfraType = 'mill' | 'post_relay' | 'watchtower' | 'supply_depot' | 'castle' | 'village'

export type PlayerId = string

export type Point = [number, number]

export interface Troop {
  id: string
  owner: PlayerId
}

export interface Infrastructure {
  type: InfraType
  level: number
}

export interface Noble {
  id: string
  name: string
  owner: PlayerId
  location: string
}

export interface Territory {
  id: string
  code: string
  name: string
  terrain: Terrain
  village: boolean
  points: Point[]
  adjacencies: string[]
}

export interface MapData {
  territories: Territory[]
}

export interface TerritoryState {
  id: string
  owner: PlayerId | null
  resources: number
  troops: Troop[]
  infrastructures: Infrastructure[]
}

export interface StateData {
  turn: number
  season: Season
  asOf: Record<string, number>
  territories: TerritoryState[]
  nobles: Noble[]
}
