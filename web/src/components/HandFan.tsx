import { useState, type ReactNode } from 'react'

import { CodeTag } from '@/components/MapLinks'
import { NoblePawnIcon } from '@/components/NobleHolder'
import { useLanguage } from '@/i18n/LanguageContext'
import { NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'
import { useIsDesktop } from '@/lib/use-media-query'
import { cn } from '@/lib/utils'

/**
 * Card families, told apart by their rule and corner piece rather than by a
 * bright colour: special cards (brass rule), noble and claim cards (ink rule,
 * pawn in the corner), dignities (double rule).
 */
export type HandCardFamily = 'special' | 'noble' | 'dignity'

export interface FanCard {
  key: string
  family: HandCardFamily
  name: string
  code: string
  /** Pictogram drawn in ink at the top of the card. */
  icon: ReactNode
  /** Copies of a special card held. */
  copies?: number
  /** Discarded by this turn's draft: greyed and struck through. */
  discarded?: boolean
  /** Controls shown once the card is picked: play, discard. */
  actions: ReactNode
}

const FAMILY_CLASSES: Record<HandCardFamily, string> = {
  special: 'border-2 border-brass-700',
  noble: 'border border-ink-900',
  dignity: 'border-[3px] border-double border-ink-700',
}

/** Fan tilt between two neighbouring cards, in degrees (desktop only). */
const FAN_STEP = 3

function HandCardFace({
  card,
  selected,
  onSelect,
}: {
  card: FanCard
  selected: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      aria-label={`${card.name} (${card.code})${card.copies && card.copies > 1 ? ` ×${card.copies}` : ''}`}
      data-hand-card={card.code}
      data-card-family={card.family}
      data-discarded={card.discarded || undefined}
      className={cn(
        'relative flex h-[88px] w-16 flex-col items-center justify-between rounded-sm bg-paper-50 px-1 pt-2 pb-1 text-ink-900 shadow-panel transition hover:-translate-y-1 motion-reduce:transition-none lg:h-20 lg:w-14',
        FAMILY_CLASSES[card.family],
        selected && '-translate-y-2 ring-2 ring-brass-700 ring-offset-1 ring-offset-paper-100',
        card.discarded && 'opacity-50 grayscale',
      )}
      onClick={onSelect}
    >
      {card.family === 'noble' && (
        <span aria-hidden="true" className="absolute top-0.5 left-0.5 text-[0.6rem]">
          <NoblePawnIcon noble={{ status: 'free' }} color={NEUTRAL_ARMY_COLOR} />
        </span>
      )}
      <span aria-hidden="true" className="flex size-6 items-center justify-center text-ink-900">
        {card.icon}
      </span>
      <span
        className={cn(
          'line-clamp-2 text-center text-2xs leading-tight',
          card.discarded && 'line-through',
        )}
      >
        {card.name}
      </span>
      <span className="font-mono text-2xs text-ink-500">{card.code}</span>
      {card.copies && card.copies > 1 ? (
        <span
          aria-hidden="true"
          className="absolute -top-1.5 -right-1.5 rounded-full bg-ink-900 px-1 text-2xs text-paper-50"
        >
          ×{card.copies}
        </span>
      ) : null}
    </button>
  )
}

/**
 * The hand as a fan of small playing cards (charte-graphique.md § Poste de
 * commandement). Picking a card shows its play and discard controls under
 * the fan, which insert the order line in the input; on phones the fan
 * scrolls sideways.
 */
export function HandFan({
  cards,
  label,
  footer,
}: {
  cards: FanCard[]
  /** Accessible name of the card list. */
  label: string
  /** Deck and hand limit, written under the fan. */
  footer?: ReactNode
}) {
  const { t } = useLanguage()
  const desktop = useIsDesktop()
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const selected = cards.find((card) => card.key === selectedKey)
  const middle = (cards.length - 1) / 2

  return (
    <div className="space-y-2">
      {cards.length === 0 ? (
        <p className="text-xs text-ink-500">{t('orders.handEmpty')}</p>
      ) : (
        <ul
          aria-label={label}
          className="flex gap-1.5 overflow-x-auto px-1 pt-3 pb-2 lg:gap-0 lg:overflow-visible lg:px-3"
        >
          {cards.map((card, index) => (
            <li
              key={card.key}
              className="shrink-0 origin-bottom lg:-ml-2 lg:first:ml-0"
              style={
                desktop
                  ? { transform: `rotate(${(index - middle) * FAN_STEP}deg)` }
                  : undefined
              }
            >
              <HandCardFace
                card={card}
                selected={card.key === selectedKey}
                onSelect={() =>
                  setSelectedKey((current) => (current === card.key ? null : card.key))
                }
              />
            </li>
          ))}
        </ul>
      )}
      {selected && (
        <div
          data-hand-actions
          className="flex flex-wrap items-center gap-2 rounded-md border border-line bg-paper-50 p-2 text-sm"
        >
          <span className="font-semibold">
            {selected.name} <CodeTag code={selected.code} />
          </span>
          {selected.actions}
        </div>
      )}
      {footer}
    </div>
  )
}
