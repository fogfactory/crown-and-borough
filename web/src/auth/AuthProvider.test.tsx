import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

const authHarness = vi.hoisted(() => ({
  auth: {},
  user: {
    uid: 'alice-uid',
    email: 'alice@example.test',
    getIdToken: vi.fn(async () => 'alice-token'),
  },
  stateListener: null as ((user: unknown) => void) | null,
  signInLink: false,
  unsubscribe: vi.fn(),
  sendSignInLinkToEmail: vi.fn(async () => undefined),
  signInWithEmailLink: vi.fn(async () => ({ user: authHarness.user })),
  firebaseSignOut: vi.fn(async () => undefined),
}))

vi.mock('@/lib/firebase', () => ({
  firebaseConfigured: true,
  getFirebaseServices: () => ({ auth: authHarness.auth, firestore: {} }),
}))

vi.mock('firebase/auth', () => ({
  browserLocalPersistence: {},
  isSignInWithEmailLink: vi.fn(() => authHarness.signInLink),
  onAuthStateChanged: vi.fn((_auth: unknown, listener: (user: unknown) => void) => {
    authHarness.stateListener = listener
    return authHarness.unsubscribe
  }),
  sendSignInLinkToEmail: authHarness.sendSignInLinkToEmail,
  setPersistence: vi.fn(async () => undefined),
  signInWithEmailLink: authHarness.signInWithEmailLink,
  signOut: authHarness.firebaseSignOut,
}))

import { AuthProvider, useAuth } from '@/auth/AuthProvider'

afterEach(() => {
  vi.unstubAllGlobals()
  authHarness.stateListener = null
  authHarness.signInLink = false
  authHarness.sendSignInLinkToEmail.mockClear()
  authHarness.signInWithEmailLink.mockClear()
  authHarness.firebaseSignOut.mockClear()
  authHarness.unsubscribe.mockClear()
  authHarness.user.getIdToken.mockClear()
  window.sessionStorage.clear()
  window.localStorage.clear()
})

function Probe() {
  const { status, profile, profileLoading, sendSignInLink, completeSignIn, getIdToken } =
    useAuth()
  return (
    <div>
      <output data-testid="status">{status}</output>
      <output data-testid="profile">{profile?.displayName ?? ''}</output>
      <output data-testid="profile-loading">{String(profileLoading)}</output>
      <button
        type="button"
        onClick={() => void sendSignInLink('alice@example.test', '/join?gameId=game-1')}
      >
        send
      </button>
      <button
        type="button"
        onClick={() => void completeSignIn('https://example.test/finish?oobCode=1')}
      >
        complete
      </button>
      <button type="button" onClick={() => void getIdToken()}>
        token
      </button>
    </div>
  )
}

describe('Firebase auth provider', () => {
  it('restores a Firebase account and loads the UID profile', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              player: {
                uid: 'alice-uid',
                email: 'alice@example.test',
                displayName: 'Alice',
              },
            }),
            {
              status: 200,
              headers: { 'Content-Type': 'application/json' },
            },
          ),
      ),
    )
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )

    await waitFor(() => expect(authHarness.stateListener).not.toBeNull())
    expect(screen.getByTestId('profile-loading')).toHaveTextContent('true')
    act(() => authHarness.stateListener?.(authHarness.user))

    await waitFor(() => {
      expect(screen.getByTestId('status')).toHaveTextContent('signed-in')
      expect(screen.getByTestId('profile')).toHaveTextContent('Alice')
    })
    fireEvent.click(screen.getByRole('button', { name: 'token' }))
    await waitFor(() => expect(authHarness.user.getIdToken).toHaveBeenCalledWith())
  })

  it('sends and completes an email link without storing an ID token', async () => {
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )
    await waitFor(() => expect(authHarness.stateListener).not.toBeNull())

    fireEvent.click(screen.getByRole('button', { name: 'send' }))
    await waitFor(() => expect(authHarness.sendSignInLinkToEmail).toHaveBeenCalledOnce())
    expect(window.sessionStorage.getItem('crown-and-borough.sign-in-email')).toBe(
      'alice@example.test',
    )
    expect(window.localStorage.getItem('crown-and-borough.sign-in-email')).toBe(
      'alice@example.test',
    )
    expect(authHarness.sendSignInLinkToEmail).toHaveBeenCalledWith(
      authHarness.auth,
      'alice@example.test',
      expect.objectContaining({ handleCodeInApp: true }),
    )

    authHarness.signInLink = true
    fireEvent.click(screen.getByRole('button', { name: 'complete' }))
    await waitFor(() => expect(authHarness.signInWithEmailLink).toHaveBeenCalledOnce())
    expect(authHarness.user.getIdToken).toHaveBeenCalledWith(true)
    expect(window.sessionStorage.getItem('crown-and-borough.sign-in-email')).toBeNull()
    expect(window.localStorage.getItem('crown-and-borough.sign-in-email')).toBeNull()
  })

  it('completes an email link from local storage when it opens in another tab', async () => {
    window.localStorage.setItem('crown-and-borough.sign-in-email', 'alice@example.test')
    authHarness.signInLink = true
    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    )
    await waitFor(() => expect(authHarness.stateListener).not.toBeNull())

    fireEvent.click(screen.getByRole('button', { name: 'complete' }))
    await waitFor(() => expect(authHarness.signInWithEmailLink).toHaveBeenCalledOnce())
    expect(authHarness.user.getIdToken).toHaveBeenCalledWith(true)
    expect(window.localStorage.getItem('crown-and-borough.sign-in-email')).toBeNull()
  })
})
