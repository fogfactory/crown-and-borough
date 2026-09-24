import { GameIconGlyph } from '@/components/MapMarkers'
import { useLanguage } from '@/i18n/LanguageContext'
import { GAME_ICON_GLYPHS } from '@/lib/game-icon-glyphs'
import type { ImpassableBorderIcon } from '@/lib/map-borders'
import type { CalamityIcon, ScatteredIcon } from '@/lib/region-effect-icons'

/** Mountain glyphs strung along the impassable frontiers. */
export function ImpassableBorderChain({ icons }: { icons: ImpassableBorderIcon[] }) {
  return (
    <g data-impassable-chain="true">
      {icons.map((icon) => (
        <GameIconGlyph
          key={icon.key}
          glyph={GAME_ICON_GLYPHS['peaks']}
          x={icon.placement.x - icon.placement.size / 2}
          y={icon.placement.y - icon.placement.size / 2}
          size={icon.placement.size}
          fill="#30291f"
          stroke="#f5ecd9"
          strokeWidth={20}
          opacity={0.92}
          rotation={(icon.placement.rotation * 180) / Math.PI}
        />
      ))}
    </g>
  )
}

/** Active calamities, with a circle-slash badge when a card counters them. */
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
            <g pointerEvents="none">
              <circle
                cx={icon.placement.x}
                cy={icon.placement.y}
                r={icon.placement.size * 0.55}
                fill="none"
                stroke="#a84632"
                strokeWidth={icon.placement.size * 0.16}
                opacity={0.95}
              />
              <line
                x1={icon.placement.x - icon.placement.size * 0.4}
                y1={icon.placement.y - icon.placement.size * 0.4}
                x2={icon.placement.x + icon.placement.size * 0.4}
                y2={icon.placement.y + icon.placement.size * 0.4}
                stroke="#a84632"
                strokeWidth={icon.placement.size * 0.16}
                opacity={0.95}
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
