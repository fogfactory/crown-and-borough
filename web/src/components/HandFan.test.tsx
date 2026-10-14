import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { HandFan, type FanCard } from '@/components/HandFan'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { stubPhoneViewport } from '@/test/viewport'

const cards: FanCard[] = [
  {
    key: 'special:fair_weather',
    family: 'special',
    name: 'Beau temps',
    code: 'BT',
    icon: null,
    copies: 2,
    actions: <button type="button">Jouer le beau temps</button>,
  },
  {
    key: 'noble:c1',
    family: 'noble',
    name: 'Albert',
    code: 'ALB',
    icon: null,
    discarded: true,
    actions: <button type="button">Jouer Albert</button>,
  },
  {
    key: 'noble:c2',
    family: 'dignity',
    name: 'Bâtard',
    code: 'BAS',
    icon: null,
    actions: <button type="button">Jouer le bâtard</button>,
  },
]

const renderFan = () =>
  render(
    <LanguageProvider initialLanguage="fr">
      <HandFan cards={cards} label="Cartes en main" footer={<p>Deck : 3 · Main : 4/6</p>} />
    </LanguageProvider>,
  )

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('HandFan', () => {
  it('tells the families apart by their rule, not by a colour', () => {
    const { container } = renderFan()

    const face = (code: string) => container.querySelector(`[data-hand-card="${code}"]`)
    expect(face('BT')).toHaveClass('border-brass-700')
    expect(face('ALB')).toHaveClass('border-ink-900')
    expect(face('BAS')).toHaveClass('border-double')
    expect(screen.getByRole('button', { name: 'Beau temps (BT) ×2' })).toBeInTheDocument()
    expect(face('ALB')).toHaveAttribute('data-discarded', 'true')
    expect(screen.getByText('Deck : 3 · Main : 4/6')).toBeInTheDocument()
  })

  it('shows the controls of the picked card only', () => {
    renderFan()

    expect(screen.queryByRole('button', { name: /^Jouer/ })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Bâtard (BAS)' }))
    expect(screen.getByRole('button', { name: 'Bâtard (BAS)' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: 'Jouer le bâtard' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Jouer Albert' })).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Bâtard (BAS)' }))
    expect(screen.queryByRole('button', { name: 'Jouer le bâtard' })).toBeNull()
  })

  it('fans the cards on desktop and lines them up on phones', () => {
    const { container, unmount } = renderFan()
    expect(container.querySelector('li')?.getAttribute('style')).toContain('rotate(-3deg)')
    unmount()

    stubPhoneViewport()
    const phone = renderFan()
    expect(phone.container.querySelector('li')?.getAttribute('style')).toBeNull()
    expect(phone.container.querySelector('[data-hand-card]')).toHaveClass('h-[88px]', 'w-16')
  })
})
