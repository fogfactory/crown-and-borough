import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { VictorySimulatorDialog } from '@/components/VictorySimulatorDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { Noble, VictoryReading, VictorySimulation } from '@/types'

const players = [
  { id: 'P1', name: 'Alice', color: '#a84632' },
  { id: 'P2', name: 'Bob', color: '#2f6f9f' },
]
const nobles: Noble[] = [
  { id: 'N1', code: 'AAA', name: 'Anne', owner: 'P1', location: 'T1', status: 'free' },
  { id: 'N2', code: 'BBB', name: 'Bastien', owner: 'P2', location: 'T2', status: 'free' },
]
const reading = (total: number, status: VictoryReading['status']): VictoryReading => ({
  score: { titles: total, total },
  victory: { mode: 'solo', required: 3 },
  status,
  missing: Math.max(3 - total, 0),
})

describe('VictorySimulatorDialog', () => {
  it('sends the stacked hypotheses and shows the projected change', async () => {
    const request = vi.fn(
      async (): Promise<VictorySimulation> => ({
        current: reading(1, 'ongoing'),
        projected: reading(3, 'major'),
      }),
    )
    render(
      <LanguageProvider initialLanguage="en">
        <VictorySimulatorDialog players={players} nobles={nobles} playerId="P1" request={request} />
      </LanguageProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Simulate victory' }))
    fireEvent.change(screen.getByLabelText('Noble'), { target: { value: 'AAA' } })
    fireEvent.change(screen.getByLabelText('Spouse'), { target: { value: 'BBB' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith(
        expect.objectContaining({ marriages: [{ noble: 'AAA', spouse: 'BBB' }] }),
        expect.anything(),
      ),
    )
    expect(await screen.findByTestId('sim-delta')).toHaveTextContent('+2')
    expect(screen.getByTestId('sim-status-projected')).toHaveTextContent('Major victory')
    expect(screen.getByTestId('sim-status-current')).toHaveTextContent('No victory yet')
  })

  it('reports an impossible scenario', async () => {
    const request = vi.fn(async (): Promise<VictorySimulation> => {
      throw new Error('422')
    })
    render(
      <LanguageProvider initialLanguage="en">
        <VictorySimulatorDialog players={players} nobles={nobles} playerId="P1" request={request} />
      </LanguageProvider>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Simulate victory' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('not possible')
  })
})
