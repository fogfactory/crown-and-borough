import { ArmsField, BannerGlyph } from '@/components/MapHeraldry'
import {
  BANNER_HEIGHT,
  BANNER_PATH,
  BANNER_WIDTH,
  SHIELD_HEIGHT,
  SHIELD_PATH,
  SHIELD_WIDTH,
  type CrownKind,
  type ResolvedArms,
} from '@/lib/heraldry'
import { MAP_INK, SHIELD_NUMBER } from '@/lib/map-palette'

/**
 * A banner in the given arms, sized for a line of text (legend, territory
 * cartouche); crowned for a fief capital, or marked with its fief's trigram.
 */
export function BannerIcon({
  id,
  arms,
  crown,
  attachedTo,
}: {
  /** Unique id of the drawing, for its symbol and clip. */
  id: string
  arms: ResolvedArms
  crown?: CrownKind
  attachedTo?: string
}) {
  return (
    <svg
      className="h-5 w-6 shrink-0"
      viewBox="-3 -24 20 26"
      aria-hidden="true"
      data-banner-icon={crown ?? attachedTo ?? 'plain'}
    >
      <defs>
        <symbol id={`${id}-flag`} overflow="visible">
          <ArmsField
            arms={arms}
            outline={BANNER_PATH}
            clipId={`${id}-clip`}
            box={[0, 0, BANNER_WIDTH, BANNER_HEIGHT]}
          />
        </symbol>
      </defs>
      <BannerGlyph
        owner={id}
        flagHref={`#${id}-flag`}
        crown={crown}
        attachedTo={attachedTo}
      />
    </svg>
  )
}

/** A shield in the given arms, with an army's strength when given. */
export function ShieldIcon({
  id,
  arms,
  size,
}: {
  id: string
  arms: ResolvedArms
  size?: number
}) {
  return (
    <svg
      className="h-5 w-6 shrink-0"
      viewBox="-10 -11 20 22"
      aria-hidden="true"
      data-shield-icon
    >
      <ArmsField
        arms={arms}
        outline={SHIELD_PATH}
        clipId={`${id}-clip`}
        box={[-SHIELD_WIDTH / 2, -SHIELD_HEIGHT / 2, SHIELD_WIDTH, SHIELD_HEIGHT]}
      />
      {size !== undefined && (
        <text
          fill={SHIELD_NUMBER}
          fontSize={10}
          fontWeight="700"
          textAnchor="middle"
          dominantBaseline="central"
          stroke={MAP_INK}
          strokeWidth={2.4}
          strokeLinejoin="round"
          paintOrder="stroke"
        >
          {size}
        </text>
      )}
    </svg>
  )
}
