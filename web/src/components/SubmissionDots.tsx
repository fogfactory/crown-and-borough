import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

export interface SubmissionDotPlayer {
  id: string
  name: string
  color: string
  submitted: boolean
  /** Whether the current turn is still awaiting this player. Defaults to
   * true: a caller that hasn't fetched this yet should still show a player
   * as waiting rather than falsely as having nothing to submit. */
  required?: boolean
  isYou?: boolean
}

type DotStatus = 'submitted' | 'waiting' | 'notRequired'

function statusOf(player: SubmissionDotPlayer): DotStatus {
  if (player.submitted) return 'submitted'
  return (player.required ?? true) ? 'waiting' : 'notRequired'
}

const STATUS_LABEL_KEY: Record<
  DotStatus,
  'online.submitted' | 'online.waiting' | 'online.notRequired'
> = {
  submitted: 'online.submitted',
  waiting: 'online.waiting',
  notRequired: 'online.notRequired',
}

/**
 * Compact per-player order submission status: one dot per player, filled
 * when the player submitted, hollow while the turn still waits for them, and
 * muted when they have nothing to submit this turn (eliminated, no free or
 * hostage noble, no playable card outside winter).
 */
export function SubmissionDots({ players }: { players: SubmissionDotPlayer[] }) {
  const { t } = useLanguage()

  if (players.length === 0) {
    return null
  }

  return (
    <TooltipProvider delayDuration={100}>
      <div
        role="group"
        aria-label={t('app.submissionStatus')}
        className="flex items-center gap-1"
      >
        {players.map((player) => {
          const status = statusOf(player)
          return (
            <Tooltip key={player.id}>
              <TooltipTrigger asChild>
                <span
                  role="img"
                  aria-label={`${player.name} · ${t(STATUS_LABEL_KEY[status])}`}
                  className={cn(
                    'size-2.5 rounded-full border',
                    status === 'submitted' && 'border-[#30291f]/30',
                    status === 'waiting' && 'border-dashed opacity-80',
                    status === 'notRequired' && 'border-[#806f57]/30 opacity-40',
                  )}
                  style={
                    status === 'submitted'
                      ? { backgroundColor: player.color }
                      : status === 'waiting'
                        ? { borderColor: player.color }
                        : undefined
                  }
                />
              </TooltipTrigger>
              <TooltipContent>
                <span>
                  {player.name}
                  {player.isYou ? ` · ${t('online.you')}` : ''} ·{' '}
                  {t(STATUS_LABEL_KEY[status])}
                </span>
              </TooltipContent>
            </Tooltip>
          )
        })}
      </div>
    </TooltipProvider>
  )
}
