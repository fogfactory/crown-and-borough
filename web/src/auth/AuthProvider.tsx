import {
  browserLocalPersistence,
  isSignInWithEmailLink,
  onAuthStateChanged,
  sendSignInLinkToEmail,
  setPersistence,
  signInWithEmailLink,
  signOut as firebaseSignOut,
  type User,
} from 'firebase/auth'
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useCallback,
  useState,
  type ReactNode,
} from 'react'

import { ApiError, apiRequest, type TokenProvider } from '@/lib/api'
import { firebaseConfigured, getFirebaseServices } from '@/lib/firebase'
import type { ProfileData } from '@/types'

const EMAIL_STORAGE_KEY = 'crown-and-borough.sign-in-email'
const MISSING_EMAIL_ERROR = 'the email used to request the sign-in link is missing'

export type AuthStatus = 'loading' | 'signed-out' | 'signed-in' | 'unconfigured'

interface AuthContextValue extends TokenProvider {
  status: AuthStatus
  user: User | null
  profile: ProfileData | null
  profileLoading: boolean
  profileError: string | null
  authError: string | null
  sendSignInLink: (email: string, redirectPath?: string) => Promise<void>
  completeSignIn: (href?: string, email?: string) => Promise<boolean>
  updateProfile: (displayName: string) => Promise<ProfileData>
  refreshProfile: () => Promise<ProfileData | null>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return 'authentication failed'
}

function redirectPathFromHref(href: string): string | null {
  try {
    return new URL(href).searchParams.get('redirect')
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const services = getFirebaseServices()
  const [user, setUser] = useState<User | null>(null)
  const [authReady, setAuthReady] = useState(!services)
  const [authError, setAuthError] = useState<string | null>(null)
  const [profile, setProfile] = useState<ProfileData | null>(null)
  const [profileLoading, setProfileLoading] = useState(Boolean(services))
  const [profileError, setProfileError] = useState<string | null>(null)

  useEffect(() => {
    if (!services) return

    let active = true
    let unsubscribe: () => void = () => undefined
    void setPersistence(services.auth, browserLocalPersistence)
      .then(() => {
        if (!active) return
        unsubscribe = onAuthStateChanged(services.auth, (nextUser) => {
          if (!active) return
          setUser(nextUser)
          setAuthReady(true)
        })
      })
      .catch((error: unknown) => {
        if (!active) return
        setAuthError(errorMessage(error))
        setAuthReady(true)
      })

    return () => {
      active = false
      unsubscribe()
    }
  }, [services])

  const getIdToken = useCallback(async () => {
    if (!user) return null
    return user.getIdToken()
  }, [user])

  const refreshProfile = useCallback(async (): Promise<ProfileData | null> => {
    if (!user) {
      setProfile(null)
      return null
    }
    setProfileLoading(true)
    setProfileError(null)
    try {
      const response = await apiRequest<{ player: ProfileData }>(
        { getIdToken },
        '/api/auth/me',
      )
      setProfile(response.player)
      return response.player
    } catch (error) {
      setProfileError(errorMessage(error))
      if (error instanceof ApiError && error.status === 401 && services) {
        await firebaseSignOut(services.auth).catch(() => undefined)
      }
      throw error
    } finally {
      setProfileLoading(false)
    }
  }, [getIdToken, services, user])

  useEffect(() => {
    if (!authReady) return
    if (!user) {
      setProfile(null)
      setProfileLoading(false)
      return
    }
    void refreshProfile().catch(() => undefined)
  }, [authReady, refreshProfile, user])

  const sendSignInLink = useCallback(
    async (email: string, redirectPath?: string) => {
      if (!services) throw new Error('Firebase is not configured')
      const destination =
        redirectPath ?? `${window.location.pathname}${window.location.search}`
      const continuation = new URL('/finish', window.location.origin)
      if (destination && destination !== '/') {
        continuation.searchParams.set('redirect', destination)
      }
      await sendSignInLinkToEmail(services.auth, email, {
        url: continuation.toString(),
        handleCodeInApp: true,
      })
      window.localStorage.setItem(EMAIL_STORAGE_KEY, email)
      window.sessionStorage.setItem(EMAIL_STORAGE_KEY, email)
    },
    [services],
  )

  const completeSignIn = useCallback(
    async (href = window.location.href, emailOverride?: string): Promise<boolean> => {
      if (!services || !isSignInWithEmailLink(services.auth, href)) return false
      const email =
        emailOverride?.trim() ||
        window.sessionStorage.getItem(EMAIL_STORAGE_KEY) ||
        window.localStorage.getItem(EMAIL_STORAGE_KEY)
      if (!email) throw new Error(MISSING_EMAIL_ERROR)
      const credential = await signInWithEmailLink(services.auth, email, href)
      // Custom claims are assigned outside the browser. Force the first token
      // refresh so a newly granted game_creator claim is used immediately.
      await credential.user.getIdToken(true)
      window.localStorage.removeItem(EMAIL_STORAGE_KEY)
      window.sessionStorage.removeItem(EMAIL_STORAGE_KEY)
      return true
    },
    [services],
  )

  const updateProfile = useCallback(
    async (displayName: string): Promise<ProfileData> => {
      const response = await apiRequest<{ player: ProfileData }>(
        { getIdToken },
        '/api/auth/me',
        {
          method: 'PUT',
          body: JSON.stringify({ displayName }),
        },
      )
      setProfile(response.player)
      setProfileError(null)
      return response.player
    },
    [getIdToken],
  )

  const signOut = useCallback(async () => {
    if (!services) return
    await firebaseSignOut(services.auth)
    setProfile(null)
  }, [services])

  const value = useMemo<AuthContextValue>(
    () => ({
      status: !firebaseConfigured
        ? 'unconfigured'
        : !authReady
          ? 'loading'
          : user
            ? 'signed-in'
            : 'signed-out',
      user,
      profile,
      profileLoading,
      profileError,
      authError,
      getIdToken,
      sendSignInLink,
      completeSignIn,
      updateProfile,
      refreshProfile,
      signOut,
    }),
    [
      authError,
      authReady,
      completeSignIn,
      getIdToken,
      profile,
      profileError,
      profileLoading,
      refreshProfile,
      sendSignInLink,
      signOut,
      updateProfile,
      user,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}

export { MISSING_EMAIL_ERROR, redirectPathFromHref }
