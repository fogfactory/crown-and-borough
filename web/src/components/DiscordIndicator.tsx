import { IconBrandDiscord } from '@tabler/icons-react'

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useLanguage } from '@/i18n/LanguageContext'

/**
 * Discreet sign, next to the submission dots, that the game announces to a
 * Discord webhook; its tooltip says what leaves.
 */
export function DiscordIndicator() {
  const { t } = useLanguage()
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            role="img"
            aria-label={t('discord.playerNotice')}
            className="inline-flex text-ink-500"
          >
            <IconBrandDiscord aria-hidden="true" className="size-4" />
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-64">
          <span>{t('discord.playerNotice')}</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
