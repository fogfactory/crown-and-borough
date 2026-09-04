import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FaqPanel } from '@/components/FaqPanel'
import { LanguageProvider } from '@/i18n/LanguageContext'

describe('FaqPanel', () => {
  it('renders the complete French tactical FAQ', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <FaqPanel />
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: 'FAQ tactique' })).toBeInTheDocument()
    expect(
      screen.getByText('Pourquoi joindre avant d’attaquer ?', { exact: true }),
    ).toBeInTheDocument()
    expect(screen.getByText(/Quel est l’effet des moulins/)).toBeInTheDocument()
    expect(
      screen.getByText(/Une dispersion est un partage pacifique/),
    ).toBeInTheDocument()
    expect(container.querySelectorAll('details')).toHaveLength(9)
  })

  it('renders the English tactical FAQ', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <FaqPanel />
      </LanguageProvider>,
    )

    expect(screen.getByRole('heading', { name: 'Tactical FAQ' })).toBeInTheDocument()
    expect(
      screen.getByText('Why join before attacking?', { exact: true }),
    ).toBeInTheDocument()
    expect(screen.getByText(/How do mills affect production/)).toBeInTheDocument()
    expect(
      screen.getByText(/A dispersal is peaceful strength-0 splitting/),
    ).toBeInTheDocument()
    expect(container.querySelectorAll('details')).toHaveLength(9)
  })
})
