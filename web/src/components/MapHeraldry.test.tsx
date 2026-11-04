import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ArmyGlyph, ControlShield } from '@/components/MapHeraldry'
import { BANNER_HEIGHT, FIGURE_HEAD_TOP, SPEAR_WIDTH, SPEAR_X } from '@/lib/heraldry'

function renderArmy(size: number, compact = false) {
  const { container } = render(
    <svg>
      <ArmyGlyph owner="P1" size={size} seed="ROS" compact={compact} />
    </svg>,
  )
  return container
}

const numberAttr = (element: Element | null, name: string) =>
  Number(element?.getAttribute(name))

describe('ArmyGlyph', () => {
  it('draws one soldier per troop up to six, and the strength on the banner', () => {
    const army = renderArmy(4)
    expect(army.querySelectorAll('[data-army-soldier]')).toHaveLength(4)
    expect(army.querySelectorAll('[data-army-shadow]')).toHaveLength(0)
    expect(army.querySelector('text')).toHaveTextContent('4')
    const big = renderArmy(25)
    expect(big.querySelectorAll('[data-army-soldier]')).toHaveLength(6)
    expect(big.querySelectorAll('[data-army-shadow]')).toHaveLength(4)
    expect(big.querySelector('text')).toHaveTextContent('25')
  })

  it("plants the banner pole on the leftmost soldier's spear", () => {
    const army = renderArmy(6)
    const soldierXs = [...army.querySelectorAll('[data-army-soldier]')].map((soldier) =>
      numberAttr(soldier, 'x'),
    )
    const pole = army.querySelector('line[stroke="#30291f"]')
    expect(numberAttr(pole, 'x1')).toBeCloseTo(Math.min(...soldierXs) + SPEAR_X)
    expect(numberAttr(pole, 'x2')).toBe(numberAttr(pole, 'x1'))
  })

  it('continues the spear with a pole of the same thickness, to the banner', () => {
    const army = renderArmy(6)
    const pole = army.querySelector('[data-army-pole]')
    const flag = army.querySelector('use[href^="#arms-banner"]')
    expect(numberAttr(pole, 'stroke-width')).toBe(SPEAR_WIDTH)
    // The banner hangs from the top of the pole, against it.
    expect(numberAttr(flag, 'x')).toBe(numberAttr(pole, 'x1'))
    expect(numberAttr(flag, 'y')).toBe(numberAttr(pole, 'y2'))
  })

  it('floats the banner just above its bearer, in front of the pack', () => {
    const army = renderArmy(12)
    const soldiers = [...army.querySelectorAll('[data-army-soldier]')]
    const pole = army.querySelector('[data-army-pole]')
    const flag = army.querySelector('use[href^="#arms-banner"]')
    const bearer = soldiers.find(
      (soldier) => numberAttr(soldier, 'x') + SPEAR_X === numberAttr(pole, 'x1'),
    )
    const headTop = numberAttr(bearer ?? null, 'y') + FIGURE_HEAD_TOP
    const gap = headTop - (numberAttr(flag, 'y') + BANNER_HEIGHT)
    expect(gap).toBeGreaterThan(0)
    expect(gap).toBeLessThan(4)
    // Drawn after every soldier and shadow: it stays visible in a big army.
    const drawn = [...army.querySelectorAll('use')]
    expect(drawn.indexOf(flag as SVGUseElement)).toBe(drawn.length - 1)
  })

  it('keeps the standard-bearer alone when compact', () => {
    const army = renderArmy(40, true)
    expect(army.querySelectorAll('[data-army-soldier]')).toHaveLength(1)
    expect(army.querySelectorAll('[data-army-shadow]')).toHaveLength(0)
    expect(army.querySelector('text')).toHaveTextContent('40')
  })
})

describe('ControlShield', () => {
  it('greys, crowns and writes the fief trigram when asked', () => {
    const { container } = render(
      <svg>
        <ControlShield owner="P1" grey crown="barony" attachedTo="ROS" />
      </svg>,
    )
    expect(container.querySelector('use')).toHaveAttribute('href', '#arms-shield-grey-P1')
    expect(container.querySelector('[data-control-grey]')).toBeInTheDocument()
    expect(container.querySelector('[data-crown="barony"]')).toBeInTheDocument()
    expect(container.querySelector('[data-fief-attachment="ROS"]')).toHaveTextContent(
      'ROS',
    )
  })

  it('is a plain colour shield by default', () => {
    const { container } = render(
      <svg>
        <ControlShield owner="P2" />
      </svg>,
    )
    expect(container.querySelector('use')).toHaveAttribute('href', '#arms-shield-P2')
    expect(container.querySelector('[data-control-grey]')).toBeNull()
    expect(container.querySelector('[data-crown]')).toBeNull()
  })
})
