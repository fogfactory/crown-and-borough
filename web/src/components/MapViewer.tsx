import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from 'react'

import { MapLegend, TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { MapControls } from '@/components/MapControls'
import { HeraldryDefs } from '@/components/MapHeraldry'
import { EngravedTerrainLayer, snowPatternVariant } from '@/components/MapDecorations'
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
import type { MapFrame } from '@/lib/map-frame'
import { boundsOf, centroid, pointsToPath } from '@/lib/map-svg-geometry'
import { resolveArmsByOwner } from '@/lib/heraldry'
import { nobleAttributesByCode } from '@/lib/noble-attributes'
import { hasSupplySource } from '@/lib/supply'
import {
  NAME_MIN_SCREEN_PX,
  nameScreenSize,
  particularTerritoryIds,
  territoryLabelLayout,
} from '@/lib/territory-labels'
import { useMapBorders } from '@/lib/use-map-borders'
import { useElementSize } from '@/lib/use-element-size'
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
import {
  MAP_BACKGROUND,
  MAP_INK,
  MAP_PAPER_FILL,
  MAP_PARCHMENT,
  NEUTRAL_ARMY_COLOR,
  PLAYER_PALETTE,
  PLAYER_RED,
  SEASON_VEILS,
  SELECTION_HALO,
  SELECTION_STROKE,
} from '@/lib/map-palette'

const OUTER_BORDER_WIDTH = 2.2
const PASSABLE_BORDER_WIDTH = 1.7
const SELECTION_STROKE_WIDTH = 2.5
/** Width of the brass halo inside a selected territory, in map units. */
const SELECTION_HALO_WIDTH = 8

interface MapViewerProps {
  map: MapData
  state: StateData
  onSelect?: (id: string | null) => void
  /** Selected territory, when the page controls the selection. */
  selectedId?: string | null
  /**
   * Offered every territory clicked before it is selected; returning true
   * takes the click (an open order form filling its field), and the
   * selection stays as it is.
   */
  onPick?: (id: string) => boolean
  /** Pans and zooms onto these territories each time its key changes. */
  frame?: MapFrame
  /** Territory lit while a link to it is hovered (charte § Identifiants). */
  highlightedId?: string | null
  /** Noble whose pawn wears the selection halo. */
  selectedNoble?: string | null
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
  selectedId: controlledSelectedId,
  onPick,
  frame,
  highlightedId = null,
  selectedNoble = null,
  supply,
  intentions = [],
  winterIntentions = [],
  showIntentions = false,
  intentionsColor = PLAYER_RED,
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
  const [ownSelectedId, setSelectedId] = useState<string | null>(null)
  const selectedId =
    controlledSelectedId === undefined ? ownSelectedId : controlledSelectedId
  const [legendOpen, setLegendOpen] = useState(false)

  const {
    mapWidth,
    mapHeight,
    outerBorders,
    sharedBorders,
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
    if (id !== null && onPick?.(id)) return
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

  const { svgRef, view, isDragging, pointerHandlers, zoomBy, frameOn } = useMapGestures({
    viewWidth,
    viewHeight,
    bandMarginX,
    bandMarginY,
    mapWidth,
    mapHeight,
    onTap: selectTerritory,
  })
  const svgSize = useElementSize(svgRef)
  // Frame once per key, after the SVG has its size.
  const framedKey = useRef<string | number | null>(null)
  useEffect(() => {
    if (!frame || frame.key === framedKey.current) return
    if (svgSize.width === 0 || svgSize.height === 0) return
    framedKey.current = frame.key
    const wanted = new Set(frame.territories)
    const bounds = boundsOf(
      map.territories
        .filter((territory) => wanted.has(territory.id))
        .map((territory) => territory.points),
    )
    if (bounds) frameOn(bounds, Math.max(mapWidth, mapHeight) * (frame.padding ?? 0.04))
  })
  // "meet": the viewBox fits the narrower side of the element.
  const pixelsPerUnit =
    svgSize.width > 0 && svgSize.height > 0
      ? Math.min(svgSize.width / viewWidth, svgSize.height / viewHeight) * view.k
      : 0
  // Ordinary territories show their name once it reads at 10 px; until the
  // element is measured, names show.
  const showOrdinaryNames =
    pixelsPerUnit === 0 ||
    nameScreenSize(annotationScale, pixelsPerUnit) >= NAME_MIN_SCREEN_PX
  const particularIds = useMemo(
    () =>
      particularTerritoryIds(
        state,
        (map.regions ?? []).map((region) => region.seed),
      ),
    [state, map.regions],
  )
  const labelLayouts = useMemo(
    () =>
      new Map(
        map.territories.map((territory) => [
          territory.id,
          territoryLabelLayout(
            territory,
            showOrdinaryNames || particularIds.has(territory.id),
          ),
        ]),
      ),
    [map.territories, particularIds, showOrdinaryNames],
  )

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
  const armsByOwner = resolveArmsByOwner(state, playerColors)
  const nobleAttributes = useMemo(() => nobleAttributesByCode(state), [state])
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
  const supplyColor = playerColors.get(selectedSupply?.armyOwner ?? '') ?? PLAYER_RED
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
          <rect width={mapWidth} height={mapHeight} fill={MAP_BACKGROUND} />
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
            <defs>
              <HeraldryDefs armsByOwner={armsByOwner} />
            </defs>

            <RegionBands
              bands={regionBands}
              regionStyleByID={regionStyleByID}
              regionsById={regionsById}
              territories={map.territories}
              mapWidth={mapWidth}
              mapHeight={mapHeight}
            />

            <g aria-label={t('map.terrains')}>
              {map.territories.map((territory) => (
                <Tooltip key={territory.id}>
                  <TooltipTrigger asChild>
                    <path
                      data-territory-id={territory.id}
                      d={pointsToPath(territory.points)}
                      fill={MAP_PAPER_FILL}
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

            <EngravedTerrainLayer
              territories={map.territories}
              annotationScale={annotationScale}
              label={t('map.terrainTextures')}
            />
            {regionsActive && (
              <RegionRings
                outlines={regionOutlines}
                regionStyleByID={regionStyleByID}
                regionHoleTesters={regionHoleTesters}
                annotationScale={annotationScale}
              />
            )}

            <g
              aria-label={t(
                state.season === 'winter' ? 'map.winterVeil' : 'map.seasonVeil',
              )}
              data-season-veil={state.season}
              data-winter-veil-overlay={state.season === 'winter' || undefined}
              pointerEvents="none"
            >
              <rect
                x={-bandMarginX}
                y={-bandMarginY}
                width={viewWidth}
                height={viewHeight}
                fill={SEASON_VEILS[state.season].color}
                opacity={SEASON_VEILS[state.season].opacity}
              />
            </g>
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

            <g aria-label={t('map.outerBorders')} pointerEvents="none">
              {outerBorders.map((border) => (
                <line
                  key={border.key}
                  x1={border.from[0]}
                  y1={border.from[1]}
                  x2={border.to[0]}
                  y2={border.to[1]}
                  stroke={MAP_INK}
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
                    stroke={MAP_INK}
                    strokeOpacity="0.75"
                    strokeWidth={PASSABLE_BORDER_WIDTH}
                    strokeDasharray={passableBorderDots}
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
              <ImpassableBorderChain icons={impassableBorderIcons} />
            </g>

            <g aria-label={t('map.selection')} pointerEvents="none">
              {map.territories.map((territory) => {
                if (territory.id === highlightedId && territory.id !== selectedId) {
                  return (
                    <path
                      key={territory.id}
                      data-highlighted-territory={territory.id}
                      d={pointsToPath(territory.points)}
                      fill={SELECTION_HALO}
                      fillOpacity="0.18"
                      stroke={SELECTION_STROKE}
                      strokeWidth={SELECTION_STROKE_WIDTH}
                      strokeDasharray="4 3"
                      strokeLinejoin="round"
                      vectorEffect="non-scaling-stroke"
                    />
                  )
                }
                if (territory.id !== selectedId) {
                  return null
                }

                const outline = pointsToPath(territory.points)
                return (
                  <g key={territory.id} data-selected-territory={territory.id}>
                    <path
                      d={outline}
                      fill="none"
                      stroke={SELECTION_HALO}
                      strokeOpacity="0.3"
                      strokeWidth={SELECTION_HALO_WIDTH * 2 * annotationScale}
                      strokeLinejoin="round"
                      clipPath={`url(#territory-clip-${territory.id})`}
                    />
                    <path
                      d={outline}
                      fill="none"
                      stroke={SELECTION_STROKE}
                      strokeWidth={SELECTION_STROKE_WIDTH}
                      strokeLinejoin="round"
                      vectorEffect="non-scaling-stroke"
                    />
                  </g>
                )
              })}
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
                  fill={MAP_PARCHMENT}
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
            {showOwnership && (
              <OwnershipLayer
                territories={map.territories}
                state={state}
                annotationScale={annotationScale}
                fiefs={state.fiefs ?? []}
                labelLayouts={labelLayouts}
              />
            )}
            <LiveLayer
              territories={map.territories}
              state={state}
              playerColors={playerColors}
              armsByOwner={armsByOwner}
              nobleAttributes={nobleAttributes}
              annotationScale={annotationScale}
              selectedNoble={selectedNoble}
            />

            <TerritoryLabels
              territories={map.territories}
              labelLayouts={labelLayouts}
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

            {/* Intentions draw on top in every season (layer order, charter). */}
            {showIntentions && (
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
          <div className="absolute left-3 top-3 z-10">
            <button
              type="button"
              aria-expanded={legendOpen}
              aria-label={t(legendOpen ? 'legend.hide' : 'legend.show')}
              title={t(legendOpen ? 'legend.hide' : 'legend.show')}
              className="flex size-9 items-center justify-center rounded-lg border border-line bg-paper-50 text-ink-700 shadow-md transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
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
              <div className="absolute left-0 top-11 w-72 max-w-[calc(100vw-1.5rem)]">
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
