import type { ComponentType } from 'react'
import {
  IconAward,
  IconCoins,
  IconScale,
  IconScript,
  IconShovel,
  IconSword,
  IconTarget,
  IconUser,
} from '@tabler/icons-react'

import { GAME_ICON_GLYPHS } from '@/lib/game-icon-glyphs'
import type { CardKind, NobleCardKind } from '@/types'

// Calamity and bonus cards reuse the map's game-icons.net artwork (CC BY 3.0,
// icons by Delapouite); the other cards take a Tabler icon.
const GLYPH_NAMES: Partial<Record<CardKind, string>> = {
  fair_weather: 'sun',
  abundant_harvest: 'wheat',
  revolt: 'uprising',
  plague: 'reaper-scythe',
  bad_weather: 'raining',
  famine: 'desert-skull',
}

type Icon = ComponentType<{ className?: string; 'aria-hidden'?: boolean }>

const ICONS: Partial<Record<CardKind | NobleCardKind, Icon>> = {
  seigneurial_tax: IconCoins,
  tunnel: IconShovel,
  ambush: IconTarget,
  assassination: IconSword,
  justice: IconScale,
  noble: IconUser,
  claim: IconScript,
  dignity: IconAward,
}

/** Ink pictogram of a card of the hand. */
export function CardPictogram({ kind }: { kind: CardKind | NobleCardKind }) {
  const glyphName = GLYPH_NAMES[kind as CardKind]
  const glyph = glyphName ? GAME_ICON_GLYPHS[glyphName] : undefined
  if (glyph) {
    return (
      <svg viewBox="0 0 512 512" className="size-6" aria-hidden="true">
        <g transform={glyph.transform}>
          <path d={glyph.path} fill="currentColor" />
        </g>
      </svg>
    )
  }
  const Icon = ICONS[kind] ?? IconScript
  return <Icon aria-hidden={true} className="size-5" />
}
