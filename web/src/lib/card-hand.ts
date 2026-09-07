import type { Translate, MessageKey } from '@/i18n/messages'
import type { CardKind } from '@/types'

const CARD_KIND_ORDER: CardKind[] = [
  'fair_weather',
  'abundant_harvest',
  'plague',
  'bad_weather',
  'revolt',
  'famine',
]

const CARD_SHORT_KEYS: Record<CardKind, MessageKey> = {
  fair_weather: 'card.short.fair_weather',
  abundant_harvest: 'card.short.abundant_harvest',
  revolt: 'card.short.revolt',
  plague: 'card.short.plague',
  bad_weather: 'card.short.bad_weather',
  famine: 'card.short.famine',
}

const CARD_NAME_KEYS: Record<CardKind, MessageKey> = {
  fair_weather: 'card.fair_weather',
  abundant_harvest: 'card.abundant_harvest',
  revolt: 'card.revolt',
  plague: 'card.plague',
  bad_weather: 'card.bad_weather',
  famine: 'card.famine',
}

export function formatCardLabel(kind: CardKind, t: Translate): string {
  return `${t(CARD_NAME_KEYS[kind])} (${t(CARD_SHORT_KEYS[kind])})`
}

export function formatCardCode(kind: CardKind, t: Translate): string {
  return t(CARD_SHORT_KEYS[kind])
}

export function formatCardHand(hand: readonly CardKind[], t: Translate): string {
  if (hand.length === 0) return t('orders.deckEmpty')

  const counts = new Map<CardKind, number>()
  for (const kind of hand) counts.set(kind, (counts.get(kind) ?? 0) + 1)

  return CARD_KIND_ORDER.filter((kind) => counts.has(kind))
    .map((kind) => {
      const count = counts.get(kind) ?? 0
      return `${formatCardLabel(kind, t)}${count > 1 ? `x${count}` : ''}`
    })
    .join(', ')
}
