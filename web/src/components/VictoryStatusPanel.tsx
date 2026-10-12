import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import type {
  Player,
  PlayerId,
  ScoreBreakdown,
  VictoryProjectionStatus,
  VictoryReading,
  VictoryStatus,
} from '@/types'

const STATUS_KEYS: Record<VictoryProjectionStatus, MessageKey> = {
  major: 'score.outcomeMajor',
  minor: 'score.outcomeMinor',
  failure: 'score.outcomeFailure',
  ongoing: 'sim.status.ongoing',
}

const STATUS_STYLES: Record<VictoryProjectionStatus, string> = {
  major: 'border-gold-500 bg-gold-100 text-gold-850',
  minor: 'border-steel-400 bg-steel-100 text-steel-700',
  failure: 'border-brass-700/50 bg-vermilion-250 text-vermilion-800',
  ongoing: 'border-line bg-paper-100 text-ink-700',
}

/** The simulation of every player, before and after the hypothetical actions. */
export interface PanelSimulation {
  current: Record<PlayerId, VictoryReading>
  projected: Record<PlayerId, VictoryReading>
}

function StatusBadge({ status, testId }: { status: VictoryProjectionStatus; testId: string }) {
  const { t } = useLanguage()
  return (
    <span
      data-testid={testId}
      className={`rounded border px-1.5 py-0.5 text-3xs font-bold uppercase tracking-[0.1em] ${STATUS_STYLES[status]}`}
    >
      {t(STATUS_KEYS[status])}
    </span>
  )
}

/**
 * One row per house: title score, threshold, mode and victory status. During a
 * simulation each row compares the real state with the hypothetical one.
 */
export function VictoryStatusPanel({
  players,
  scores,
  victory,
  simulation,
}: {
  players: Player[]
  /** Scores and thresholds of the state drawn in the dialog. */
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
  simulation?: PanelSimulation | null
}) {
  const { t } = useLanguage()
  const playerName = (id: string) => players.find((p) => p.id === id)?.name || id
  return (
    <section
      aria-label={t('sim.panel')}
      className="border-t border-line/60 bg-paper-80 px-4 py-2"
    >
      <ul className="grid gap-x-6 gap-y-1.5 sm:grid-cols-2 xl:grid-cols-4">
        {players.map((player) => {
          const after = simulation?.projected[player.id]
          const before = simulation?.current[player.id]
          const score = after?.score ?? scores?.[player.id]
          const goal = after?.victory ?? victory?.players[player.id]
          if (!score || !goal) return null
          const delta = before ? score.total - before.score.total : 0
          const missing = Math.max(goal.required - score.total, 0)
          return (
            <li key={player.id} className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
              <span
                aria-hidden="true"
                className="size-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: player.color }}
              />
              <span className="font-semibold">{player.name || player.id}</span>
              <span
                data-testid={`panel-score-${player.id}`}
                className="font-display text-sm font-semibold text-brass-700"
              >
                {score.total}
                <span className="text-xs text-ink-500">/{goal.required}</span>
              </span>
              {delta !== 0 && (
                <span
                  data-testid={`panel-delta-${player.id}`}
                  className={delta > 0 ? 'font-semibold text-success-600' : 'font-semibold text-vermilion-800'}
                >
                  {delta > 0 ? `+${delta}` : delta}
                </span>
              )}
              <span className="rounded border border-line bg-paper-50 px-1.5 py-0.5 text-3xs font-bold uppercase tracking-[0.1em] text-ink-700">
                {t(goal.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo')}
                {goal.mode === 'alliance' && goal.partner ? ` · ${playerName(goal.partner)}` : ''}
              </span>
              {after ? (
                <>
                  {before && before.status !== after.status && (
                    <>
                      <StatusBadge status={before.status} testId={`panel-status-before-${player.id}`} />
                      <span aria-hidden="true">→</span>
                    </>
                  )}
                  <StatusBadge status={after.status} testId={`panel-status-${player.id}`} />
                </>
              ) : null}
              <span className="text-ink-500">
                {missing > 0 ? t('sim.missing', { count: missing }) : t('sim.reached')}
              </span>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
