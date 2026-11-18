import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { OUT_OF_SYNC_GRACE_MS, OutOfSyncDialog } from '@/components/OutOfSyncDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'

function renderDialog(props: Partial<Parameters<typeof OutOfSyncDialog>[0]> = {}) {
  const onReload = vi.fn()
  const view = render(
    <LanguageProvider>
      <OutOfSyncDialog
        serverRevision={5}
        displayedRevision={5}
        onReload={onReload}
        {...props}
      />
    </LanguageProvider>,
  )
  return { onReload, ...view }
}

describe('OutOfSyncDialog', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('stays closed while the screen is up to date', () => {
    renderDialog()
    act(() => vi.advanceTimersByTime(OUT_OF_SYNC_GRACE_MS * 2))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('waits for the grace delay before opening', () => {
    renderDialog({ serverRevision: 6, displayedRevision: 5 })
    act(() => vi.advanceTimersByTime(OUT_OF_SYNC_GRACE_MS - 100))
    expect(screen.queryByRole('dialog')).toBeNull()
    act(() => vi.advanceTimersByTime(200))
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('never opens when the screen catches up in time', () => {
    const { rerender, onReload } = renderDialog({ serverRevision: 6, displayedRevision: 5 })
    act(() => vi.advanceTimersByTime(1000))
    rerender(
      <LanguageProvider>
        <OutOfSyncDialog serverRevision={6} displayedRevision={6} onReload={onReload} />
      </LanguageProvider>,
    )
    act(() => vi.advanceTimersByTime(OUT_OF_SYNC_GRACE_MS * 2))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('opens when the live connection is lost and reloads on demand', () => {
    const { onReload } = renderDialog({ connectionLost: true })
    act(() => vi.advanceTimersByTime(OUT_OF_SYNC_GRACE_MS + 1))
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons.find((b) => /reload|recharger/i.test(b.textContent ?? ''))!)
    expect(onReload).toHaveBeenCalledOnce()
  })

  it('can be dismissed', () => {
    renderDialog({ serverRevision: 6, displayedRevision: 5 })
    act(() => vi.advanceTimersByTime(OUT_OF_SYNC_GRACE_MS + 1))
    fireEvent.click(
      screen.getAllByRole('button').find((b) => /not now|plus tard/i.test(b.textContent ?? ''))!,
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})
