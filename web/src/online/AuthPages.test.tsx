import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { LanguageProvider } from '@/i18n/LanguageContext'
import { ProfilePage } from '@/online/AuthPages'
import type { ProfileData } from '@/types'

const authState = vi.hoisted(() => ({
  profile: null as ProfileData | null,
  profileLoading: false,
  updateProfile: vi.fn(),
  updateReleaseNotes: vi.fn(),
  updateNotifications: vi.fn(),
}))
vi.mock('@/auth/AuthProvider', () => ({
  MISSING_EMAIL_ERROR: 'missing email',
  useAuth: () => authState,
}))

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

function renderPage(language: 'en' | 'fr' = 'en') {
  return render(
    <LanguageProvider initialLanguage={language}>
      <MemoryRouter>
        <ProfilePage />
      </MemoryRouter>
    </LanguageProvider>,
  )
}

describe('ProfilePage notifications', () => {
  beforeEach(() => {
    authState.updateNotifications = vi.fn().mockResolvedValue(profile())
  })

  it('shows every notification off for a profile without the settings', () => {
    authState.profile = profile() // no `notifications`: old response
    renderPage()
    for (const name of ['New turn', 'Private messages', 'Public room']) {
      expect(screen.getByRole('checkbox', { name: new RegExp(name) })).not.toBeChecked()
    }
    const reduce = screen.getByRole('checkbox', { name: /Reduce notifications/ })
    expect(reduce).toBeDisabled()
    expect(
      screen.getByText('Turn on at least one notification above to use this option.'),
    ).toBeInTheDocument()
  })

  it('records the app language with the first setting saved', async () => {
    authState.profile = profile()
    renderPage('fr')
    fireEvent.click(screen.getByRole('checkbox', { name: /Nouveau tour/ }))
    await waitFor(() =>
      expect(authState.updateNotifications).toHaveBeenCalledWith({
        newTurn: true,
        language: 'fr',
      }),
    )
  })

  it('enables the reduce option once a notification is on', async () => {
    authState.profile = profile({
      notifications: {
        newTurn: false,
        privateMessage: true,
        publicMessage: false,
        reduceWhenActive: false,
        language: 'en',
      },
    })
    renderPage()
    const reduce = screen.getByRole('checkbox', { name: /Reduce notifications/ })
    expect(reduce).toBeEnabled()
    expect(screen.getByRole('checkbox', { name: /Private messages/ })).toBeChecked()
    fireEvent.click(reduce)
    await waitFor(() =>
      expect(authState.updateNotifications).toHaveBeenCalledWith({
        reduceWhenActive: true,
      }),
    )
  })

  it('lets the player choose the language of the emails', async () => {
    authState.profile = profile({
      notifications: {
        newTurn: true,
        privateMessage: false,
        publicMessage: false,
        reduceWhenActive: false,
        language: 'en',
      },
    })
    renderPage('fr')
    const select = screen.getByRole('combobox', { name: 'Langue des emails' })
    expect(select).toHaveValue('en')
    fireEvent.change(select, { target: { value: 'fr' } })
    await waitFor(() =>
      expect(authState.updateNotifications).toHaveBeenCalledWith({ language: 'fr' }),
    )
  })

  it('shows the app language until one is saved', () => {
    authState.profile = profile()
    renderPage('en')
    expect(screen.getByRole('combobox', { name: 'Language of the emails' })).toHaveValue(
      'en',
    )
  })

  it('reports a save failure', async () => {
    authState.profile = profile()
    authState.updateNotifications = vi.fn().mockRejectedValue(new Error('boom'))
    renderPage()
    fireEvent.click(screen.getByRole('checkbox', { name: /Public room/ }))
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
