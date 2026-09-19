import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

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

  it('draws terrain swatches with the map texture patterns', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    for (const terrain of ['plain', 'forest', 'hill', 'mountain', 'swamp']) {
      const pattern = container.querySelector(`#legend-terrain-${terrain}`)
      expect(pattern).toBeInTheDocument()
      expect(pattern?.querySelectorAll('path, circle, line').length).toBeGreaterThan(0)
    }
  })

  it('shows settlements with the owner-colored glyph style', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    const filledGlyphs = container.querySelectorAll('svg path[fill="#a84632"]')
    expect(filledGlyphs.length).toBeGreaterThanOrEqual(2)
  })

  it('shows the intentions toggle only when a handler is provided', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.queryByText("Calque d'intentions")).not.toBeInTheDocument()
  })

  it('toggles the intentions overlay through the legend checkbox', () => {
    const onToggleIntentions = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend showIntentions={true} onToggleIntentions={onToggleIntentions} />
      </LanguageProvider>,
    )

    const checkbox = screen.getByRole('checkbox')
    expect(checkbox).toBeChecked()

    fireEvent.click(checkbox)

    expect(onToggleIntentions).toHaveBeenCalledWith(false)
  })
})
