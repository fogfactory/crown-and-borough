import type { ReactNode } from 'react'
import { Popover } from 'radix-ui'

import { StaticCard, type FanCard } from '@/components/HandFan'
import type { RulesSection } from '@/components/RulesPanel'
import { useLanguage } from '@/i18n/LanguageContext'

/** A card of a deck, drawn like the cards of the hand. */
export type DeckCard = Omit<FanCard, 'actions'>

/**
 * A deck counter that opens, on click or touch, the whole content of the deck
 * as small cards with their "?" badge.
 */
export function DeckContents({
  deckName,
  counter,
  title,
  cards,
  onOpenRules,
}: {
  /** Name of the deck, for the accessible name of the counter. */
  deckName: string
  /** Text of the counter (e.g. "Nobles : 5"). */
  counter: ReactNode
  title: string
  cards: DeckCard[]
  onOpenRules?: (section: RulesSection) => void
}) {
  const { t } = useLanguage()
  const total = cards.reduce((sum, card) => sum + (card.copies ?? 1), 0)

  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={t('deck.show', { deck: deckName })}
          title={t('deck.show', { deck: deckName })}
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
          className="z-50 max-h-[min(70dvh,32rem)] w-[22rem] max-w-[calc(100vw-1.5rem)] space-y-2 overflow-y-auto rounded-md border border-line bg-paper-50 p-3 text-sm text-ink-700 shadow-panel"
        >
          <p className="font-display text-base font-semibold text-ink-700">{title}</p>
          <p className="text-xs text-ink-500">{t('deck.content', { count: total })}</p>
          <ul className="flex flex-wrap gap-2.5 pt-1">
            {cards.map((card) => (
              <li key={card.key}>
                <StaticCard card={card} onOpenRules={onOpenRules} />
              </li>
            ))}
          </ul>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
