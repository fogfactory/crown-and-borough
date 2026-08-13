import type { NobleStatus } from '@/types'
import { translate, type Language, type MessageKey } from '@/i18n/messages'

export const NOBLE_STATUS_LABEL_KEYS: Record<NobleStatus, MessageKey> = {
  free: 'orders.nobleStatus.free',
  hostage: 'orders.nobleStatus.hostage',
  dungeon: 'orders.nobleStatus.dungeon',
}

export function nobleStatusLabel(language: Language, status: NobleStatus): string {
  return translate(language, NOBLE_STATUS_LABEL_KEYS[status])
}
