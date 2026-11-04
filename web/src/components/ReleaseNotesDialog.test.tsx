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
vi.mock('@/lib/release-notes', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/lib/release-notes')>()
  return {
    ...original,
    pendingReleaseNotes: (lastSeen: string, current: string) =>
      original.releaseNotesBetween(
        original.parseChangelog(
          '## [2.0.0](u) (2026-10-01)\n\n### Features\n\n* **web:** big thing ([abcdef1](u))\n\n## [1.1.0](u) (2026-09-01)\n\n### Bug Fixes\n\n* small fix ([abcdef2](u))\n',
        ),
        lastSeen,
        current,
      ),
  }
})

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
    vi.stubEnv('VITE_APP_VERSION', '2.0.0')
  })

  it('lists every version since the last seen one with a releases link', () => {
    authState.profile = profile()
    renderDialog()
    expect(screen.getByText(/web: big thing/)).toBeTruthy()
    expect(screen.getByText('small fix')).toBeTruthy()
    const link = screen.getByRole('link', { name: /GitHub/ })
    expect(link.getAttribute('href')).toBe(
      'https://github.com/fogfactory/crown-and-borough/releases',
    )
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
  })
})
