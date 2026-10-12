import { NobleHolder } from '@/components/NobleHolder'
import { useLanguage } from '@/i18n/LanguageContext'
import type { StateData } from '@/types'

/** Pope, cardinals and bishops in place, with the player owning each title. */
export function ReligiousHierarchyPanel({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const bishoprics = state.bishoprics ?? []
  const cardinals = state.cardinals ?? []
  if (bishoprics.length === 0 && cardinals.length === 0 && !state.pope) return null

  const holder = (code: string | undefined) => (
    <NobleHolder code={code} nobles={state.nobles} players={state.players} />
  )
  return (
    <details
      data-religious-hierarchy
      className="max-w-[18rem] rounded-lg border border-[#b7a786] bg-[#fffaf0]/95 p-2 text-xs text-[#594b3c] shadow-md"
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
