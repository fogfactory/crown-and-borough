import { NoblePawn } from '@/components/MapHeraldry'
import { useLanguage } from '@/i18n/LanguageContext'
import type { NobleAttributes } from '@/lib/noble-attributes'
import type { Noble, Player } from '@/types'
import { NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'

/** A noble's pawn with its title attributes, sized for a line of text. */
export function NoblePawnIcon({
  noble,
  color,
  attributes,
  label,
}: {
  noble: Pick<Noble, 'status'>
  color: string
  attributes?: NobleAttributes
  /** Accessible name; the pawn is decorative without it. */
  label?: string
}) {
  return (
    <svg
      // Tall enough for a crown over a tiara (pope and baron).
      viewBox="-11 -17 18 26"
      className="inline-block h-[1.5em] w-[1.05em] shrink-0 align-[-0.35em]"
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    >
      <NoblePawn color={color} status={noble.status} attributes={attributes} />
    </svg>
  )
}

/** A title holder line: pawn, name, code and owner, or "vacant". */
export function NobleHolder({
  code,
  nobles,
  players,
  attributes,
}: {
  code: string | undefined
  nobles: Noble[]
  players: Player[]
  attributes: Map<string, NobleAttributes>
}) {
  const { t } = useLanguage()
  const noble = code ? nobles.find((candidate) => candidate.code === code) : undefined
  if (!noble)
    return <span className="italic text-ink-500">{t('map.hierarchyVacant')}</span>
  const owner = players.find((player) => player.id === noble.owner)
  return (
    <span className="inline-flex items-center gap-1">
      <NoblePawnIcon
        noble={noble}
        color={owner?.color ?? NEUTRAL_ARMY_COLOR}
        attributes={attributes.get(noble.code)}
      />
      {noble.name} ({noble.code}) · {owner?.name ?? noble.owner}
      {noble.status === 'dungeon' && ` · ${t('crown.dungeon')}`}
    </span>
  )
}
