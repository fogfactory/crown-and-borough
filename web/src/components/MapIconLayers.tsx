import { GameIconGlyph } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import { GAME_ICON_GLYPHS } from '@/lib/game-icon-glyphs'
import type { ImpassableBorderIcon } from '@/lib/map-borders'
import type { CalamityIcon, ScatteredIcon } from '@/lib/region-effect-icons'
import { BORDER_SYMBOL_BOX, borderSymbolId } from '@/components/MapDecorations'
import { MAP_HALO, MAP_INK, MAP_PAPER_FILL } from '@/lib/map-palette'

/**
 * Icon series strung along the impassable frontiers: peaks, dense marsh or
 * escarpment, by the terrains the frontier separates.
 */
export function ImpassableBorderChain({ icons }: { icons: ImpassableBorderIcon[] }) {
  return (
    <g data-impassable-chain="true">
      {icons.map(({ key, kind, placement }) =>
        kind === 'peaks' ? (
          <g key={key} data-border-kind={kind}>
            <GameIconGlyph
              glyph={GAME_ICON_GLYPHS['peaks']}
              x={placement.x - placement.size / 2}
              y={placement.y - placement.size / 2}
              size={placement.size}
              fill={MAP_INK}
              stroke={MAP_HALO}
              strokeWidth={20}
              opacity={1}
            />
          </g>
        ) : (
          <use
            key={key}
            href={`#${borderSymbolId(kind)}`}
            data-border-kind={kind}
            transform={`translate(${placement.x} ${placement.y}) rotate(${
              (placement.rotation * 180) / Math.PI
            }) scale(${placement.size / BORDER_SYMBOL_BOX})`}
          />
        ),
      )}
    </g>
  )
}

/** Active calamities, struck through in ink when a card counters them. */
export function CalamityIconLayer({ icons }: { icons: CalamityIcon[] }) {
  const { t } = useLanguage()
  if (icons.length === 0) return null

  return (
    <g aria-label={t('map.calamityOverlay')} pointerEvents="none">
      {icons.map((icon) => (
        <g key={icon.key}>
          <GameIconGlyph
            glyph={icon.glyph}
            x={icon.placement.x - icon.placement.size / 2}
            y={icon.placement.y - icon.placement.size / 2}
            size={icon.placement.size}
            fill={icon.fill}
            stroke={icon.stroke}
            strokeWidth={icon.strokeWidth}
            opacity={icon.opacity}
            rotation={(icon.placement.rotation * 180) / Math.PI}
          />
          {icon.canceled && (
            <g pointerEvents="none" data-calamity-canceled="true" strokeLinecap="round">
              <line
                x1={icon.placement.x - icon.placement.size * 0.45}
                y1={icon.placement.y - icon.placement.size * 0.45}
                x2={icon.placement.x + icon.placement.size * 0.45}
                y2={icon.placement.y + icon.placement.size * 0.45}
                stroke={MAP_PAPER_FILL}
                strokeWidth={icon.placement.size * 0.2}
              />
              <line
                x1={icon.placement.x - icon.placement.size * 0.45}
                y1={icon.placement.y - icon.placement.size * 0.45}
                x2={icon.placement.x + icon.placement.size * 0.45}
                y2={icon.placement.y + icon.placement.size * 0.45}
                stroke={MAP_INK}
                strokeWidth={icon.placement.size * 0.1}
              />
            </g>
          )}
        </g>
      ))}
    </g>
  )
}

/** Drafted deck cards scattered over their target territory or region. */
export function CardIconLayer({ icons }: { icons: ScatteredIcon[] }) {
  const { t } = useLanguage()
  if (icons.length === 0) return null

  return (
    <g aria-label={t('map.cardOverlay')} pointerEvents="none">
      {icons.map((icon) => (
        <GameIconGlyph
          key={icon.key}
          glyph={icon.glyph}
          x={icon.placement.x - icon.placement.size / 2}
          y={icon.placement.y - icon.placement.size / 2}
          size={icon.placement.size}
          fill={icon.fill}
          stroke={icon.stroke}
          strokeWidth={icon.strokeWidth}
          opacity={icon.opacity}
          rotation={(icon.placement.rotation * 180) / Math.PI}
        />
      ))}
    </g>
  )
}
