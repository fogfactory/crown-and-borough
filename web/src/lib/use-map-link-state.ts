import { useMemo, useRef, useState } from 'react'

import type { MapLinks } from '@/components/MapLinks'
import { FOCUS_FRAME_PADDING, type MapFrame } from '@/lib/map-frame'
import type { MapData, StateData } from '@/types'

interface MapLinkStateOptions {
  map: MapData | null
  state: StateData | null
  /** Selected territory of the page. */
  selectedId: string | null
  /** Frame the page shows by default (phones: the viewer's territories). */
  baseFrame?: MapFrame
  /** Selects a territory, as a click on the map would. */
  onSelect: (territory: string) => void
}

interface MapLinkState {
  links: MapLinks
  /** Frame to hand to the map: the last followed link, else the base frame. */
  frame?: MapFrame
  highlightedId: string | null
  selectedNoble: string | null
}

interface Focus {
  frame: MapFrame
  noble?: string
  /** Base frame current when the link was followed. */
  baseKey?: string | number
}

/**
 * Map side of the links written in the report and the command post
 * (charte-graphique.md § Identifiants et liens vers la carte): following a
 * link frames the map on the territory and selects it, a noble's pawn gets
 * the selection halo, and hovering a link lights its territory.
 */
export function useMapLinkState({
  map,
  state,
  selectedId,
  baseFrame,
  onSelect,
}: MapLinkStateOptions): MapLinkState {
  const [focus, setFocus] = useState<Focus | null>(null)
  const [highlightedId, setHighlightedId] = useState<string | null>(null)
  const followed = useRef(0)
  // Latest page values, read when a link is followed, so the context value
  // stays the same object between renders.
  const latest = useRef({ baseKey: baseFrame?.key, onSelect })
  latest.current = { baseKey: baseFrame?.key, onSelect }

  const territoryNames = useMemo(
    () => new Map((map?.territories ?? []).map((territory) => [territory.id, territory.name])),
    [map],
  )
  const nobles = useMemo(
    () => new Map((state?.nobles ?? []).map((noble) => [noble.code, noble])),
    [state],
  )
  const deceased = useMemo(
    () => new Map((state?.deceased ?? []).map((noble) => [noble.code, noble.name])),
    [state],
  )

  const links = useMemo<MapLinks>(
    () => ({
      focus: (territory, noble) => {
        followed.current += 1
        setFocus({
          frame: {
            territories: [territory],
            key: `link:${followed.current}`,
            padding: FOCUS_FRAME_PADDING,
          },
          noble,
          baseKey: latest.current.baseKey,
        })
        setHighlightedId(null)
        latest.current.onSelect(territory)
      },
      highlight: setHighlightedId,
      hasTerritory: (id) => territoryNames.has(id),
      territoryName: (id) => territoryNames.get(id),
      nobleName: (code) => nobles.get(code)?.name ?? deceased.get(code),
      nobleTerritory: (code) => {
        const location = nobles.get(code)?.location
        return location && territoryNames.has(location) ? location : undefined
      },
      isDeceased: (code) => deceased.has(code),
    }),
    [territoryNames, nobles, deceased],
  )

  // A new base frame (another game or hotseat player) supersedes the link.
  const current = focus && focus.baseKey === baseFrame?.key ? focus : null
  return {
    links,
    frame: current?.frame ?? baseFrame,
    highlightedId,
    selectedNoble:
      current?.noble && selectedId === current.frame.territories[0]
        ? current.noble
        : null,
  }
}
