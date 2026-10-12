import { useLanguage } from '@/i18n/LanguageContext'
import type { Noble, Player } from '@/types'
import { NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'

/** A title holder line: owner color dot, name, code and owner, or "vacant". */
export function NobleHolder({
  code,
  nobles,
  players,
}: {
  code: string | undefined
  nobles: Noble[]
  players: Player[]
}) {
  const { t } = useLanguage()
  const noble = code ? nobles.find((candidate) => candidate.code === code) : undefined
  if (!noble) return <span className="italic text-ink-500">{t('map.hierarchyVacant')}</span>
  const owner = players.find((player) => player.id === noble.owner)
  return (
    <span className="inline-flex items-center gap-1">
      <span
        aria-hidden="true"
        className="inline-block size-2.5 rounded-full border border-ink-900/40"
        style={{ backgroundColor: owner?.color ?? NEUTRAL_ARMY_COLOR }}
      />
      {noble.name} ({noble.code}) · {owner?.name ?? noble.owner}
      {noble.status === 'dungeon' && ` · ${t('crown.dungeon')}`}
    </span>
  )
}
