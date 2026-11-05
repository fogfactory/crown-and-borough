import { useState, type ReactNode } from 'react'
import { Popover } from 'radix-ui'
import { IconBook } from '@tabler/icons-react'

import type { RulesSection } from '@/components/RulesPanel'
import { useLanguage } from '@/i18n/LanguageContext'
import { cn } from '@/lib/utils'

interface SectionHeadingProps {
  title: ReactNode
  /** Text of the title, naming the "?" button for screen readers. */
  topic: string
  /** One line under the title, 80 characters at most. */
  summary?: string
  /** Full explanation, shown by the "?" button. */
  help: string
  /** Rules section the popover links to. */
  rulesSection?: RulesSection
  onOpenRules?: (section: RulesSection) => void
  /** Heading level: h3 for a season's panel, h4 for a section inside it. */
  level?: 'h3' | 'h4'
  /** Winter panels write their texts in the winter ink. */
  tone?: 'ink' | 'winter'
  titleClassName?: string
}

/**
 * Section title of the command post with its help (charte-graphique.md
 * § Textes d'aide): a one-line summary under the title, and a "?" button
 * opening the full text with a link to the rule.
 */
export function SectionHeading({
  title,
  topic,
  summary,
  help,
  rulesSection,
  onOpenRules,
  level = 'h4',
  tone = 'ink',
  titleClassName,
}: SectionHeadingProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  const Heading = level

  return (
    <div>
      <div className="flex items-start justify-between gap-2">
        <Heading className={titleClassName}>{title}</Heading>
        <Popover.Root open={open} onOpenChange={setOpen}>
          <Popover.Trigger asChild>
            <button
              type="button"
              aria-label={t('help.about', { topic })}
              title={t('help.about', { topic })}
              // A 24 px circle inside a 44 px touch target on phones.
              className="-my-2.5 -mr-2.5 flex size-11 shrink-0 items-center justify-center lg:m-0 lg:size-6"
            >
              <span
                aria-hidden="true"
                className="flex size-6 items-center justify-center rounded-full border border-line-strong bg-paper-50 text-xs font-semibold text-ink-700"
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
      </div>
      {summary && (
        <p
          className={cn(
            'mt-0.5 text-xs',
            tone === 'winter' ? 'text-winter-700' : 'text-ink-500',
          )}
        >
          {summary}
        </p>
      )}
    </div>
  )
}
