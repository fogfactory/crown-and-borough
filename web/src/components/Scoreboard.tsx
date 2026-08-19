import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Player, ScoreBreakdown } from '@/types'

const scoreKeys: Array<[keyof ScoreBreakdown, MessageKey]> = [
  ['territories', 'score.territories'],
  ['villages', 'score.villages'],
  ['mills', 'score.mills'],
  ['castles', 'score.castles'],
  ['nobles', 'score.nobles'],
  ['troops', 'score.troops'],
  ['resources', 'score.resources'],
]

const emptyScore: ScoreBreakdown = {
  territories: 0,
  villages: 0,
  mills: 0,
  castles: 0,
  nobles: 0,
  troops: 0,
  resources: 0,
  total: 0,
}

export function Scoreboard({
  players,
  scores,
}: {
  players: Player[]
  scores?: Record<string, ScoreBreakdown>
}) {
  const { t } = useLanguage()
  return (
    <section
      aria-labelledby="scoreboard-title"
      className="rounded-xl border border-[#b7a786]/60 bg-[#f8f0e2] p-4"
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id="scoreboard-title"
          className="font-serif text-xl font-semibold text-[#30291f]"
        >
          {t('app.scores')}
        </h2>
      </div>
      <ul className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {players.map((player) => {
          const score = scores?.[player.id] ?? emptyScore
          return (
            <li
              key={player.id}
              className="rounded-lg border border-[#b7a786]/50 bg-[#fffaf0] p-3"
            >
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-3 shrink-0 rounded-full border border-[#30291f]/30"
                  style={{ backgroundColor: player.color }}
                />
                <span className="min-w-0 flex-1 truncate text-sm font-semibold">
                  {player.name || player.id}
                </span>
                <span className="shrink-0 font-serif text-lg font-semibold text-[#a84632]">
                  {score.total}
                </span>
              </div>
              <dl className="mt-2 grid grid-cols-[1fr_auto] gap-x-3 gap-y-1 text-xs">
                {scoreKeys.map(([key, labelKey]) => (
                  <div key={key} className="contents">
                    <dt className="text-[#806f57]">{t(labelKey)}</dt>
                    <dd className="text-right font-medium">{score[key]}</dd>
                  </div>
                ))}
              </dl>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
