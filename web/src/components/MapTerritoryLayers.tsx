import { BannerGlyph, shieldSymbolId } from '@/components/MapHeraldry'
import { FamineMarker, InfrastructureMarker, NobleMarker } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { ResolvedArms } from '@/lib/heraldry'
import type { NobleAttributes } from '@/lib/noble-attributes'
import { centroid } from '@/lib/map-svg-geometry'
import { isOccupiedAgainstController } from '@/lib/occupation'
import type { RegionStyle } from '@/lib/region-color'
import {
  NEUTRAL_PLAYER_ID,
  type Fief,
  type FiefTitle,
  type Region,
  type StateData,
  type Territory,
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

/** The banner's foot sits left of the territory code, on its baseline. */
const BANNER_FOOT_X = -33
const BANNER_FOOT_Y = 4
/** Half advance of a chef-lieu name glyph (12 units, weight 800), in units. */
const CHEF_LIEU_HALF_ADVANCE = 3.9
/** Room between the banner's foot and the end of its flag, plus a gap. */
const BANNER_CLEARANCE = 15

/**
 * Horizontal offset of the banner's foot: left of the code, or left of the
 * chef-lieu name written above it when that name is wider.
 */
function bannerFootX(territory: Territory, chefLieu: boolean): number {
  if (!chefLieu) return BANNER_FOOT_X
  return Math.min(
    BANNER_FOOT_X,
    -(territory.name.length * CHEF_LIEU_HALF_ADVANCE + BANNER_CLEARANCE),
  )
}

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
  chefLieuIds = new Set(),
}: {
  territories: Territory[]
  state: StateData
  annotationScale: number
  fiefs?: Fief[]
  /** Territories whose label also writes the chef-lieu name. */
  chefLieuIds?: Set<string>
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
            transform={`translate(${
              centerX +
              bannerFootX(territory, chefLieuIds.has(territory.id)) * annotationScale
            } ${centerY + BANNER_FOOT_Y * annotationScale}) scale(${annotationScale})`}
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
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
  armsByOwner: Map<string, ResolvedArms>
  nobleAttributes: Map<string, NobleAttributes>
  annotationScale: number
}) {
  const { t } = useLanguage()

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
                y={centerY - 25 * annotationScale}
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
                <title>
                  {territoryState.army.starving
                    ? t('map.armyMarkerStarving', {
                        owner: territoryState.army.owner,
                        size: territoryState.army.size,
                      })
                    : t('map.armyMarker', {
                        owner: territoryState.army.owner,
                        size: territoryState.army.size,
                      })}
                </title>
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
              />
            ))}
          </g>
        )
      })}
    </g>
  )
}

/**
 * Territory codes at each centroid. While regions are shown, a region's
 * seed (chef-lieu) also carries its name in the region color.
 */
export function TerritoryLabels({
  territories,
  regionsActive,
  regionBySeed,
  regionStyleByID,
  annotationScale,
}: {
  territories: Territory[]
  regionsActive: boolean
  regionBySeed: Map<string, Region>
  regionStyleByID: Map<string, RegionStyle>
  annotationScale: number
}) {
  const { t } = useLanguage()

  return (
    <g aria-label={t('map.territoryLabels')} pointerEvents="none">
      {territories.map((territory) => {
        const [centerX, centerY] = centroid(territory.points)
        const chefRegion = regionsActive ? regionBySeed.get(territory.id) : undefined
        const chefFill = chefRegion
          ? (regionStyleByID.get(chefRegion.id)?.text ?? MAP_INK)
          : MAP_INK
        const labelProps = {
          fontSize: 13 * annotationScale,
          fontWeight: '800' as const,
          letterSpacing: 0.5 * annotationScale,
          textAnchor: 'middle' as const,
          stroke: MAP_HALO,
          strokeWidth: 3 * annotationScale,
          paintOrder: 'stroke' as const,
        }
        return chefRegion ? (
          <g key={territory.id}>
            <text
              {...labelProps}
              x={centerX}
              y={centerY - 5 * annotationScale}
              fill={chefFill}
              fontSize={12 * annotationScale}
              data-chef-lieu={territory.id}
            >
              {territory.name}
            </text>
            <text
              {...labelProps}
              x={centerX}
              y={centerY + 11 * annotationScale}
              fill={MAP_INK}
            >
              {territory.id}
            </text>
          </g>
        ) : (
          <text
            key={territory.id}
            x={centerX}
            y={centerY + 4 * annotationScale}
            {...labelProps}
            fill={MAP_INK}
          >
            {territory.id}
          </text>
        )
      })}
    </g>
  )
}
