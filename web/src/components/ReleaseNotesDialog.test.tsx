import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ReleaseNotesDialog } from '@/components/ReleaseNotesDialog'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { ProfileData } from '@/types'

const authState = vi.hoisted(() => ({
  profile: null as ProfileData | null,
  updateReleaseNotes: vi.fn(),
}))
vi.mock('@/auth/useAuth', () => ({ useAuth: () => authState }))
const fetchReleaseNotes = vi.hoisted(() => vi.fn())
vi.mock('@/lib/release-notes', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/lib/release-notes')>()
  return { ...original, fetchReleaseNotes }
})

const ENTRIES = [
  {
    version: '2.0.0',
    date: '2026-10-01',
    sections: [{ title: 'Features', items: ['web: big thing'] }],
  },
  {
    version: '1.1.0',
    date: '2026-09-01',
    sections: [{ title: 'Bug Fixes', items: ['small fix'] }],
  },
]

function profile(extra: Partial<ProfileData> = {}): ProfileData {
  return {
    uid: 'u',
    email: 'a@example.test',
    displayName: 'Alice',
    lastSeenVersion: '1.0.0',
    hideReleaseNotes: false,
    ...extra,
  }
}

function renderDialog() {
  return render(
    <LanguageProvider initialLanguage="en">
      <ReleaseNotesDialog />
    </LanguageProvider>,
  )
}

describe('ReleaseNotesDialog', () => {
  beforeEach(() => {
    authState.updateReleaseNotes = vi.fn().mockResolvedValue(profile())
    fetchReleaseNotes.mockReset().mockResolvedValue(ENTRIES)
    vi.stubEnv('VITE_APP_VERSION', '2.0.0')
  })

  it('fetches and lists every version since the last seen one', async () => {
    authState.profile = profile()
    renderDialog()
    expect(screen.getByText('Loading the release notes...')).toBeTruthy()
    expect(await screen.findByText('web: big thing')).toBeTruthy()
    expect(screen.getByText('small fix')).toBeTruthy()
    expect(fetchReleaseNotes).toHaveBeenCalledWith(
      '1.0.0',
      '2.0.0',
      expect.any(AbortSignal),
    )
    const link = screen.getByRole('link', { name: /GitHub/ })
    expect(link.getAttribute('href')).toBe(
      'https://github.com/fogfactory/crown-and-borough/releases',
    )
  })

  it('points to GitHub when the notes cannot be fetched', async () => {
    fetchReleaseNotes.mockRejectedValue(new Error('offline'))
    authState.profile = profile()
    renderDialog()
    expect(await screen.findByText(/could not be loaded/)).toBeTruthy()
    expect(screen.getByRole('link', { name: /GitHub/ })).toBeTruthy()
  })

  it('records the current version on close', async () => {
    authState.profile = profile()
    renderDialog()
    fireEvent.click(screen.getAllByRole('button', { name: 'Close' })[1])
    await waitFor(() =>
      expect(authState.updateReleaseNotes).toHaveBeenCalledWith({
        lastSeenVersion: '2.0.0',
        hideReleaseNotes: false,
      }),
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('also saves the opt-out when the box is checked', async () => {
    authState.profile = profile()
    renderDialog()
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(screen.getAllByRole('button', { name: 'Close' })[1])
    await waitFor(() =>
      expect(authState.updateReleaseNotes).toHaveBeenCalledWith({
        lastSeenVersion: '2.0.0',
        hideReleaseNotes: true,
      }),
    )
  })

  it('stays hidden when up to date, opted out, or on a dev build', () => {
    authState.profile = profile({ lastSeenVersion: '2.0.0' })
    const upToDate = renderDialog()
    expect(screen.queryByRole('dialog')).toBeNull()
    upToDate.unmount()

    authState.profile = profile({ hideReleaseNotes: true })
    const optedOut = renderDialog()
    expect(screen.queryByRole('dialog')).toBeNull()
    optedOut.unmount()

    authState.profile = profile()
    vi.stubEnv('VITE_APP_VERSION', '')
    renderDialog()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(fetchReleaseNotes).not.toHaveBeenCalled()
  })
})
