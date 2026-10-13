import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type { Player, ScoreBreakdown, VictoryStatus } from '@/types'

const scoreKeys: Array<[keyof ScoreBreakdown, MessageKey]> = [
  ['titles', 'score.titles'],
  ['alliance', 'score.alliance'],
]

const emptyScore: ScoreBreakdown = {
  titles: 0,
  alliance: 0,
  total: 0,
}

export function Scoreboard({
  players,
  scores,
  victory,
  finished,
  winners,
  minorWinner,
}: {
  players: Player[]
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
  finished?: boolean
  winners?: string[]
  minorWinner?: string | null
}) {
  const { t } = useLanguage()
  const playerName = (id: string) => players.find((p) => p.id === id)?.name || id
  return (
    <section
      aria-labelledby="scoreboard-title"
      className="rounded-xl border border-line/60 bg-paper-100 p-3"
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id="scoreboard-title"
          className="font-display text-lg font-semibold text-ink-900 sm:text-xl"
        >
          {t('app.scores')}
        </h2>
      </div>
      <ul className="mt-2 grid gap-2 min-[420px]:grid-cols-2">
        {players.map((player) => {
          const score = scores?.[player.id] ?? emptyScore
          const goal = victory?.players[player.id]
          // A player is covered by a major victory directly or through the
          // spouse sharing their joint win.
          const outcome = finished
            ? winners?.includes(player.id) ||
              (goal?.partner !== undefined && winners?.includes(goal.partner))
              ? 'Major'
              : minorWinner === player.id
                ? 'Minor'
                : 'Failure'
            : goal && goal.required >= 1 && score.total >= goal.required
              ? 'Reached'
              : null
          return (
            <li
              key={player.id}
              className="rounded-lg border border-line/50 bg-paper-50 p-2.5"
            >
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-3 shrink-0 rounded-full border border-ink-900/30"
                  style={{ backgroundColor: player.color }}
                />
                <span className="min-w-0 flex-1 truncate text-sm font-semibold">
                  {player.name || player.id}
                </span>
                <span className="shrink-0 font-display text-lg font-semibold text-brass-700">
                  {score.total}
                </span>
              </div>
              <dl className="mt-2 grid grid-cols-[1fr_auto] gap-x-3 gap-y-1 text-xs">
                {scoreKeys
                  .filter(([key]) => key !== 'alliance' || (score.alliance ?? 0) > 0)
                  .map(([key, labelKey]) => (
                    <div key={key} className="contents">
                      <dt className="text-ink-500">{t(labelKey)}</dt>
                      <dd className="text-right font-medium">{score[key] ?? 0}</dd>
                    </div>
                  ))}
                {goal?.mode === 'alliance' && goal.partner && (
                  <div className="contents" data-testid={`ally-${player.id}`}>
                    <dt className="text-ink-500">
                      {t('score.ally', { partner: playerName(goal.partner) })}
                    </dt>
                    <dd className="text-right font-medium">{score.ally ?? 0}</dd>
                  </div>
                )}
              </dl>
              {goal && (
                <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
                  <span className="text-ink-500">
                    {t('score.goal', { required: goal.required })}
                  </span>
                  <span
                    data-testid={`victory-mode-${player.id}`}
                    className={`rounded border px-1.5 py-0.5 text-2xs font-bold uppercase tracking-[0.1em] ${
                      goal.mode === 'alliance'
                        ? 'border-line-strong bg-paper-100 text-ink-700'
                        : 'border-brass-500 bg-brass-300 text-ink-900'
                    }`}
                  >
                    {t(
                      goal.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo',
                    )}
                  </span>
                  {outcome && (
                    <span
                      data-testid={`victory-status-${player.id}`}
                      className={`rounded border px-1.5 py-0.5 text-2xs font-bold uppercase tracking-[0.1em] ${
                        outcome === 'Major'
                          ? 'border-brass-500 bg-brass-300 text-ink-900'
                          : outcome === 'Failure'
                            ? 'border-brass-700/50 bg-vermilion-700/30 text-vermilion-700'
                            : 'border-line-strong bg-paper-100 text-ink-700'
                      }`}
                    >
                      {t(
                        `score.${outcome === 'Reached' ? 'goalReached' : `outcome${outcome}`}` as MessageKey,
                      )}
                    </span>
                  )}
                </div>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
