import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { Terrain } from '@/types'

export const TERRAIN_LABELS: Record<Terrain, string> = {
  plain: 'Plaine',
  forest: 'Forêt',
  hill: 'Colline',
  mountain: 'Montagne',
  swamp: 'Marécage',
}

export const TERRAIN_COLORS: Record<Terrain, string> = {
  plain: '#b8d99a',
  forest: '#3f7854',
  hill: '#ad8565',
  mountain: '#89929a',
  swamp: '#66a6a0',
}

const TERRAIN_ORDER: Terrain[] = ['plain', 'forest', 'hill', 'mountain', 'swamp']

export function MapLegend() {
  return (
    <Card
      aria-label="Légende de la carte"
      className="w-full border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]"
    >
      <CardHeader className="gap-0 pb-2">
        <CardTitle className="text-sm uppercase tracking-[0.16em] text-[#594b3c]">
          Légende
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-xs text-[#594b3c]">
        <div className="grid grid-cols-2 gap-x-3 gap-y-1.5">
          {TERRAIN_ORDER.map((terrain) => (
            <div key={terrain} className="flex items-center gap-2">
              <span
                className="size-3 shrink-0 rounded-full border border-[#594b3c]/30"
                style={{ backgroundColor: TERRAIN_COLORS[terrain] }}
              />
              <span>{TERRAIN_LABELS[terrain]}</span>
            </div>
          ))}
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M-9 8V-1L0-10L9-1V8Z"
                fill="#fff8e7"
                stroke="#6b4c28"
                strokeWidth="1.5"
              />
              <rect x="-4" y="1" width="8" height="7" fill="#b7834e" />
            </svg>
            <span>Village</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-11 -11 22 22" aria-hidden="true">
              <path
                d="M-9 9V-3H-5V-9H-1V-3H3V-9H7V-3H10V9Z"
                fill="#efe6d0"
                stroke="#5f4936"
                strokeWidth="1.5"
              />
            </svg>
            <span>Château</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="size-3 shrink-0 rounded-full border-2 border-[#fff8e7] bg-[#a84632]" />
            <span>Armée (pastille numérotée)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M0-8L8 0L0 8L-8 0Z"
                fill="#a84632"
                stroke="#815f1e"
                strokeWidth="1.5"
              />
              <circle cx="0" cy="0" r="2" fill="#fff3c4" />
            </svg>
            <span>Noble (couleur du propriétaire)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="-10 -10 20 20" aria-hidden="true">
              <path
                d="M0-8L8 0L0 8L-8 0Z"
                fill="#a84632"
                stroke="#8d321e"
                strokeWidth="1.5"
              />
              <circle cx="0" cy="0" r="4.5" fill="none" stroke="#8d321e" strokeWidth="1.5" />
            </svg>
            <span>Noble prisonnier (otage / donjon)</span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="size-3 shrink-0" viewBox="0 0 16 16" aria-hidden="true">
              <rect
                x="3"
                y="3"
                width="10"
                height="10"
                fill="none"
                stroke="#a84632"
                strokeWidth="3"
              />
            </svg>
            <span>Liséré coloré = contrôle territorial</span>
          </div>
        </div>
        <p className="border-t border-[#b7a786]/60 pt-2 leading-relaxed">
          Trait continu épais = frontière infranchissable · Trait pointillé = frontière
          franchissable
        </p>
      </CardContent>
    </Card>
  )
}
