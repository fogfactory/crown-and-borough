import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { PanelSheet } from '@/components/ui/panel-sheet'

function renderSheet(props: Record<string, unknown> = {}) {
  return render(
    <PanelSheet {...props}>
      <p>Panel content</p>
    </PanelSheet>,
  )
}

describe('PanelSheet', () => {
  it('renders its content with the half snap by default', () => {
    renderSheet()

    expect(screen.getByText('Panel content')).toBeInTheDocument()
    expect(screen.getByTestId('panel-sheet')).toHaveAttribute('data-sheet-snap', 'half')
  })

  it('expands to full and back to peek with the chevron button', () => {
    renderSheet()

    const chevron = screen.getByRole('button', { name: 'Expand panel' })
    fireEvent.click(chevron)
    expect(screen.getByTestId('panel-sheet')).toHaveAttribute('data-sheet-snap', 'full')

    fireEvent.click(screen.getByRole('button', { name: 'Collapse panel' }))
    expect(screen.getByTestId('panel-sheet')).toHaveAttribute('data-sheet-snap', 'peek')
  })

  it('supports arrow keys to expand and collapse', () => {
    renderSheet()

    const chevron = screen.getByRole('button', { name: 'Expand panel' })
    fireEvent.keyDown(chevron, { key: 'ArrowUp' })
    expect(screen.getByTestId('panel-sheet')).toHaveAttribute('data-sheet-snap', 'full')

    fireEvent.keyDown(chevron, { key: 'ArrowDown' })
    expect(screen.getByTestId('panel-sheet')).toHaveAttribute('data-sheet-snap', 'peek')
  })

  it('snaps on a vertical drag of the handle', () => {
    renderSheet()
    const sheet = screen.getByTestId('panel-sheet')
    const handle = sheet.querySelector('div[aria-hidden="true"]') as HTMLElement

    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 200 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientY: 300 })
    expect(sheet).toHaveAttribute('data-sheet-snap', 'peek')

    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 300 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientY: 180 })
    expect(sheet).toHaveAttribute('data-sheet-snap', 'half')
  })

  it('keeps the snap on a small drag', () => {
    renderSheet()
    const sheet = screen.getByTestId('panel-sheet')
    const handle = sheet.querySelector('div[aria-hidden="true"]') as HTMLElement

    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 200 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientY: 220 })
    expect(sheet).toHaveAttribute('data-sheet-snap', 'half')
  })

  it('jumps to half when the focus signal increments', () => {
    const { rerender } = renderSheet({ focusSignal: 0 })
    const sheet = screen.getByTestId('panel-sheet')

    fireEvent.click(screen.getByRole('button', { name: 'Expand panel' }))
    expect(sheet).toHaveAttribute('data-sheet-snap', 'full')

    rerender(
      <PanelSheet focusSignal={1}>
        <p>Panel content</p>
      </PanelSheet>,
    )
    expect(sheet).toHaveAttribute('data-sheet-snap', 'half')
  })

  it('forwards the custom className', () => {
    renderSheet({ className: 'lg:w-96' })

    expect(screen.getByTestId('panel-sheet').className).toContain('lg:w-96')
  })

  it('renders without errors when callbacks are absent', () => {
    expect(() => renderSheet({ focusSignal: 0 })).not.toThrow()
  })
})
