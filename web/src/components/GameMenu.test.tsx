import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { GameMenu } from '@/components/GameMenu'
import { LanguageProvider } from '@/i18n/LanguageContext'

const renderMenu = (props: Parameters<typeof GameMenu>[0]) =>
  render(
    <LanguageProvider initialLanguage="fr">
      <GameMenu {...props} />
    </LanguageProvider>,
  )

describe('GameMenu', () => {
  it('groups the forced resolution and the language under "Partie"', () => {
    const onResolve = vi.fn()
    renderMenu({ language: true, resolution: { disabled: false, onResolve } })

    fireEvent.click(screen.getByRole('button', { name: 'Partie' }))
    expect(screen.getByRole('group', { name: 'Langue' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Résoudre' }))
    expect(onResolve).toHaveBeenCalledOnce()
  })

  it('asks for a confirmation before an online forced resolution', () => {
    const onResolve = vi.fn()
    renderMenu({ resolution: { disabled: false, confirm: true, onResolve } })

    fireEvent.click(screen.getByRole('button', { name: 'Partie' }))
    fireEvent.click(screen.getByRole('button', { name: 'Résoudre' }))
    expect(onResolve).not.toHaveBeenCalled()
    expect(screen.getByText(/avant que tout le monde ait soumis/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(screen.queryByText(/avant que tout le monde ait soumis/)).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Résoudre' }))
    fireEvent.click(screen.getByRole('button', { name: 'Résoudre maintenant' }))
    expect(onResolve).toHaveBeenCalledOnce()
  })

  it('creates a game with the chosen seed', () => {
    const onCreate = vi.fn()
    const onSeedChange = vi.fn()
    renderMenu({
      creation: {
        playerCount: 4,
        years: 10,
        seed: 'graine',
        creating: false,
        error: null,
        canCreate: true,
        onPlayerCountChange: vi.fn(),
        onYearsChange: vi.fn(),
        onSeedChange,
        onCreate,
      },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Partie' }))
    fireEvent.change(screen.getByLabelText('Graine'), { target: { value: 'autre' } })
    expect(onSeedChange).toHaveBeenCalledWith('autre')
    fireEvent.click(screen.getByRole('button', { name: 'Nouvelle partie' }))
    expect(onCreate).toHaveBeenCalledOnce()
  })
})
