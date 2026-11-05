import type { MessageKey } from '@/i18n/messages'
import type { Season } from '@/types'

export const SEASON_LABEL_KEYS: Record<Season, MessageKey> = {
  spring: 'season.spring',
  summer: 'season.summer',
  autumn: 'season.autumn',
  winter: 'season.winter',
}

/** Interface tint of each season (charte-graphique.md § Saisons). */
export const SEASON_TINT_BG: Record<Season, string> = {
  spring: 'bg-season-spring',
  summer: 'bg-season-summer',
  autumn: 'bg-season-autumn',
  winter: 'bg-season-winter',
}
