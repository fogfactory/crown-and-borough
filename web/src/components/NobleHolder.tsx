import { NoblePawn } from '@/components/MapHeraldry'
import { NobleRef } from '@/components/MapLinks'
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

/**
 * A title holder line: pawn, name and code, then the owner's tincture as a
 * pastille (the owner's name stays readable for screen readers and on
 * hover), or "vacant".
 */
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
  const ownerName = owner?.name ?? noble.owner
  return (
    <span className="inline-flex flex-wrap items-center gap-x-1.5">
      <NoblePawnIcon
        noble={noble}
        color={owner?.color ?? NEUTRAL_ARMY_COLOR}
        attributes={attributes.get(noble.code)}
      />
      <NobleRef code={noble.code} name={noble.name} />
      <span
        title={ownerName}
        aria-hidden="true"
        data-owner-pastille={noble.owner}
        className="size-2.5 shrink-0 rounded-full border border-ink-900/30"
        style={{ backgroundColor: owner?.color ?? NEUTRAL_ARMY_COLOR }}
      />
      <span className="sr-only">· {ownerName}</span>
      {noble.status === 'dungeon' && (
        <span className="text-ink-500">· {t('crown.dungeon')}</span>
      )}
    </span>
  )
}
