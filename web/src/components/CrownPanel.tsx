import { NobleHolder } from '@/components/NobleHolder'
import { useLanguage } from '@/i18n/LanguageContext'
import { royalIssuerOf } from '@/lib/royal'
import type { StateData } from '@/types'

/** Throne, king, queen, regent and crown fiefs (politique.md). */
export function CrownPanel({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const crownFiefs = (state.fiefs ?? []).filter((fief) => fief.crown)
  if (!state.king && !state.queen && crownFiefs.length === 0) return null

  const holder = (code: string | undefined) => (
    <NobleHolder code={code} nobles={state.nobles} players={state.players} />
  )
  const issuer = royalIssuerOf(state)
  const regent = issuer && issuer.code !== state.king ? issuer : undefined

  return (
    <details
      data-crown-panel
      className="max-w-[18rem] rounded-lg border border-[#b7a786] bg-[#fffaf0]/95 p-2 text-xs text-[#594b3c] shadow-md"
    >
      <summary className="cursor-pointer font-bold uppercase tracking-[0.12em]">
        {t('crown.title')}
      </summary>
      <dl className="mt-2 space-y-1">
        <div>
          <dt className="font-semibold">{t('crown.king')}</dt>
          <dd>{holder(state.king)}</dd>
        </div>
        {state.queen && (
          <div>
            <dt className="font-semibold">
              {t(state.queenDowager ? 'crown.queenDowager' : 'crown.queen')}
            </dt>
            <dd>{holder(state.queen)}</dd>
          </div>
        )}
        {regent && (
          <div data-crown-regent>
            <dt className="font-semibold">{t('crown.regent')}</dt>
            <dd>{holder(regent.code)}</dd>
          </div>
        )}
        {crownFiefs.length > 0 && (
          <div>
            <dt className="font-semibold">{t('crown.fiefs')}</dt>
            {crownFiefs.map((fief) => (
              <dd key={fief.capital} data-crown-fief={fief.capital}>
                {fief.capital} · {fief.territories.length}
              </dd>
            ))}
          </div>
        )}
      </dl>
    </details>
  )
}
