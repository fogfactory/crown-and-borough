import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { SectionHeading } from '@/components/SectionHeading'
import { LanguageProvider } from '@/i18n/LanguageContext'

describe('SectionHeading', () => {
  it('shows one summary line and the full help behind the "?" button', () => {
    const onOpenRules = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <SectionHeading
          title="Chaînes d'ordres"
          topic="Chaînes d'ordres"
          summary="Une chaîne par noble."
          help="Le texte complet de l'aide."
          rulesSection="action-orders"
          onOpenRules={onOpenRules}
        />
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: "Chaînes d'ordres" })).toBeInTheDocument()
    expect(screen.getByText('Une chaîne par noble.')).toBeInTheDocument()
    expect(screen.queryByText("Le texte complet de l'aide.")).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: "Aide : Chaînes d'ordres" }))
    expect(screen.getByText("Le texte complet de l'aide.")).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Voir la règle' }))
    expect(onOpenRules).toHaveBeenCalledWith('action-orders')
    expect(screen.queryByText("Le texte complet de l'aide.")).toBeNull()
  })
})
