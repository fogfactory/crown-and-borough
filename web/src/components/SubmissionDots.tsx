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
  isYou?: boolean
}

/**
 * Compact per-player order submission status: one dot per player, filled when
 * the player submitted, hollow while waiting.
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
        {players.map((player) => (
          <Tooltip key={player.id}>
            <TooltipTrigger asChild>
              <span
                role="img"
                aria-label={`${player.name} · ${t(player.submitted ? 'online.submitted' : 'online.waiting')}`}
                className={cn(
                  'size-2.5 rounded-full border',
                  player.submitted ? 'border-[#30291f]/30' : 'border-dashed opacity-80',
                )}
                style={
                  player.submitted
                    ? { backgroundColor: player.color }
                    : { borderColor: player.color }
                }
              />
            </TooltipTrigger>
            <TooltipContent>
              <span>
                {player.name}
                {player.isYou ? ` · ${t('online.you')}` : ''} ·{' '}
                {t(player.submitted ? 'online.submitted' : 'online.waiting')}
              </span>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  )
}
