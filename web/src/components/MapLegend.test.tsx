import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { MapLegend } from '@/components/MapLegend'
import { LanguageProvider } from '@/i18n/LanguageContext'

describe('MapLegend', () => {
  it('renders the map legend as a standalone panel', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.getByText('Légende')).toBeInTheDocument()
    expect(screen.getByText('Plaine')).toBeInTheDocument()
    expect(
      screen.getByText(/Trait continu épais = frontière infranchissable/),
    ).toBeInTheDocument()
    expect(
      container.querySelector('svg[aria-label="Carte des territoires"]'),
    ).not.toBeInTheDocument()
  })

  it('documents noble affiliation and prisoner marks', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.getByText('Noble (couleur du propriétaire)')).toBeInTheDocument()
    expect(screen.getByText('Noble prisonnier (otage / donjon)')).toBeInTheDocument()
  })
})
