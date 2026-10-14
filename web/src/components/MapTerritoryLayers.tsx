import { BannerGlyph, shieldSymbolId } from '@/components/MapHeraldry'
import { BANNER_WIDTH } from '@/lib/heraldry'
import { FamineMarker, InfrastructureMarker, NobleMarker } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { ResolvedArms } from '@/lib/heraldry'
import type { NobleAttributes } from '@/lib/noble-attributes'
import { centroid } from '@/lib/map-svg-geometry'
import { isOccupiedAgainstController } from '@/lib/occupation'
import type { RegionStyle } from '@/lib/region-color'
import {
  CODE_FONT_SIZE,
  CODE_LETTER_SPACING,
  NAME_FONT_SIZE,
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
  SHIELD_NUMBER,
  UNKNOWN_OWNER_COLOR,
} from '@/lib/map-palette'

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

/** Nobles sharing a territory stand side by side, this far apart. */
const NOBLE_SPACING = 12

/** From a label's left edge to the banner's foot: the flag and a gap. */
const BANNER_REACH = BANNER_WIDTH + 3

/**
 * A banner on every controlled territory, in its controller's arms. A fief
 * capital crowns its banner with the fief's rank (the king's closed crown for
 * a crown fief, over a neutral banner while the fief is vacant); the other
 * members write the capital's trigram under the flag, so a glance tells
 * which fief a territory belongs to. Occupation needs no mark: the army
 * present carries its own shield, and the tooltip names the occupant.
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
  /** Each territory's label, which its banner stands beside. */
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

        const vacant = isCapital && fief && !fief.holder && !fief.crown
        const bannerOwner = vacant ? NEUTRAL_PLAYER_ID : owner
        const [centerX, centerY] = centroid(territory.points)
        const anchor = labelLayouts.get(territory.id)?.bannerAnchor ?? { x: 0, y: 0 }
        const footX = anchor.x - BANNER_REACH
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
            <BannerGlyph
              owner={bannerOwner}
              crown={isCapital && fief ? (fief.crown ? 'king' : fief.title) : undefined}
              attachedTo={fiefCapital}
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
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
  armsByOwner: Map<string, ResolvedArms>
  nobleAttributes: Map<string, NobleAttributes>
  annotationScale: number
  /** Noble whose pawn wears the selection halo. */
  selectedNoble?: string | null
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

  return (
    <g aria-label={t('map.liveLayer')} pointerEvents="none">
      {territories.map((territory) => {
        const territoryState = state.territories.find(
          (candidate) => candidate.id === territory.id,
        )
        if (!territoryState) {
          return null
        }

        const [centerX, centerY] = centroid(territory.points)
        const territoryNobles = state.nobles.filter(
          (noble) => noble.location === territory.id,
        )

        return (
          <g key={territory.id}>
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
            {territoryState.army && (
              <g key={`${territory.id}-army`}>
                <title>{armyLabel(territoryState.army)}</title>
                <use
                  href={`#${shieldSymbolId(
                    armsByOwner.has(territoryState.army.owner)
                      ? territoryState.army.owner
                      : NEUTRAL_PLAYER_ID,
                  )}`}
                  data-army-owner={territoryState.army.owner}
                  transform={`translate(${centerX - 9 * annotationScale} ${
                    centerY + 26 * annotationScale
                  }) scale(${annotationScale})`}
                />
                <text
                  x={centerX - 9 * annotationScale}
                  y={centerY + 26 * annotationScale}
                  fill={SHIELD_NUMBER}
                  fontSize={10 * annotationScale}
                  fontWeight="700"
                  textAnchor="middle"
                  dominantBaseline="central"
                  stroke={MAP_INK}
                  strokeWidth={2.4 * annotationScale}
                  strokeLinejoin="round"
                  paintOrder="stroke"
                >
                  {territoryState.army.size}
                </text>
                {territoryState.army.starving && (
                  <FamineMarker
                    x={centerX - 9 * annotationScale + 8 * annotationScale}
                    y={centerY + 26 * annotationScale - 9 * annotationScale}
                    scale={annotationScale}
                  />
                )}
              </g>
            )}
            {territoryNobles.map((noble, index) => (
              <NobleMarker
                key={noble.id}
                noble={noble}
                x={centerX + (28 + index * NOBLE_SPACING) * annotationScale}
                y={centerY + 20 * annotationScale}
                color={playerColors.get(noble.owner) ?? UNKNOWN_OWNER_COLOR}
                scale={annotationScale}
                attributes={nobleAttributes.get(noble.code)}
                selected={noble.code === selectedNoble}
              />
            ))}
          </g>
        )
      })}
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
