import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MapLinksProvider, type MapLinks } from '@/components/MapLinks'
import {
  OrderFormProvider,
  OrderLauncher,
  fullLabelComment,
} from '@/components/OrderDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'

const links: MapLinks = {
  focus: vi.fn(),
  highlight: vi.fn(),
  hasTerritory: (id) => ['ROS', 'RVN'].includes(id),
  territoryName: (id) => ({ ROS: 'Rosemont', RVN: 'Ravinval' })[id],
  nobleName: () => undefined,
  nobleTerritory: (code) => (code === 'HUG' ? 'RVN' : undefined),
  isDeceased: () => false,
}

// The map click the page would hand to the open form.
const mapPick: { current: ((territory: string) => void) | null } = { current: null }

function Harness({ onConfirm = vi.fn() }: { onConfirm?: (line: string) => void }) {
  return (
    <LanguageProvider initialLanguage="fr">
      <MapLinksProvider value={links}>
        <OrderFormProvider onPickerChange={(next) => (mapPick.current = next)}>
          <div className="flex flex-wrap">
            <OrderLauncher
              label="Recruter"
              title="Recruter (R N)"
              fields={[
                {
                  key: 'territory',
                  label: 'Territoire',
                  options: [
                    { value: 'ROS', label: 'ROS', territory: true },
                    { value: 'RVN', label: 'RVN', territory: true },
                  ],
                },
              ]}
              buildOrder={(values) => ({
                line: `R N ALB ${values.territory}`,
                comment: `recruter Albert sur ${values.territory}`,
              })}
              onConfirm={onConfirm}
            />
            <OrderLauncher
              label="Doter"
              title="Doter (D N)"
              fields={[
                {
                  key: 'noble',
                  label: 'Noble',
                  options: [
                    { value: 'ANN', label: 'Anne (ANN)' },
                    { value: 'HUG', label: 'Hugues (HUG)' },
                  ],
                },
              ]}
              buildOrder={(values) => ({
                line: `D N ${values.noble} BAS`,
                comment: 'Bâtard',
              })}
              onConfirm={onConfirm}
            />
          </div>
          <button type="button" onClick={() => mapPick.current?.('RVN')}>
            clic carte RVN
          </button>
        </OrderFormProvider>
      </MapLinksProvider>
    </LanguageProvider>
  )
}

describe('OrderLauncher', () => {
  it('unfolds its form in the drawer, one at a time', () => {
    render(<Harness />)

    fireEvent.click(screen.getByRole('button', { name: 'Recruter' }))
    expect(screen.getByRole('dialog', { name: 'Recruter (R N)' })).toHaveAttribute(
      'aria-modal',
      'false',
    )
    expect(screen.getByRole('option', { name: 'Ravinval (RVN)' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Doter' }))
    expect(screen.queryByRole('dialog', { name: 'Recruter (R N)' })).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Doter (D N)' })).toBeInTheDocument()
  })

  it('fills the active field from a click on the map', () => {
    render(<Harness />)

    fireEvent.click(screen.getByRole('button', { name: 'Recruter' }))
    const select = screen.getByRole('combobox')
    expect(select).toHaveValue('ROS')
    expect(select).toHaveClass('ring-brass-700')
    fireEvent.click(screen.getByRole('button', { name: 'clic carte RVN' }))
    expect(select).toHaveValue('RVN')

    // A noble field takes the noble standing on the clicked territory.
    fireEvent.click(screen.getByRole('button', { name: 'Doter' }))
    fireEvent.click(screen.getByRole('button', { name: 'clic carte RVN' }))
    expect(screen.getByRole('combobox')).toHaveValue('HUG')
  })

  it('closes on Escape', () => {
    render(<Harness />)

    fireEvent.click(screen.getByRole('button', { name: 'Recruter' }))
    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('writes the line, then its comment in full labels', () => {
    const onConfirm = vi.fn()
    render(<Harness onConfirm={onConfirm} />)

    fireEvent.click(screen.getByRole('button', { name: 'Recruter' }))
    const form = screen.getByRole('dialog')
    expect(within(form).getByText('R N ALB ROS')).toHaveClass('font-mono')
    expect(
      within(form).getByText('recruter Albert sur Rosemont (ROS)'),
    ).toBeInTheDocument()

    fireEvent.click(within(form).getByRole('button', { name: "Ajouter l'ordre" }))
    expect(onConfirm).toHaveBeenCalledWith(
      'R N ALB ROS # recruter Albert sur Rosemont (ROS)',
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})

describe('fullLabelComment', () => {
  it('writes bare territory codes in full, once', () => {
    expect(fullLabelComment('annexer RVN au fief ROS', links)).toBe(
      'annexer Ravinval (RVN) au fief Rosemont (ROS)',
    )
    expect(fullLabelComment('Rosemont (ROS) et HUG', links)).toBe('Rosemont (ROS) et HUG')
    expect(fullLabelComment('annexer RVN', null)).toBe('annexer RVN')
  })
})
