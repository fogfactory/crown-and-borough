import {
  InfrastructureMarker,
  NobleMarker,
  OwnershipBadge,
} from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import { centroid, pointsToPath } from '@/lib/map-svg-geometry'
import { isOccupiedAgainstController } from '@/lib/occupation'
import type { RegionStyle } from '@/lib/region-color'
import type { Fief, Region, StateData, Territory } from '@/types'

/**
 * Owner shields under each controlled territory's center. A territory that
 * belongs to a fief prints its capital's trigram on the shield instead of a
 * plain color, so several fiefs held by the same player stay tellable apart
 * (titres.md, issue #194). Since #196, control is transitive within a fief:
 * a non-capital member stays under its fief's control even while an enemy
 * (or NEUTRAL revolt) army sits on it, so the shield keeps the controller's
 * color and the tooltip names the occupant instead; `OccupiedHatchLayer`
 * paints the hatch overlay for that same case.
 */
export function OwnershipLayer({
  territories,
  state,
  playerColors,
  annotationScale,
  fiefs = [],
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
  annotationScale: number
  fiefs?: Fief[]
}) {
  const { t } = useLanguage()
  const fiefCapitalByTerritory = new Map<string, string>()
  for (const fief of fiefs) {
    for (const territoryId of fief.territories) {
      fiefCapitalByTerritory.set(territoryId, fief.capital)
    }
  }

  return (
    <g aria-label={t('map.control')} pointerEvents="none">
      {territories.map((territory) => {
        const territoryState = state.territories.find(
          (candidate) => candidate.id === territory.id,
        )
        const owner = territoryState?.owner
        if (!owner) {
          return null
        }

        const [centerX, centerY] = centroid(territory.points)
        const ownerName =
          state.players.find((player) => player.id === owner)?.name ?? owner
        const fiefCapital = fiefCapitalByTerritory.get(territory.id)
        const armyOwnerID = territoryState.army?.owner
        const occupantName = isOccupiedAgainstController(territoryState)
          ? (state.players.find((player) => player.id === armyOwnerID)?.name ??
            armyOwnerID)
          : null
        const label = occupantName
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
        return (
          <OwnershipBadge
            key={territory.id}
            ownerId={owner}
            x={centerX + (territoryState.army ? -32 : -9) * annotationScale}
            y={centerY + 26 * annotationScale}
            color={playerColors.get(owner) ?? '#475569'}
            scale={annotationScale}
            code={fiefCapital}
            label={label}
          />
        )
      })}
    </g>
  )
}

/**
 * Diagonal hatch over a territory that is controlled but currently occupied
 * against its controller (titres.md, issue #196): a non-capital fief member
 * under enemy or revolt garrison, but also a plain (non-fief) territory held
 * against a NEUTRAL revolt, since positional control elsewhere changes owner
 * immediately for any other army. Rendered under `OwnershipLayer`'s badges,
 * whose tooltip already names the occupant.
 */
export function OccupiedHatchLayer({
  territories,
  state,
}: {
  territories: Territory[]
  state: StateData
}) {
  const { t } = useLanguage()

  return (
    <g aria-label={t('map.occupiedZone')} pointerEvents="none">
      {territories.map((territory) => {
        const territoryState = state.territories.find(
          (candidate) => candidate.id === territory.id,
        )
        if (!isOccupiedAgainstController(territoryState)) {
          return null
        }

        return (
          <path
            key={territory.id}
            data-occupied-territory-id={territory.id}
            d={pointsToPath(territory.points)}
            fill="url(#occupied-hatch)"
          />
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
  annotationScale,
}: {
  territories: Territory[]
  state: StateData
  playerColors: Map<string, string>
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
                fill="#59401f"
                fontSize={11 * annotationScale}
                fontWeight="700"
                textAnchor="middle"
                stroke="#fff8e7"
                strokeWidth={3 * annotationScale}
                paintOrder="stroke"
              >
                {`×${territoryState.resources}`}
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
                  {t('map.armyMarker', {
                    owner: territoryState.army.owner,
                    size: territoryState.army.size,
                  })}
                </title>
                <circle
                  cx={centerX - 9 * annotationScale}
                  cy={centerY + 26 * annotationScale}
                  r={9 * annotationScale}
                  fill={playerColors.get(territoryState.army.owner) ?? '#475569'}
                  stroke="#fff8e7"
                  strokeWidth={2 * annotationScale}
                />
                <text
                  x={centerX - 9 * annotationScale}
                  y={centerY + 29 * annotationScale}
                  fill="#fff8e7"
                  fontSize={9 * annotationScale}
                  fontWeight="800"
                  textAnchor="middle"
                >
                  {territoryState.army.size}
                </text>
              </g>
            )}
            {territoryNobles.map((noble, index) => (
              <NobleMarker
                key={noble.id}
                noble={noble}
                x={centerX + (28 + index * 16) * annotationScale}
                y={centerY + 20 * annotationScale}
                color={playerColors.get(noble.owner) ?? '#475569'}
                scale={annotationScale}
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
          ? (regionStyleByID.get(chefRegion.id)?.fill ?? '#30291f')
          : '#30291f'
        const labelProps = {
          fontSize: 13 * annotationScale,
          fontWeight: '800' as const,
          letterSpacing: 0.5 * annotationScale,
          textAnchor: 'middle' as const,
          stroke: '#f5ecd9',
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
              fill="#30291f"
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
            fill="#30291f"
          >
            {territory.id}
          </text>
        )
      })}
    </g>
  )
}
