import { useState } from 'react'
import { Popover } from 'radix-ui'
import { IconBook } from '@tabler/icons-react'

import type { RulesSection } from '@/components/RulesPanel'
import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

interface HelpPopoverProps {
  /** Text of the thing explained, naming the "?" button for screen readers. */
  topic: string
  /** Full explanation. */
  help: string
  /** Rules section the popover links to. */
  rulesSection?: RulesSection
  onOpenRules?: (section: RulesSection) => void
  /** `card`: a small badge to put on a hand card; `section`: next to a title. */
  variant?: 'section' | 'card'
  className?: string
}

/**
 * The "?" button of the command post (charte-graphique.md § Textes d'aide):
 * it opens a popover with the explanation and a link to the rule. A popover
 * rather than a tooltip, so that it also opens by touch.
 */
export function HelpPopover({
  topic,
  help,
  rulesSection,
  onOpenRules,
  variant = 'section',
  className,
}: HelpPopoverProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={t('help.about', { topic })}
          title={t('help.about', { topic })}
          className={cn(
            variant === 'section'
              ? // A 24 px circle inside a 44 px touch target on phones.
                '-my-2.5 -mr-2.5 flex size-11 shrink-0 items-center justify-center lg:m-0 lg:size-6'
              : 'absolute top-0.5 right-0.5 flex size-5 items-center justify-center',
            className,
          )}
        >
          <span
            aria-hidden="true"
            className={cn(
              'flex items-center justify-center rounded-full border border-line-strong bg-paper-50 font-semibold text-ink-700',
              variant === 'section' ? 'size-6 text-xs' : 'size-4 text-2xs',
            )}
          >
            ?
          </span>
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={6}
          className="z-50 w-72 max-w-[calc(100vw-1.5rem)] space-y-2 rounded-md border border-line bg-paper-50 p-3 text-sm text-ink-700 shadow-panel"
        >
          <p>{help}</p>
          {rulesSection && onOpenRules && (
            <button
              type="button"
              className="inline-flex items-center gap-1.5 text-sm font-semibold text-brass-700 underline decoration-dotted underline-offset-2 hover:decoration-solid"
              onClick={() => {
                setOpen(false)
                onOpenRules(rulesSection)
              }}
            >
              <IconBook aria-hidden="true" className="size-4" />
              {t('help.seeRule')}
            </button>
          )}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
