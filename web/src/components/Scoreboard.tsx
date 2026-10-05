import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Player, ScoreBreakdown, VictoryStatus } from '@/types'

const scoreKeys: Array<[keyof ScoreBreakdown, MessageKey]> = [['titles', 'score.titles']]

const emptyScore: ScoreBreakdown = {
  titles: 0,
  total: 0,
}

export function Scoreboard({
  players,
  scores,
  victory,
}: {
  players: Player[]
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
}) {
  const { t } = useLanguage()
  const playerName = (id: string) => players.find((p) => p.id === id)?.name || id
  return (
    <section
      aria-labelledby="scoreboard-title"
      className="rounded-xl border border-[#b7a786]/60 bg-[#f8f0e2] p-3"
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id="scoreboard-title"
          className="font-serif text-lg font-semibold text-[#30291f] sm:text-xl"
        >
          {t('app.scores')}
        </h2>
      </div>
      <ul className="mt-2 grid gap-2 min-[420px]:grid-cols-2">
        {players.map((player) => {
          const score = scores?.[player.id] ?? emptyScore
          return (
            <li
              key={player.id}
              className="rounded-lg border border-[#b7a786]/50 bg-[#fffaf0] p-2.5"
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
                    <dd className="text-right font-medium">{score[key] ?? 0}</dd>
                  </div>
                ))}
              </dl>
              {victory?.players[player.id] && (
                <p className="mt-2 text-xs text-[#806f57]">
                  {victory.players[player.id].mode === 'alliance'
                    ? t('score.goalAlliance', {
                        required: victory.players[player.id].required,
                        partner: playerName(victory.players[player.id].partner ?? ''),
                      })
                    : t('score.goalSolo', {
                        required: victory.players[player.id].required,
                      })}
                </p>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
