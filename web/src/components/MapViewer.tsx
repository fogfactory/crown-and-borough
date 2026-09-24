import { useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'

import { MapLegend, TERRAIN_COLORS, TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { MapControls } from '@/components/MapControls'
import { snowPatternVariant } from '@/components/MapDecorations'
import {
  CalamityIconLayer,
  CardIconLayer,
  ImpassableBorderChain,
} from '@/components/MapIconLayers'
import { IntentionsOverlay } from '@/components/MapIntentionsOverlay'
import {
  RegionBadges,
  RegionBands,
  RegionEffectMarkers,
  RegionRings,
} from '@/components/MapRegionLayers'
import { MapSvgDefs } from '@/components/MapSvgDefs'
import {
  LiveLayer,
  OwnershipLayer,
  TerritoryLabels,
} from '@/components/MapTerritoryLayers'
import { WinterOrdersOverlay } from '@/components/MapWinterOrdersOverlay'
import { useLanguage } from '@/i18n/LanguageContext'
import type { Intention } from '@/lib/intent-overlay'
import type { WinterIntention } from '@/lib/winter-overlay'
import { NEUTRAL_PLAYER_ID } from '@/types'
import { centroid, pointsToPath, splitBoundaryPaths } from '@/lib/map-svg-geometry'
import { hasSupplySource } from '@/lib/supply'
import { useMapBorders } from '@/lib/use-map-borders'
import { useMapGestures } from '@/lib/use-map-gestures'
import { useRegionEffectIcons } from '@/lib/use-region-effect-icons'
import { useRegionLayout } from '@/lib/use-region-layout'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { MapData, PlayerId, StateData, SupplyLine } from '@/types'

const OUTER_BORDER_WIDTH = 2
const PASSABLE_BORDER_WIDTH = 2

const PLAYER_PALETTE = ['#a84632', '#2d5f9e', '#7052a1', '#0e7490', '#ad7a25']

/** Rebel armies answer to no crown: they render in a neutral gray. */
const NEUTRAL_ARMY_COLOR = '#6b7280'

interface MapViewerProps {
  map: MapData
  state: StateData
  onSelect?: (id: string | null) => void
  supply?: SupplyLine | null
  intentions?: Intention[]
  winterIntentions?: WinterIntention[]
  showIntentions?: boolean
  intentionsColor?: string
  onToggleIntentions?: (show: boolean) => void
  showOwnership?: boolean
  onToggleOwnership?: (show: boolean) => void
  showRegions?: boolean
  onToggleRegions?: (show: boolean) => void
  showCalamities?: boolean
  onToggleCalamities?: (show: boolean) => void
  showCards?: boolean
  onToggleCards?: (show: boolean) => void
  /** Deck-order drafts per player, parsed into the card overlay. */
  specialOrders?: Array<{ player: PlayerId; text: string }>
}

export function MapViewer({
  map,
  state,
  onSelect,
  supply,
  intentions = [],
  winterIntentions = [],
  showIntentions = false,
  intentionsColor = '#a84632',
  onToggleIntentions,
  showOwnership = true,
  onToggleOwnership,
  showRegions = false,
  onToggleRegions,
  showCalamities = true,
  onToggleCalamities,
  showCards = true,
  onToggleCards,
  specialOrders = [],
}: MapViewerProps) {
  const { t } = useLanguage()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [legendOpen, setLegendOpen] = useState(false)

  const {
    mapWidth,
    mapHeight,
    outerBorders,
    sharedBorders,
    passableBoundaryKeys,
    annotationScale,
    impassableBorderIcons,
  } = useMapBorders(map)
  const {
    regionsActive,
    regionStyleByID,
    regionsById,
    regionBySeed,
    regionOutlines,
    interiorRegionIDs,
    regionBands,
    regionHoleTesters,
    bandMargin,
  } = useRegionLayout(map, showRegions, mapWidth, mapHeight)
  const { calamityIcons, cardIcons } = useRegionEffectIcons({
    map,
    state,
    specialOrders,
    showCalamities,
    showCards,
  })

  const bandMarginX = bandMargin
  const bandMarginY = bandMarginX
  const viewWidth = mapWidth + bandMarginX * 2
  const viewHeight = mapHeight + bandMarginY * 2

  const selectTerritory = (id: string | null) => {
    const nextId = id !== null && id !== selectedId ? id : null
    setSelectedId(nextId)
    onSelect?.(nextId)
  }

  const handleTerritoryKeyDown = (
    event: ReactKeyboardEvent<SVGPathElement>,
    id: string,
  ) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return
    }

    event.preventDefault()
    selectTerritory(id)
  }

  const { svgRef, view, isDragging, pointerHandlers, zoomBy } = useMapGestures({
    viewWidth,
    viewHeight,
    bandMarginX,
    bandMarginY,
    mapWidth,
    mapHeight,
    onTap: selectTerritory,
  })

  const colorsByPlayer = new Map(state.players.map((player) => [player.id, player.color]))
  const owners = Array.from(
    new Set<string>([
      ...state.players.map((player) => player.id),
      ...state.territories.flatMap((territoryState) =>
        territoryState.owner ? [territoryState.owner] : [],
      ),
      ...state.territories.flatMap((territoryState) =>
        territoryState.army ? [territoryState.army.owner] : [],
      ),
      ...state.nobles.map((noble) => noble.owner),
    ]),
  ).sort((first, second) => first.localeCompare(second))
  const playerColors = new Map(
    owners.map((owner, index) => [
      owner,
      owner === NEUTRAL_PLAYER_ID
        ? NEUTRAL_ARMY_COLOR
        : (colorsByPlayer.get(owner) ?? PLAYER_PALETTE[index % PLAYER_PALETTE.length]),
    ]),
  )
  const selectedTerritoryState = state.territories.find(
    (territoryState) => territoryState.id === selectedId,
  )
  const selectedSupply =
    supply?.territory === selectedId &&
    ((supply.kind === 'army' && Boolean(selectedTerritoryState?.army)) ||
      (supply.kind === 'source' &&
        !selectedTerritoryState?.army &&
        hasSupplySource(selectedTerritoryState)))
      ? supply
      : null
  const supplyReachable = new Set(selectedSupply?.reachable ?? [])
  const supplyColor = playerColors.get(selectedSupply?.armyOwner ?? '') ?? '#a84632'
  const supplyPathPoints = (selectedSupply?.path ?? []).flatMap((territoryID) => {
    const territory = map.territories.find((candidate) => candidate.id === territoryID)
    return territory ? [centroid(territory.points)] : []
  })
  const passableBorderDash = `${4 * annotationScale} ${3 * annotationScale}`
  const passableBorderDots = `0.1 ${4.5 * annotationScale}`
  const supplyPathDash = `${8 * annotationScale} ${5 * annotationScale}`
  const supplyEndpointDash = `${3 * annotationScale} ${3 * annotationScale}`

  return (
    <TooltipProvider delayDuration={100}>
      <div
        className={`relative h-full w-full overflow-hidden ${
          isDragging ? 'cursor-grabbing' : 'cursor-grab'
        }`}
      >
        <svg
          ref={svgRef}
          className="h-full w-full select-none"
          viewBox={`${-bandMarginX} ${-bandMarginY} ${viewWidth} ${viewHeight}`}
          preserveAspectRatio="xMidYMid meet"
          role="group"
          aria-label={t('map.territories')}
          {...pointerHandlers}
          style={{ touchAction: 'none' }}
        >
          <rect width={mapWidth} height={mapHeight} fill="#e6d8bb" />
          <g transform={`translate(${view.x} ${view.y}) scale(${view.k})`}>
            <MapSvgDefs
              territories={map.territories}
              annotationScale={annotationScale}
              bandMarginX={bandMarginX}
              bandMarginY={bandMarginY}
              viewWidth={viewWidth}
              viewHeight={viewHeight}
              intentionsColor={intentionsColor}
            />

            <RegionBands
              bands={regionBands}
              regionStyleByID={regionStyleByID}
              regionsById={regionsById}
              territories={map.territories}
            />

            <g aria-label={t('map.terrains')}>
              {map.territories.map((territory) => (
                <Tooltip key={territory.id}>
                  <TooltipTrigger asChild>
                    <path
                      data-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill={TERRAIN_COLORS[territory.terrain]}
                      stroke="none"
                      tabIndex={0}
                      role="button"
                      aria-pressed={territory.id === selectedId}
                      aria-label={t('map.territoryLabel', {
                        name: territory.name,
                        terrain: t(TERRAIN_LABEL_KEYS[territory.terrain]),
                      })}
                      onKeyDown={(event) => handleTerritoryKeyDown(event, territory.id)}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="top" sideOffset={8}>
                    <div className="space-y-0.5">
                      <p className="font-semibold">{territory.name}</p>
                      <p>{t(TERRAIN_LABEL_KEYS[territory.terrain])}</p>
                    </div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </g>

            <g aria-label={t('map.terrainTextures')} pointerEvents="none">
              {map.territories.map((territory) => (
                <path
                  key={territory.id}
                  d={pointsToPath(territory.points)}
                  fill={`url(#terrain-${territory.terrain})`}
                />
              ))}
            </g>
            {regionsActive && (
              <RegionRings
                outlines={regionOutlines}
                regionStyleByID={regionStyleByID}
                regionHoleTesters={regionHoleTesters}
                annotationScale={annotationScale}
              />
            )}

            {state.season === 'winter' && showIntentions && (
              <IntentionsOverlay
                intentions={intentions}
                annotationScale={annotationScale}
                passableBorderDash={passableBorderDash}
                intentionsColor={intentionsColor}
              />
            )}

            {state.season === 'winter' && (
              <g
                aria-label={t('map.winterVeil')}
                data-winter-veil-overlay="true"
                pointerEvents="none"
              >
                <rect
                  x={-bandMarginX}
                  y={-bandMarginY}
                  width={viewWidth}
                  height={viewHeight}
                  fill="#eaf3ff"
                  opacity="0.2"
                />
              </g>
            )}
            {state.season === 'winter' && (
              <g aria-label={t('map.winterSnow')} pointerEvents="none">
                {map.territories.map((territory) => (
                  <path
                    key={territory.id}
                    d={pointsToPath(territory.points)}
                    fill={`url(#winter-snow-${snowPatternVariant(territory.id)})`}
                  />
                ))}
              </g>
            )}

            {showOwnership && (
              <OwnershipLayer
                territories={map.territories}
                state={state}
                playerColors={playerColors}
                annotationScale={annotationScale}
              />
            )}

            {supplyReachable.size > 0 && (
              <g aria-label={t('map.supplyZone')} pointerEvents="none">
                {map.territories.map((territory) => {
                  if (!supplyReachable.has(territory.id)) {
                    return null
                  }

                  return (
                    <path
                      key={territory.id}
                      data-supply-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill="url(#supply-zone-hatch)"
                    />
                  )
                })}
              </g>
            )}

            <g aria-label={t('map.selection')} pointerEvents="none">
              {map.territories.map((territory) => {
                if (territory.id !== selectedId) {
                  return null
                }

                const { solidPath, passablePath } = splitBoundaryPaths(
                  territory.points,
                  passableBoundaryKeys,
                )

                return (
                  <g key={territory.id}>
                    {solidPath && (
                      <path
                        d={solidPath}
                        fill="none"
                        stroke="#d28b22"
                        strokeWidth="5"
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                    {passablePath && (
                      <path
                        d={passablePath}
                        fill="none"
                        stroke="#d28b22"
                        strokeWidth="5"
                        strokeDasharray={passableBorderDots}
                        strokeLinecap="round"
                        clipPath={`url(#territory-clip-${territory.id})`}
                        vectorEffect="non-scaling-stroke"
                      />
                    )}
                  </g>
                )
              })}
            </g>

            <g aria-label={t('map.outerBorders')} pointerEvents="none">
              {outerBorders.map((border) => (
                <line
                  key={border.key}
                  x1={border.from[0]}
                  y1={border.from[1]}
                  x2={border.to[0]}
                  y2={border.to[1]}
                  stroke="#594b3c"
                  strokeOpacity="0.85"
                  strokeWidth={OUTER_BORDER_WIDTH}
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              ))}
            </g>

            <g aria-label={t('map.borders')} pointerEvents="none">
              {sharedBorders
                .filter((border) => border.passable)
                .map((border) => (
                  <line
                    key={border.key}
                    x1={border.from[0]}
                    y1={border.from[1]}
                    x2={border.to[0]}
                    y2={border.to[1]}
                    stroke="#39271b"
                    strokeOpacity="0.85"
                    strokeWidth={PASSABLE_BORDER_WIDTH}
                    strokeDasharray={passableBorderDots}
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
              <ImpassableBorderChain icons={impassableBorderIcons} />
            </g>

            {supplyPathPoints.length > 0 && (
              <g aria-label={t('map.supplyLine')} pointerEvents="none">
                <polyline
                  points={supplyPathPoints.map(([x, y]) => `${x},${y}`).join(' ')}
                  fill="none"
                  stroke={supplyColor}
                  strokeWidth="5"
                  strokeDasharray={supplyPathDash}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  vectorEffect="non-scaling-stroke"
                />
                <circle
                  cx={supplyPathPoints[0][0]}
                  cy={supplyPathPoints[0][1]}
                  r={7 * annotationScale}
                  fill="#fff8e7"
                  stroke={supplyColor}
                  strokeWidth="3"
                  vectorEffect="non-scaling-stroke"
                />
                {supplyPathPoints.length > 1 && (
                  <circle
                    cx={supplyPathPoints[supplyPathPoints.length - 1][0]}
                    cy={supplyPathPoints[supplyPathPoints.length - 1][1]}
                    r={13 * annotationScale}
                    fill="none"
                    stroke={supplyColor}
                    strokeWidth="2"
                    strokeDasharray={supplyEndpointDash}
                    vectorEffect="non-scaling-stroke"
                  />
                )}
              </g>
            )}

            <CalamityIconLayer icons={calamityIcons} />
            <CardIconLayer icons={cardIcons} />
            <LiveLayer
              territories={map.territories}
              state={state}
              playerColors={playerColors}
              annotationScale={annotationScale}
            />

            <TerritoryLabels
              territories={map.territories}
              regionsActive={regionsActive}
              regionBySeed={regionBySeed}
              regionStyleByID={regionStyleByID}
              annotationScale={annotationScale}
            />

            {regionsActive && (
              <RegionBadges
                regions={map.regions ?? []}
                interiorRegionIDs={interiorRegionIDs}
                territories={map.territories}
                regionStyleByID={regionStyleByID}
                annotationScale={annotationScale}
              />
            )}

            {showRegions && (state.activeRegionEffects?.length ?? 0) > 0 && (
              <RegionEffectMarkers
                effects={state.activeRegionEffects}
                regions={map.regions}
                territories={map.territories}
                annotationScale={annotationScale}
              />
            )}

            {state.season !== 'winter' && showIntentions && (
              <IntentionsOverlay
                intentions={intentions}
                annotationScale={annotationScale}
                passableBorderDash={passableBorderDash}
                intentionsColor={intentionsColor}
              />
            )}

            {showIntentions &&
              state.season === 'winter' &&
              winterIntentions.length > 0 && (
                <WinterOrdersOverlay
                  territories={map.territories}
                  winterIntentions={winterIntentions}
                  annotationScale={annotationScale}
                />
              )}
          </g>
        </svg>

        <MapControls onZoom={zoomBy} />

        {(onToggleIntentions ||
          onToggleOwnership ||
          onToggleRegions ||
          onToggleCalamities ||
          onToggleCards) && (
          <div className="absolute right-3 top-3 z-10">
            <button
              type="button"
              aria-expanded={legendOpen}
              aria-label={t(legendOpen ? 'legend.hide' : 'legend.show')}
              title={t(legendOpen ? 'legend.hide' : 'legend.show')}
              className="flex size-9 items-center justify-center rounded-lg border border-[#b7a786] bg-[#fffaf0] text-[#594b3c] shadow-md transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
              onClick={() => setLegendOpen((open) => !open)}
            >
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                className="size-4"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M4 4h6v6h-6z" />
                <path d="M14 6l6 0" />
                <path d="M14 10l6 0" />
                <path d="M4 14h16" />
                <path d="M4 18h16" />
              </svg>
            </button>
            {legendOpen && (
              <div className="absolute right-0 top-11 w-72 max-w-[calc(100vw-1.5rem)]">
                <MapLegend
                  showIntentions={showIntentions}
                  onToggleIntentions={onToggleIntentions}
                  showOwnership={showOwnership}
                  onToggleOwnership={onToggleOwnership}
                  showRegions={showRegions}
                  onToggleRegions={onToggleRegions}
                  showCalamities={showCalamities}
                  onToggleCalamities={onToggleCalamities}
                  showCards={showCards}
                  onToggleCards={onToggleCards}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </TooltipProvider>
  )
}
