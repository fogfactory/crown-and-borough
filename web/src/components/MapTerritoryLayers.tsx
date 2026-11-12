import type { ReactNode } from 'react'

import { ArmyGlyph, CONTROL_REACH, ControlShield } from '@/components/MapHeraldry'
import { FamineMarker, InfrastructureMarker, NobleMarker } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { armyDrawingOwner, type ResolvedArms } from '@/lib/heraldry'
import type { NobleAttributes } from '@/lib/noble-attributes'
import { centroid, pointsToPath, placeMarker } from '@/lib/map-svg-geometry'
import { isOccupiedAgainstController } from '@/lib/occupation'
import type { RegionStyle } from '@/lib/region-color'
import {
  CODE_FONT_SIZE,
  CODE_LETTER_SPACING,
  NAME_FONT_SIZE,
  labelExtent,
  type TerritoryLabelLayout,
} from '@/lib/territory-labels'
import {
  NEUTRAL_PLAYER_ID,
  type Fief,
  type FiefTitle,
  type Region,
  type StateData,
  type Territory,
  type TerritoryState,
} from '@/types'
import {
  MAP_BROWN_FILL,
  MAP_HALO,
  MAP_INK,
  MAP_PARCHMENT,
  UNKNOWN_OWNER_COLOR,
} from '@/lib/map-palette'

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

/**
 * An army stands right of the territory label (the control shield is on its
 * left): this far from the label's right edge to the pack's centre, and with
 * its feet this far under the name's baseline, in units.
 */
const ARMY_REACH = 15
const ARMY_BELOW_BASELINE = 4

/**
 * Corners of a marker's footprint around its anchor, in units, with a margin
 * for the border band drawn along each edge: an army pack stands with its feet
 * on the anchor, a noble's pawn is centred on it.
 */
/** Height of an army pack and the gap kept between it and the label, in units. */
const ARMY_HEIGHT = 24
const ARMY_LABEL_GAP = 2
/** Half the height of a noble's pawn plus a gap, in units. */
const NOBLE_LABEL_GAP = 10
/** A noble's pawn is centred on its anchor: its feet are this far below it, in units. */
const NOBLE_HALF_HEIGHT = 8

const ARMY_FOOTPRINT: Array<[number, number]> = [
  [-16, 4],
  [16, 4],
  [-16, -26],
  [16, -26],
]
const NOBLE_FOOTPRINT: Array<[number, number]> = [
  [-14, -14],
  [14, -14],
  [-14, 14],
  [14, 14],
]

function scaled(
  footprint: Array<[number, number]>,
  scale: number,
): Array<[number, number]> {
  return footprint.map(([dx, dy]) => [dx * scale, dy * scale])
}

/** Nobles sharing a territory stand side by side, this far apart. */
const NOBLE_SPACING = 12

const COLOR_WASH_OPACITY = 0.35
const HATCH_OPACITY = 0.6
/** Wide diagonal bands for occupied territories: stripe and period, in map units. */
const HATCH_STRIPE = 8
const HATCH_PERIOD = 18

/**
 * Each controller's colour as a solid wash over the territories they
 * control (neutral excluded). A territory occupied by an army carries wide
 * diagonal bands in the occupier's colour; between the bands the controller's
 * wash shows through, or the bare map when nobody controls the territory.
 */
export function TerritoryColorLayer({
  territories,
  state,
  playerColors,
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
}) {
  const { t } = useLanguage()
  const stateById = new Map(state.territories.map((entry) => [entry.id, entry]))
  // Controlled in its own right: a fief member or capital, or a player's
  // capital. Any other territory is held only by the army standing on it.
  const anchored = new Set<string>([
    ...(state.fiefs ?? []).flatMap((fief) => [fief.capital, ...fief.territories]),
    ...state.players.flatMap((player) =>
      player.capitalTerritory ? [player.capitalTerritory] : [],
    ),
  ])
  const occupiers = new Set<string>()
  const cells = territories.flatMap((territory) => {
    const territoryState = stateById.get(territory.id)
    const owner = anchored.has(territory.id) ? territoryState?.owner : undefined
    const wash =
      owner && owner !== NEUTRAL_PLAYER_ID ? playerColors.get(owner) : undefined
    const armyOwner = territoryState?.army?.owner
    const occupier = armyOwner && armyOwner !== owner ? armyOwner : undefined
    if (occupier) occupiers.add(occupier)
    return wash || occupier ? [{ territory, wash, occupier }] : []
  })
  const hatchId = (owner: string) => `territory-hatch-${owner.replace(/\W/g, '_')}`

  return (
    <g aria-label={t('map.territoryColors')} data-territory-colors pointerEvents="none">
      <defs>
        {Array.from(occupiers).map((owner) => (
          <pattern
            key={owner}
            id={hatchId(owner)}
            width={HATCH_PERIOD}
            height={HATCH_PERIOD}
            patternUnits="userSpaceOnUse"
            patternTransform="rotate(45)"
          >
            <rect
              width={HATCH_PERIOD}
              height={HATCH_STRIPE}
              fill={playerColors.get(owner) ?? UNKNOWN_OWNER_COLOR}
              fillOpacity={HATCH_OPACITY}
            />
          </pattern>
        ))}
      </defs>
      {cells.map(({ territory, wash, occupier }) => {
        const d = pointsToPath(territory.points)
        return (
          <g key={territory.id}>
            {wash && (
              <path
                data-territory-color={territory.id}
                d={d}
                fill={wash}
                fillOpacity={COLOR_WASH_OPACITY}
                stroke="none"
              />
            )}
            {occupier && (
              <path
                data-territory-occupied={territory.id}
                d={d}
                fill={`url(#${hatchId(occupier)})`}
                stroke="none"
              />
            )}
          </g>
        )
      })}
    </g>
  )
}

/**
 * A shield on every controlled territory, in its controller's arms; greyed
 * while an army of another player (or rebels) occupies it. A fief capital
 * crowns its shield with the fief's rank (the king's closed crown for a crown
 * fief, over a neutral shield while the fief is vacant); every member of a
 * fief, capital included, writes the capital's trigram under its shield, so a
 * glance tells which fief a territory belongs to. A territory nobody
 * controls, or held only by the army standing on it (no fief, no player
 * capital), needs no shield: the army shows the occupation.
 */
export function OwnershipLayer({
  territories,
  state,
  annotationScale,
  fiefs = [],
  labelLayouts,
}: {
  territories: Territory[]
  state: StateData
  annotationScale: number
  fiefs?: Fief[]
  /** Each territory's label, which its shield stands beside. */
  labelLayouts: Map<string, TerritoryLabelLayout>
}) {
  const { t } = useLanguage()
  const fiefByTerritory = new Map<string, Fief>()
  for (const fief of fiefs) {
    for (const territoryId of fief.territories) {
      fiefByTerritory.set(territoryId, fief)
    }
    fiefByTerritory.set(fief.capital, fief)
  }
  const playerName = (id: string | undefined) =>
    state.players.find((player) => player.id === id)?.name ?? id ?? ''

  return (
    <g aria-label={t('map.control')} pointerEvents="none">
      {territories.map((territory) => {
        const territoryState = state.territories.find(
          (candidate) => candidate.id === territory.id,
        )
        const fief = fiefByTerritory.get(territory.id)
        const isCapital = fief?.capital === territory.id
        const owner = territoryState?.owner ?? (isCapital ? fief?.owner : undefined)
        if (!owner) {
          return null
        }
        // Control that only holds while the owner's army stands there (no
        // fief, no capital) is marked by that army, not by a shield.
        const anchored =
          Boolean(fief) ||
          state.players.some((player) => player.capitalTerritory === territory.id)
        if (!anchored && territoryState?.army?.owner === owner) {
          return null
        }

        const vacant = isCapital && fief && !fief.holder && !fief.crown
        const shieldOwner = vacant ? NEUTRAL_PLAYER_ID : owner
        const [centerX, centerY] = centroid(territory.points)
        const anchor = labelLayouts.get(territory.id)?.shieldAnchor ?? { x: 0, y: 0 }
        const footX = anchor.x - CONTROL_REACH
        const footY = anchor.y
        const ownerName = playerName(owner)
        const fiefCapital = fief && !isCapital ? fief.capital : undefined
        const occupantName = isOccupiedAgainstController(territoryState)
          ? playerName(territoryState?.army?.owner)
          : null
        const control = vacant
          ? t('map.fiefVacant')
          : occupantName
            ? fiefCapital
              ? t('map.ownershipBadgeFiefOccupied', {
                  owner: ownerName,
                  capital: fiefCapital,
                  occupant: occupantName,
                })
              : t('map.ownershipBadgeOccupied', {
                  owner: ownerName,
                  occupant: occupantName,
                })
            : fiefCapital
              ? t('map.ownershipBadgeFief', { owner: ownerName, capital: fiefCapital })
              : t('map.ownershipBadge', { owner: ownerName })
        const label =
          isCapital && fief
            ? `${t('map.fiefLabel', {
                title: t(FIEF_TITLE_KEYS[fief.title]),
                capital: territory.name,
              })} (${fief.capital}) · ${control}`
            : control
        return (
          <g
            key={territory.id}
            data-ownership-badge={owner}
            data-fief-capital={isCapital ? territory.id : undefined}
            transform={`translate(${centerX + footX * annotationScale} ${
              centerY + footY * annotationScale
            }) scale(${annotationScale})`}
          >
            <title>{label}</title>
            <ControlShield
              owner={shieldOwner}
              grey={Boolean(occupantName)}
              crown={isCapital && fief ? (fief.crown ? 'king' : fief.title) : undefined}
              attachedTo={fief?.capital}
            />
          </g>
        )
      })}
    </g>
  )
}

/** Per-territory live state: resources, infrastructures, army and nobles. */
export function LiveLayer({
  territories,
  state,
  playerColors,
  armsByOwner,
  nobleAttributes,
  annotationScale,
  selectedNoble = null,
  compactArmies = false,
  labelLayouts,
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
  armsByOwner: Map<string, ResolvedArms>
  nobleAttributes: Map<string, NobleAttributes>
  annotationScale: number
  /** Noble whose pawn wears the selection halo. */
  selectedNoble?: string | null
  /** Overview zoom: an army shrinks to its standard-bearer. */
  compactArmies?: boolean
  /** Each territory's label, which its army stands beside. */
  labelLayouts: Map<string, TerritoryLabelLayout>
}) {
  const { t } = useLanguage()
  // Full label: the owner's name and code, or the rebels.
  const armyLabel = (army: NonNullable<TerritoryState['army']>) => {
    if (army.owner === NEUTRAL_PLAYER_ID) {
      return t('map.rebelArmyMarker', { size: army.size })
    }
    const name = state.players.find((player) => player.id === army.owner)?.name
    const owner = name && name !== army.owner ? `${name} (${army.owner})` : army.owner
    return t(army.starving ? 'map.armyMarkerStarving' : 'map.armyMarker', {
      owner,
      size: army.size,
    })
  }

  // Each territory's stock and infrastructure, and its pawns (army, nobles),
  // each with the baseline it stands on.
  const placed = territories.flatMap((territory) => {
    const territoryState = state.territories.find(
      (candidate) => candidate.id === territory.id,
    )
    if (!territoryState) {
      return []
    }

    const [centerX, centerY] = centroid(territory.points)
    // The label's left edge is where the shield stands: its right edge
    // mirrors it, so the pack stands beyond it, feet on the baseline.
    const labelLayout = labelLayouts.get(territory.id)
    const labelLeft = labelLayout?.shieldAnchor ?? { x: 0, y: 0 }
    const label = labelLayout ? labelExtent(labelLayout) : { top: 0, bottom: 0 }
    const center: [number, number] = [centerX, centerY]
    const territoryPawns: Array<{ key: string; feet: number; node: ReactNode }> = []

    if (territoryState.army) {
      const army = territoryState.army
      const [armyX, armyFeet] = placeMarker(
        [
          centerX + (-labelLeft.x + ARMY_REACH) * annotationScale,
          centerY + (labelLeft.y + ARMY_BELOW_BASELINE) * annotationScale,
        ],
        // Too small a cell: the pack stands above the label, else below it.
        [
          [centerX, centerY + (label.top - ARMY_LABEL_GAP) * annotationScale],
          [
            centerX,
            centerY + (label.bottom + ARMY_HEIGHT + ARMY_LABEL_GAP) * annotationScale,
          ],
        ],
        territory.points,
        center,
        scaled(ARMY_FOOTPRINT, annotationScale),
      )
      territoryPawns.push({
        key: `${territory.id}-army`,
        feet: armyFeet,
        node: (
          <>
            <title>{armyLabel(army)}</title>
            <g transform={`translate(${armyX} ${armyFeet}) scale(${annotationScale})`}>
              <ArmyGlyph
                owner={armyDrawingOwner(armsByOwner, army.owner)}
                size={army.size}
                seed={territory.id}
                compact={compactArmies}
              />
            </g>
            {army.starving && (
              <FamineMarker
                x={armyX + 14 * annotationScale}
                y={armyFeet - 8 * annotationScale}
                scale={annotationScale}
              />
            )}
          </>
        ),
      })
    }

    state.nobles
      .filter((noble) => noble.location === territory.id)
      .forEach((noble, index) => {
        const nobleOffset = index * NOBLE_SPACING * annotationScale
        const [nobleX, nobleY] = placeMarker(
          [
            centerX + (28 + index * NOBLE_SPACING) * annotationScale,
            centerY + 20 * annotationScale,
          ],
          [
            [
              centerX + nobleOffset,
              centerY + (label.top - NOBLE_LABEL_GAP) * annotationScale,
            ],
            [
              centerX + nobleOffset,
              centerY + (label.bottom + NOBLE_LABEL_GAP) * annotationScale,
            ],
          ],
          territory.points,
          center,
          scaled(NOBLE_FOOTPRINT, annotationScale),
        )
        territoryPawns.push({
          key: noble.id,
          feet: nobleY + NOBLE_HALF_HEIGHT * annotationScale,
          node: (
            <NobleMarker
              noble={noble}
              x={nobleX}
              y={nobleY}
              color={playerColors.get(noble.owner) ?? UNKNOWN_OWNER_COLOR}
              scale={annotationScale}
              attributes={nobleAttributes.get(noble.code)}
              selected={noble.code === selectedNoble}
            />
          ),
        })
      })

    const ground = (
      <>
        {territoryState.resources > 0 && (
          <text
            x={centerX + 30 * annotationScale}
            y={centerY - 20 * annotationScale}
            fill={MAP_BROWN_FILL}
            fontSize={11 * annotationScale}
            fontWeight="700"
            textAnchor="middle"
            stroke={MAP_PARCHMENT}
            strokeWidth={3 * annotationScale}
            paintOrder="stroke"
          >
            {t('map.stock', { amount: territoryState.resources })}
          </text>
        )}
        {territoryState.infrastructures.map((infrastructure, index) => (
          <InfrastructureMarker
            key={`${territory.id}-${infrastructure.type}-${index}`}
            infrastructure={infrastructure}
            x={centerX + (index * 24 - 9) * annotationScale}
            y={centerY - 30 * annotationScale}
            isCapital={
              infrastructure.type === 'castle' &&
              state.players.some((player) => player.capitalTerritory === territory.id)
            }
            scale={annotationScale}
            ownerColor={
              territoryState.owner
                ? (playerColors.get(territoryState.owner) ?? null)
                : null
            }
          />
        ))}
      </>
    )
    return [{ key: territory.id, ground, pawns: territoryPawns }]
  })

  // Pawns are drawn from the back to the front: the lower one stands in front
  // of the one it overlaps, whichever territory it belongs to.
  const pawns = placed.flatMap((entry) => entry.pawns).sort((a, b) => a.feet - b.feet)

  return (
    <g aria-label={t('map.liveLayer')} pointerEvents="none">
      {placed.map(({ key, ground }) => (
        <g key={key}>{ground}</g>
      ))}
      {pawns.map(({ key, node }) => (
        <g key={key}>{node}</g>
      ))}
    </g>
  )
}

/**
 * Territory labels at each centroid: the name in Alegreya italic above the
 * code in Cinzel capitals, or the code alone for an ordinary territory in the
 * synthetic view. While regions are shown, a chef-lieu writes its name in its
 * bishopric's tint.
 */
export function TerritoryLabels({
  territories,
  labelLayouts,
  regionsActive,
  regionBySeed,
  regionStyleByID,
  annotationScale,
}: {
  territories: Territory[]
  labelLayouts: Map<string, TerritoryLabelLayout>
  regionsActive: boolean
  regionBySeed: Map<string, Region>
  regionStyleByID: Map<string, RegionStyle>
  annotationScale: number
}) {
  const { t } = useLanguage()
  const halo = {
    stroke: MAP_HALO,
    strokeLinejoin: 'round' as const,
    paintOrder: 'stroke' as const,
    textAnchor: 'middle' as const,
  }

  return (
    <g aria-label={t('map.territoryLabels')} pointerEvents="none">
      {territories.map((territory) => {
        const layout = labelLayouts.get(territory.id)
        if (!layout) return null
        const [centerX, centerY] = centroid(territory.points)
        const chefRegion = regionsActive ? regionBySeed.get(territory.id) : undefined
        const nameFill = chefRegion
          ? (regionStyleByID.get(chefRegion.id)?.text ?? MAP_INK)
          : MAP_INK
        return (
          <g key={territory.id} data-territory-label={territory.id}>
            {layout.nameLines.length > 0 && (
              <text
                {...halo}
                x={centerX}
                fill={nameFill}
                fontFamily="var(--font-sans)"
                fontStyle="italic"
                fontWeight="500"
                fontSize={NAME_FONT_SIZE * annotationScale}
                strokeWidth={3 * annotationScale}
                data-territory-name={territory.id}
                data-chef-lieu={chefRegion ? territory.id : undefined}
              >
                {layout.nameLines.map((line, index) => (
                  <tspan
                    key={line}
                    x={centerX}
                    y={centerY + layout.nameBaselines[index] * annotationScale}
                  >
                    {line}
                  </tspan>
                ))}
              </text>
            )}
            <text
              {...halo}
              x={centerX}
              y={centerY + layout.codeBaseline * annotationScale}
              fill={MAP_INK}
              fontFamily="var(--font-display)"
              fontWeight="700"
              fontSize={CODE_FONT_SIZE * annotationScale}
              letterSpacing={CODE_FONT_SIZE * CODE_LETTER_SPACING * annotationScale}
              strokeWidth={2.5 * annotationScale}
              data-territory-code={territory.id}
            >
              {territory.id}
            </text>
          </g>
        )
      })}
    </g>
  )
}
