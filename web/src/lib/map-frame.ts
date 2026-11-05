import type { PlayerId, StateData } from '@/types'

/** Territories to frame on the map; a new `key` frames them again. */
export interface MapFrame {
  territories: string[]
  key: string | number
  /** Margin around the territories, as a share of the map's longer side. */
  padding?: number
}

/** What a report event draws on the map: its territories and its moves. */
export interface EventMarks {
  territories: string[]
  /** Moves from one territory to another (an attack, a support, a retreat). */
  arrows?: Array<[string, string]>
}

/** Margin of a frame on one territory: room for its neighbours. */
export const FOCUS_FRAME_PADDING = 0.12

/**
 * Frame on the territories a player controls: phones open the map on them
 * (charte-graphique.md § Téléphone). `key` names the game and the viewer, so
 * a new game or another hotseat player reframes the map.
 */
export function ownTerritoriesFrame(
  state: StateData,
  player: PlayerId,
  key: string,
): MapFrame {
  return {
    territories: state.territories
      .filter((territory) => territory.owner === player)
      .map((territory) => territory.id),
    key,
  }
}
