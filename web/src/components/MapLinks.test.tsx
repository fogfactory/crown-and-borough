import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import {
  MapLinksProvider,
  MapMention,
  NobleRef,
  RegionLabel,
  RegionRef,
  TerritoryRef,
  type MapLinks,
} from '@/components/MapLinks'
import { LanguageProvider } from '@/i18n/LanguageContext'

function links(overrides: Partial<MapLinks> = {}): MapLinks {
  return {
    focus: vi.fn(),
    follow: vi.fn(),
    showEvent: vi.fn(),
    highlight: vi.fn(),
    regionTerritories: (seed: string) => [seed],
    hasTerritory: (id) => ['RVE', 'RVN'].includes(id),
    territoryName: (id) => ({ RVE: 'Rochevent', RVN: 'Ravinval' })[id],
    nobleName: (code) => ({ RAI: 'Rainaud de Rochevent', DED: 'Gautier' })[code],
    nobleTerritory: (code) => (code === 'RAI' ? 'RVE' : undefined),
    isDeceased: (code) => code === 'DED',
    ...overrides,
  }
}

function renderWith(value: MapLinks, node: React.ReactNode) {
  return render(
    <LanguageProvider initialLanguage="fr">
      <MapLinksProvider value={value}>{node}</MapLinksProvider>
    </LanguageProvider>,
  )
}

describe('RegionRef', () => {
  it('lights the whole region on hover and follows to its seed', () => {
    const value = links({ regionTerritories: () => ['RVE', 'RVN', 'CHA'] })
    renderWith(value, <RegionRef id="RVE" />)

    const link = screen.getByRole('button', { name: 'Rochevent (RVE)' })
    fireEvent.mouseEnter(link)
    expect(value.highlight).toHaveBeenLastCalledWith(['RVE', 'RVN', 'CHA'])
    fireEvent.click(link)
    expect(value.focus).toHaveBeenCalledWith('RVE', undefined)
  })
})

describe('RegionLabel', () => {
  it('lights the region on hover and focus, and frames the seed on click', () => {
    const value = links({ regionTerritories: () => ['RVE', 'RVN'] })
    renderWith(value, <RegionLabel seed="RVE">Alpilles</RegionLabel>)

    const link = screen.getByRole('button', { name: 'Alpilles' })
    fireEvent.mouseEnter(link)
    expect(value.highlight).toHaveBeenLastCalledWith(['RVE', 'RVN'])
    fireEvent.mouseLeave(link)
    expect(value.highlight).toHaveBeenLastCalledWith(null)
    fireEvent.focus(link)
    expect(value.highlight).toHaveBeenLastCalledWith(['RVE', 'RVN'])
    fireEvent.blur(link)
    fireEvent.click(link)
    expect(value.focus).toHaveBeenCalledWith('RVE', undefined)
  })

  it('stays plain text when the seed is not on the map', () => {
    renderWith(links(), <RegionLabel seed="ZZZ">Alpilles</RegionLabel>)
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.getByText('Alpilles')).toBeInTheDocument()
  })
})

describe('TerritoryRef', () => {
  it('writes the name then the code, and links to the map', () => {
    const value = links()
    renderWith(value, <TerritoryRef id="RVE" />)

    const link = screen.getByRole('button', { name: 'Rochevent (RVE)' })
    expect(link.querySelector('.font-mono')).toHaveTextContent('(RVE)')

    fireEvent.mouseEnter(link)
    expect(value.highlight).toHaveBeenLastCalledWith('RVE')
    fireEvent.mouseLeave(link)
    expect(value.highlight).toHaveBeenLastCalledWith(null)

    fireEvent.click(link)
    expect(value.focus).toHaveBeenCalledWith('RVE', undefined)
  })

  it('stays plain text outside a page that offers links', () => {
    render(<TerritoryRef id="RVE" name="Rochevent" />)

    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.getByText('Rochevent')).toBeInTheDocument()
  })
})

describe('NobleRef', () => {
  it('links a noble to the territory where it stands, with its halo', () => {
    const value = links()
    renderWith(value, <NobleRef code="RAI" />)

    fireEvent.click(screen.getByRole('button', { name: 'Rainaud de Rochevent (RAI)' }))
    expect(value.focus).toHaveBeenCalledWith('RVE', 'RAI')
  })

  it('writes a dead noble or one of unknown position without a link', () => {
    renderWith(
      links(),
      <>
        <NobleRef code="DED" />
        <NobleRef code="ZZZ" name="Inconnu" />
      </>,
    )

    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.getByText('Gautier').closest('[title]')).toHaveAttribute(
      'title',
      'décédé',
    )
    expect(screen.getByText('Inconnu').closest('[title]')).toHaveAttribute(
      'title',
      'position inconnue',
    )
  })
})

describe('MapMention', () => {
  it('keeps the written name and follows a commune', () => {
    const value = links()
    renderWith(
      value,
      <MapMention territory="RVE" nobles={[]}>
        Rochevent
      </MapMention>,
    )
    const link = screen.getByRole('button', { name: 'Rochevent' })
    fireEvent.mouseEnter(link)
    expect(value.highlight).toHaveBeenLastCalledWith('RVE')
    fireEvent.click(link)
    expect(value.focus).toHaveBeenCalledWith('RVE', undefined)
  })

  it('follows a noble to where it stands, with its halo', () => {
    const value = links()
    renderWith(value, <MapMention nobles={['RAI']}>Rainaud</MapMention>)
    fireEvent.click(screen.getByRole('button', { name: 'Rainaud' }))
    expect(value.focus).toHaveBeenCalledWith('RVE', 'RAI')
  })

  it('frames every bearer of a shared name', () => {
    const value = links({
      nobleTerritory: (code) => ({ RAI: 'RVE', RA2: 'RVN' })[code],
    })
    renderWith(value, <MapMention nobles={['RAI', 'RA2', 'ZZZ']}>Rainaud</MapMention>)
    const link = screen.getByRole('button', { name: 'Rainaud' })
    fireEvent.mouseEnter(link)
    expect(value.highlight).toHaveBeenLastCalledWith(['RVE', 'RVN'])
    fireEvent.click(link)
    expect(value.showEvent).toHaveBeenCalledWith({ territories: ['RVE', 'RVN'] })
    expect(value.focus).not.toHaveBeenCalled()
  })

  it('follows a commune without halo when its namesake stands there', () => {
    const value = links()
    renderWith(
      value,
      <MapMention territory="RVE" nobles={['RAI']}>
        Rochevent
      </MapMention>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Rochevent' }))
    expect(value.focus).toHaveBeenCalledWith('RVE', undefined)
    expect(value.showEvent).not.toHaveBeenCalled()
  })

  it('stays text without a known place', () => {
    renderWith(links(), <MapMention nobles={['ZZZ']}>Inconnu</MapMention>)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.getByText('Inconnu')).toBeInTheDocument()
  })
})
