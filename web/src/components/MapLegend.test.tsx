import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MapLegend, OWNERSHIP_SHIELD_PATH } from '@/components/MapLegend'
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
    expect(screen.getByText('Frontière franchissable')).toBeInTheDocument()
    expect(screen.getByText('Escarpement (infranchissable)')).toBeInTheDocument()
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
  it('samples the engraved terrain symbols and the border series', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    for (const terrain of ['plain', 'forest', 'hill', 'mountain', 'swamp']) {
      const swatch = container.querySelector(`[data-legend-terrain="${terrain}"]`)
      expect(swatch?.querySelector('rect')).toHaveAttribute('fill', '#f1e6cb')
      expect(swatch?.querySelector('path[stroke="#30291f"]')).toBeInTheDocument()
    }
    for (const kind of ['passable', 'peaks', 'marsh', 'cliff']) {
      expect(
        container.querySelector(`[data-legend-border="${kind}"]`),
      ).toBeInTheDocument()
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

  it('toggles the shared regions layer', () => {
    const onToggle = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend showRegions={false} onToggleRegions={onToggle} />
      </LanguageProvider>,
    )
    fireEvent.click(screen.getByLabelText('Régions'))
    expect(onToggle).toHaveBeenCalledWith(true)
    expect(document.querySelectorAll('[data-region-color]').length).toBe(6)
    expect(document.querySelectorAll('[data-region-pattern]').length).toBe(0)
  })

  it('toggles the player control layer', () => {
    const onToggle = vi.fn()
    render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend showOwnership={false} onToggleOwnership={onToggle} />
      </LanguageProvider>,
    )
    fireEvent.click(screen.getByLabelText('Contrôle joueur'))
    expect(onToggle).toHaveBeenCalledWith(true)
  })

  it('describes territorial control with the shield vignette', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.getByText('Colored shield = territorial control')).toBeInTheDocument()
    const vignette = container.querySelector('svg[viewBox="-13 -15 26 30"]')
    expect(vignette?.querySelector(`path[d="${OWNERSHIP_SHIELD_PATH}"]`)).not.toBeNull()
  })

  it('shows the occupied fief member hatch swatch', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(
      screen.getByText('Occupied (controlled, held by another army)'),
    ).toBeInTheDocument()
    expect(container.querySelector('#legend-occupied-hatch')).toBeInTheDocument()
  })
})
