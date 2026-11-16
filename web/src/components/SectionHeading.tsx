import type { ReactNode } from 'react'

import { HelpPopover } from '@/components/HelpPopover'
import type { RulesSection } from '@/components/RulesPanel'
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
  const Heading = level

  return (
    <div>
      <div className="flex items-start justify-between gap-2">
        <Heading className={titleClassName}>{title}</Heading>
        <HelpPopover
          topic={topic}
          help={help}
          rulesSection={rulesSection}
          onOpenRules={onOpenRules}
        />
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
