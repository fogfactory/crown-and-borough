import { useLanguage } from '@/i18n/LanguageContext'
import { royalIssuerOf } from '@/lib/royal'
import type { Noble, Player, StateData } from '@/types'

/** Throne, king, queen, regent and crown fiefs (politique.md). */
export function CrownPanel({ state }: { state: StateData }) {
  const { t } = useLanguage()
  const crownFiefs = (state.fiefs ?? []).filter((fief) => fief.crown)
  if (!state.king && !state.queen && crownFiefs.length === 0) return null

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
        {noble.status === 'dungeon' && ` · ${t('crown.dungeon')}`}
      </span>
    )
  }

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
