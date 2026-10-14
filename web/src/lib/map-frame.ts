import type { PlayerId, StateData } from '@/types'

/** Territories to frame on the map; a new `key` frames them again. */
export interface MapFrame {
  territories: string[]
  key: string | number
}

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
