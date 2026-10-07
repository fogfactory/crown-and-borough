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
          const goal = victory?.players[player.id]
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
                {scoreKeys
                  .filter(([key]) => key !== 'alliance' || (score.alliance ?? 0) > 0)
                  .map(([key, labelKey]) => (
                  <div key={key} className="contents">
                    <dt className="text-[#806f57]">{t(labelKey)}</dt>
                    <dd className="text-right font-medium">{score[key] ?? 0}</dd>
                  </div>
                ))}
              </dl>
              {goal && (
                <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
                  <span className="text-[#806f57]">
                    {t(goal.mode === 'alliance' ? 'score.goalCombined' : 'score.goal', {
                      required: goal.required,
                    })}
                  </span>
                  <span
                    data-testid={`victory-mode-${player.id}`}
                    className={`rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-[0.1em] ${
                      goal.mode === 'alliance'
                        ? 'border-[#8a929b] bg-[#e4e7ea] text-[#3f464d]'
                        : 'border-[#b8860b] bg-[#f8e8ae] text-[#6b4e0a]'
                    }`}
                  >
                    {t(goal.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo')}
                    {goal.mode === 'alliance' && goal.partner
                      ? ` ${t('score.alliancePartner', { partner: playerName(goal.partner) })}`
                      : ''}
                  </span>
                </div>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
