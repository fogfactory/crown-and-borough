import { ArmsField, ArmyGlyph, ControlShield, Soldier } from '@/components/MapHeraldry'
import {
  BANNER_HEIGHT,
  BANNER_PATH,
  BANNER_WIDTH,
  SHIELD_HEIGHT,
  SHIELD_PATH,
  SHIELD_WIDTH,
  greyArms,
  type CrownKind,
  type ResolvedArms,
} from '@/lib/heraldry'

/**
 * The shield of a controller in the given arms, sized for a line of text
 * (legend, territory cartouche); crowned for a fief capital, or marked with
 * its fief's trigram.
 */
export function ControlShieldIcon({
  id,
  arms,
  crown,
  attachedTo,
  grey = false,
}: {
  /** Unique id of the drawing, for its symbol and clip. */
  id: string
  arms: ResolvedArms
  crown?: CrownKind
  attachedTo?: string
  /** The shield of a territory occupied against its controller. */
  grey?: boolean
}) {
  return (
    <svg
      className="h-6 w-6 shrink-0"
      viewBox="-8 -23 16 32"
      aria-hidden="true"
      data-control-icon={grey ? 'occupied' : (crown ?? attachedTo ?? 'plain')}
    >
      <defs>
        <symbol id={`${id}-shield`} overflow="visible">
          <ArmsField
            arms={grey ? greyArms(arms) : arms}
            outline={SHIELD_PATH}
            clipId={`${id}-clip`}
            box={[-SHIELD_WIDTH / 2, -SHIELD_HEIGHT / 2, SHIELD_WIDTH, SHIELD_HEIGHT]}
          />
        </symbol>
      </defs>
      <ControlShield
        owner={id}
        shieldHref={`#${id}-shield`}
        grey={grey}
        crown={crown}
        attachedTo={attachedTo}
      />
    </svg>
  )
}

/** A shield in the given arms: the house of a noble or a player. */
export function ShieldIcon({ id, arms }: { id: string; arms: ResolvedArms }) {
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
    </svg>
  )
}

/** An army in the given arms: its standard-bearer and a banner with its strength. */
export function ArmyIcon({
  id,
  arms,
  size,
}: {
  id: string
  arms: ResolvedArms
  size: number
}) {
  return (
    <svg
      className="h-6 w-6 shrink-0"
      viewBox="-8 -26 18 28"
      aria-hidden="true"
      data-army-icon
    >
      <defs>
        <symbol id={`${id}-soldier`} overflow="visible">
          <Soldier arms={arms} clipId={`${id}-soldier-clip`} />
        </symbol>
        <symbol id={`${id}-banner`} overflow="visible">
          <ArmsField
            arms={arms}
            outline={BANNER_PATH}
            clipId={`${id}-banner-clip`}
            box={[0, 0, BANNER_WIDTH, BANNER_HEIGHT]}
          />
        </symbol>
      </defs>
      <ArmyGlyph
        owner={id}
        size={size}
        seed={id}
        compact
        hrefs={{ soldier: `#${id}-soldier`, banner: `#${id}-banner` }}
      />
    </svg>
  )
}
