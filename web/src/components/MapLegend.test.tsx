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
    expect(screen.getByText('Frontière franchissable')).toBeInTheDocument()
    expect(screen.getByText('Escarpement (infranchissable)')).toBeInTheDocument()
    expect(
      container.querySelector('svg[aria-label="Carte des territoires"]'),
    ).not.toBeInTheDocument()
  })

  it('samples the noble pawns: statuses and title attributes', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="fr">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.getByText('Noble (couleur du propriétaire)')).toBeInTheDocument()
    expect(screen.getByText('Au cachot')).toBeInTheDocument()
    expect(screen.getByText('Titre de courtoisie (conjoint)')).toBeInTheDocument()
    const pawn = (key: string) => container.querySelector(`[data-legend-pawn="${key}"]`)
    expect(pawn('dungeon')?.querySelector('[data-attribute="dungeon"]')).not.toBeNull()
    expect(
      pawn('courtesy')?.querySelector('[data-attribute="courtesy-crown"]'),
    ).not.toBeNull()
    expect(pawn('king')?.querySelector('[data-crown="king"]')).not.toBeNull()
    expect(pawn('pope')?.querySelector('[data-attribute="tiara"]')).not.toBeNull()
    expect(pawn('bishop')?.querySelector('[data-attribute="crosier"]')).not.toBeNull()
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

    const filledGlyphs = container.querySelectorAll('svg path[fill="#b3322b"]')
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

  it('samples the shields, the crowned fief capital and the army', () => {
    const { container } = render(
      <LanguageProvider initialLanguage="en">
        <MapLegend />
      </LanguageProvider>,
    )

    expect(screen.getByText('Shield = controlled territory')).toBeInTheDocument()
    expect(
      screen.getByText('Greyed shield = occupied by another army'),
    ).toBeInTheDocument()
    expect(screen.getByText('Fief capital (crown of its rank)')).toBeInTheDocument()
    expect(screen.getByText('R = resources in stock')).toBeInTheDocument()
    expect(
      container.querySelector('[data-legend-control="barony"] [data-crown="barony"]'),
    ).toBeInTheDocument()
    expect(
      container.querySelector('[data-legend-control="ROS"] [data-fief-attachment="ROS"]'),
    ).toBeInTheDocument()
    // Gules with a bend or: the field and its charge, on the control shield.
    const shield = container.querySelector('[data-legend-control="plain"]')
    expect(shield?.querySelector('path[fill="#b3322b"]')).toBeInTheDocument()
    expect(shield?.querySelector('[data-charge="bend"] path')).toHaveAttribute(
      'stroke',
      '#d8ab35',
    )
    // The occupied sample is the same arms in grey.
    const greyed = container.querySelector('[data-legend-control="occupied"]')
    expect(greyed?.querySelector('path[fill="#b3322b"]')).toBeNull()
    expect(greyed?.querySelector('[data-control-grey]')).toBeInTheDocument()
    // The army sample: a standard-bearer with its strength on the banner.
    const army = container.querySelector('[data-legend-army]')
    expect(army?.querySelectorAll('[data-army-soldier]')).toHaveLength(1)
    expect(army?.querySelector('text')).toHaveTextContent('3')
    expect(container.querySelector('#legend-occupied-hatch')).not.toBeInTheDocument()
  })
})
