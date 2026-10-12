import { useLanguage } from '@/i18n/LanguageContext'
import type { Noble, Player } from '@/types'

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
  if (!noble) return <span className="italic text-[#806f57]">{t('map.hierarchyVacant')}</span>
  const owner = players.find((player) => player.id === noble.owner)
  return (
    <span className="inline-flex items-center gap-1">
      <span
        aria-hidden="true"
        className="inline-block size-2.5 rounded-full border border-[#30291f]/40"
        style={{ backgroundColor: owner?.color ?? '#6b7280' }}
      />
      {noble.name} ({noble.code}) · {owner?.name ?? noble.owner}
      {noble.status === 'dungeon' && ` · ${t('crown.dungeon')}`}
    </span>
  )
}
