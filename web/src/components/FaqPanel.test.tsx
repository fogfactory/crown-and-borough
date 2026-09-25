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
    expect(screen.getByText(/Où va mon armée vaincue/)).toBeInTheDocument()
    expect(screen.getByText(/Comment fonctionne un transfert/)).toBeInTheDocument()
    expect(
      screen.getByText(/Comment les cartes spéciales et les calamités/),
    ).toBeInTheDocument()
    expect(screen.getByText(/Comment se calcule mon revenu territorial/)).toBeInTheDocument()
    expect(screen.getByText(/Pourquoi le risque de famine affiché/)).toBeInTheDocument()
    expect(container.querySelectorAll('details')).toHaveLength(14)
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
    expect(screen.getByText(/Where does my defeated army retreat/)).toBeInTheDocument()
    expect(screen.getByText(/How does a resource transfer/)).toBeInTheDocument()
    expect(
      screen.getByText(/How do special cards and calamities apply/),
    ).toBeInTheDocument()
    expect(screen.getByText(/How is my territory income calculated/)).toBeInTheDocument()
    expect(screen.getByText(/Why is the projected famine risk only an estimate/)).toBeInTheDocument()
    expect(container.querySelectorAll('details')).toHaveLength(14)
  })
})
