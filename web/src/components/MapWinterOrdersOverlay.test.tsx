import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { WinterOrdersOverlay } from '@/components/MapWinterOrdersOverlay'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { WinterIntention } from '@/lib/winter-overlay'
import type { Territory } from '@/types'

function squareTerritory(id: string, x: number): Territory {
  return {
    id,
    name: id,
    terrain: 'plain',
    village: false,
    points: [
      [x, 0],
      [x + 50, 0],
      [x + 50, 50],
      [x, 50],
    ],
    adjacencies: [],
    impassable: [],
  }
}

const territories: Territory[] = [
  squareTerritory('AAA', 0),
  squareTerritory('BBB', 50),
  squareTerritory('CCC', 100),
]

function renderOverlay(winterIntentions: WinterIntention[]) {
  return render(
    <LanguageProvider>
      <svg>
        <WinterOrdersOverlay
          territories={territories}
          winterIntentions={winterIntentions}
          annotationScale={1}
        />
      </svg>
    </LanguageProvider>,
  )
}

describe('WinterOrdersOverlay found_fief group', () => {
  const foundFief: WinterIntention = {
    kind: 'fief_found',
    line: 1,
    valid: true,
    source: 'draft',
    color: '#a84632',
    territory: 'AAA',
    territories: ['AAA', 'BBB', 'CCC'],
    label: 'T F HUG AAA BBB CCC',
  }

  it('plants a faded banner, marked with the capital, on each non-capital member', () => {
    const { container } = renderOverlay([{ ...foundFief, player: 'P1' }])

    const members = container.querySelectorAll('[data-fief-group-member]')
    expect(
      Array.from(members).map((member) => member.getAttribute('data-fief-group-member')),
    ).toEqual(['BBB', 'CCC'])
    for (const member of members) {
      expect(member.querySelector('use')).toHaveAttribute('href', '#arms-banner-P1')
      expect(member.querySelector('[data-fief-attachment]')?.textContent).toBe('AAA')
      // Lighter than a settled banner since this is only a draft.
      expect(member.getAttribute('opacity')).toBe('0.6')
    }
  })

  it('draws one connecting line per member from the capital, colored per player', () => {
    const { container } = renderOverlay([foundFief])

    const lines = container.querySelectorAll('line[data-fief-group-line]')
    expect(lines).toHaveLength(2)
    for (const line of lines) {
      expect(line.getAttribute('stroke')).toBe('#a84632')
    }
  })
})
