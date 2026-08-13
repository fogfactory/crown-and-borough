import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { LanguageProvider } from '@/i18n/LanguageContext'

afterEach(() => {
  window.localStorage.clear()
})

describe('LanguageSwitcher', () => {
  it('defaults to English, switches to French, and persists the choice', async () => {
    render(
      <LanguageProvider>
        <LanguageSwitcher />
      </LanguageProvider>,
    )

    expect(screen.getByRole('button', { name: 'en' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    fireEvent.click(screen.getByRole('button', { name: 'fr' }))

    expect(screen.getByRole('button', { name: 'fr' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await waitFor(() => {
      expect(window.localStorage.getItem('crown-and-borough.language')).toBe('fr')
    })
  })

  it('restores the persisted language on the next provider', () => {
    window.localStorage.setItem('crown-and-borough.language', 'fr')

    render(
      <LanguageProvider>
        <LanguageSwitcher />
      </LanguageProvider>,
    )

    expect(screen.getByRole('button', { name: 'fr' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })
})
