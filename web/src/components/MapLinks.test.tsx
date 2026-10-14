import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import {
  MapLinksProvider,
  NobleRef,
  TerritoryRef,
  type MapLinks,
} from '@/components/MapLinks'
import { LanguageProvider } from '@/i18n/LanguageContext'

function links(overrides: Partial<MapLinks> = {}): MapLinks {
  return {
    focus: vi.fn(),
    highlight: vi.fn(),
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
