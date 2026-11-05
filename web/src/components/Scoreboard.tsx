import { ShieldIcon } from '@/components/ArmsIcons'
import { useLanguage } from '@/i18n/LanguageContext'
import { playerArms } from '@/lib/heraldry'
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
  activePlayer,
}: {
  players: Player[]
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
  finished?: boolean
  winners?: string[]
  minorWinner?: string | null
  /** The viewer: its score reads in brass. */
  activePlayer?: string | null
}) {
  const { t } = useLanguage()
  const playerName = (id: string) => players.find((p) => p.id === id)?.name || id
  return (
    <section aria-labelledby="scoreboard-title" className="space-y-2">
      <h2
        id="scoreboard-title"
        className="font-display text-base font-semibold text-ink-900"
      >
        {t('app.scores')}
      </h2>
      <ul className="grid gap-2 min-[420px]:grid-cols-2">
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
          const active = player.id === activePlayer
          return (
            <li
              key={player.id}
              data-score-card={player.id}
              className={`rounded-md border bg-paper-50 p-2.5 ${active ? 'border-brass-700' : 'border-line'}`}
            >
              <div className="flex items-center gap-2">
                <ShieldIcon id={`score-arms-${player.id}`} arms={playerArms(player)} />
                <span className="min-w-0 flex-1 truncate text-sm font-semibold">
                  {player.name || player.id}
                </span>
                <span
                  data-score-total
                  className={`shrink-0 text-2xl font-bold leading-none ${active ? 'text-brass-700' : 'text-ink-900'}`}
                >
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
                <p className="mt-2 text-xs text-ink-500">
                  {t('score.goal', { required: goal.required })}
                  {' · '}
                  <span data-testid={`victory-mode-${player.id}`}>
                    {t(
                      goal.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo',
                    )}
                  </span>
                  {outcome && (
                    <>
                      {' · '}
                      <span
                        data-testid={`victory-status-${player.id}`}
                        className={`font-semibold ${
                          outcome === 'Major' || outcome === 'Reached'
                            ? 'text-brass-700'
                            : outcome === 'Failure'
                              ? 'text-vermilion-700'
                              : 'text-ink-700'
                        }`}
                      >
                        {t(
                          `score.${outcome === 'Reached' ? 'goalReached' : `outcome${outcome}`}` as MessageKey,
                        )}
                      </span>
                    </>
                  )}
                </p>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
