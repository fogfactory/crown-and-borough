import { useState } from 'react'
import { Popover } from 'radix-ui'

import { StaticCard, type FanCard } from '@/components/HandFan'
import type { RulesSection } from '@/components/RulesPanel'
import { useLanguage } from '@/i18n/LanguageContext'

/** A card of a deck, drawn like the cards of the hand. */
export type DeckCard = Omit<FanCard, 'actions'>

/**
 * A deck counter that opens, on click or touch, the whole content of the deck
 * as small cards with their "?" badge. The popover closes when one of those
 * badges sends the player to the rules.
 */
export function DeckContents({
  counter,
  title,
  cards,
  onOpenRules,
}: {
  /** Text of the counter (e.g. "Nobles : 5"); the button's name starts with it. */
  counter: string
  /** Name of the deck. */
  title: string
  cards: DeckCard[]
  onOpenRules?: (section: RulesSection) => void
}) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  const total = cards.reduce((sum, card) => sum + (card.copies ?? 1), 0)
  const show = t('deck.show', { deck: title })
  const openRules = onOpenRules
    ? (section: RulesSection) => {
        setOpen(false)
        onOpenRules(section)
      }
    : undefined

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={`${counter}. ${show}`}
          title={show}
          data-deck-counter
          className="rounded-sm underline decoration-dotted underline-offset-2 hover:text-ink-900 hover:decoration-solid focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none max-lg:min-h-11"
        >
          {counter}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="start"
          sideOffset={6}
          aria-label={title}
          className="z-50 max-h-[min(70dvh,32rem)] w-[22rem] max-w-[calc(100vw-1.5rem)] space-y-2 overflow-y-auto rounded-md border border-line bg-paper-50 p-3 text-sm text-ink-700 shadow-panel"
        >
          <p className="font-display text-base font-semibold text-ink-700">{title}</p>
          <p className="text-xs text-ink-500">{t('deck.content', { count: total })}</p>
          <ul className="flex flex-wrap gap-2.5 pt-1">
            {cards.map((card) => (
              <li key={card.key}>
                <StaticCard card={card} onOpenRules={openRules} />
              </li>
            ))}
          </ul>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
