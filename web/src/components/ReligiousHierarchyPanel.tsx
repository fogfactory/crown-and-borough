import { useLanguage } from '@/i18n/LanguageContext'
import type { Noble, Player, StateData } from '@/types'

/** Pope, cardinals and bishops in place, with the player owning each title. */
export function ReligiousHierarchyPanel({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const bishoprics = state.bishoprics ?? []
  const cardinals = state.cardinals ?? []
  if (bishoprics.length === 0 && cardinals.length === 0 && !state.pope) return null

  const nobleByCode = new Map(state.nobles.map((noble) => [noble.code, noble]))
  const playerById = new Map(state.players.map((player) => [player.id, player]))

  const holder = (code: string | undefined) => {
    const noble: Noble | undefined = code ? nobleByCode.get(code) : undefined
    if (!noble) return <span className="italic text-[#806f57]">{t('map.hierarchyVacant')}</span>
    const owner: Player | undefined = playerById.get(noble.owner)
    return (
      <span className="inline-flex items-center gap-1">
        <span
          aria-hidden="true"
          className="inline-block size-2.5 rounded-full border border-[#30291f]/40"
          style={{ backgroundColor: owner?.color ?? '#6b7280' }}
        />
        {noble.name} ({noble.code}) · {owner?.name ?? noble.owner}
      </span>
    )
  }

  return (
    <details
      data-religious-hierarchy
      className="absolute left-3 top-3 z-10 max-w-[18rem] rounded-lg border border-[#b7a786] bg-[#fffaf0]/95 p-2 text-xs text-[#594b3c] shadow-md"
    >
      <summary className="cursor-pointer font-bold uppercase tracking-[0.12em]">
        {t('map.hierarchyTitle')}
      </summary>
      <dl className="mt-2 space-y-1">
        <div>
          <dt className="font-semibold">{t('map.hierarchyPope')}</dt>
          <dd>{holder(state.pope)}</dd>
        </div>
        <div>
          <dt className="font-semibold">{t('map.hierarchyCardinals')}</dt>
          {cardinals.length === 0 ? (
            <dd>{holder(undefined)}</dd>
          ) : (
            cardinals.map((code) => <dd key={code}>{holder(code)}</dd>)
          )}
        </div>
        <div>
          <dt className="font-semibold">{t('map.hierarchyBishops')}</dt>
          {bishoprics.map((bishopric) => (
            <dd key={bishopric.region} data-bishopric={bishopric.region}>
              <span className="font-medium">{bishopric.name}</span> :{' '}
              {holder(bishopric.bishop)}
            </dd>
          ))}
        </div>
      </dl>
    </details>
  )
}
