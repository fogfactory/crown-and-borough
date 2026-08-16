import { type ReactNode } from 'react'
import {
  BrowserRouter,
  Link,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import { AuthProvider, useAuth } from '@/auth/AuthProvider'
import { FinishPage, ProfilePage, SignInPage } from '@/online/AuthPages'
import { GamePage } from '@/online/GamePage'
import { HomePage } from '@/online/HomePage'
import { JoinPage } from '@/online/JoinPage'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import { firebaseConfigured } from '@/lib/firebase'

function OnlineHeader() {
  const { profile, signOut } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()

  const handleSignOut = async () => {
    await signOut()
    navigate('/signin', { replace: true })
  }

  return (
    <header className="border-b border-[#b7a786]/60 bg-[#fffaf0]/90 px-4 py-4 shadow-sm backdrop-blur-sm sm:px-6">
      <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-4">
        <Link to="/" className="flex items-center gap-3" aria-label="Crown & Borough">
          <span className="flex size-11 items-center justify-center rounded-full border-2 border-[#a84632] bg-[#f6dfc6] font-serif text-sm font-bold text-[#a84632] shadow-inner">
            C&amp;B
          </span>
          <span>
            <span className="block font-serif text-xl font-semibold tracking-tight">
              Crown &amp; Borough
            </span>
            <span className="block text-[10px] uppercase tracking-[0.18em] text-[#806f57]">
              {t('app.tagline')}
            </span>
          </span>
        </Link>
        <div className="flex flex-wrap items-center gap-3">
          {profile && (
            <Link
              to="/profile"
              className="rounded-md px-2 py-1 text-sm font-medium text-[#594b3c] underline-offset-4 hover:text-[#a84632] hover:underline"
            >
              {profile.displayName || profile.email}
            </Link>
          )}
          <LanguageSwitcher />
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void handleSignOut()}
          >
            {t('auth.signOut')}
          </Button>
        </div>
      </div>
    </header>
  )
}

export function OnlineFrame({ children }: { children: ReactNode }) {
  const { language } = useLanguage()
  return (
    <div lang={language} className="min-h-screen bg-[#efe7d8] text-[#30291f]">
      <OnlineHeader />
      <main className="mx-auto max-w-[1500px] p-4 sm:p-6">{children}</main>
    </div>
  )
}

function RequireAuth() {
  const { status, authError } = useAuth()
  const { t } = useLanguage()
  const location = useLocation()

  if (status === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#efe7d8] px-6 text-center">
        <p className="font-serif text-lg italic text-[#806f57]">
          {authError ?? t('online.loading')}
        </p>
      </div>
    )
  }
  if (status !== 'signed-in') {
    const redirect = `${location.pathname}${location.search}`
    return <Navigate to={`/signin?redirect=${encodeURIComponent(redirect)}`} replace />
  }
  return <Outlet />
}

function RequireProfile() {
  const { profile, profileLoading, profileError } = useAuth()
  const location = useLocation()

  if (profileLoading) {
    return (
      <OnlineFrame>
        <p className="py-20 text-center font-serif text-lg italic text-[#806f57]">
          Loading profile...
        </p>
      </OnlineFrame>
    )
  }
  if (profileError && !profile) {
    return (
      <OnlineFrame>
        <p
          role="alert"
          className="mx-auto max-w-xl rounded-lg bg-[#f8e5dd] p-4 text-[#8d321e]"
        >
          {profileError}
        </p>
      </OnlineFrame>
    )
  }
  if (!profile?.displayName.trim()) {
    const redirect = `${location.pathname}${location.search}`
    return <Navigate to={`/profile?redirect=${encodeURIComponent(redirect)}`} replace />
  }
  return <Outlet />
}

function ProfileRoute() {
  return (
    <OnlineFrame>
      <ProfilePage />
    </OnlineFrame>
  )
}

function SignInRoute() {
  return (
    <div className="min-h-screen bg-[#efe7d8] px-4 py-8 text-[#30291f] sm:px-6 sm:py-16">
      <SignInPage />
    </div>
  )
}

function FinishRoute() {
  return (
    <div className="min-h-screen bg-[#efe7d8] px-4 py-8 text-[#30291f] sm:px-6 sm:py-16">
      <FinishPage />
    </div>
  )
}

function JoinRoute() {
  return (
    <OnlineFrame>
      <JoinPage />
    </OnlineFrame>
  )
}

function GameRoute() {
  return (
    <OnlineFrame>
      <GamePage />
    </OnlineFrame>
  )
}

export function OnlineRoutes() {
  return (
    <Routes>
      <Route path="/signin" element={<SignInRoute />} />
      <Route path="/finish" element={<FinishRoute />} />
      <Route element={<RequireAuth />}>
        <Route path="/profile" element={<ProfileRoute />} />
        <Route element={<RequireProfile />}>
          <Route
            path="/"
            element={
              <OnlineFrame>
                <HomePage />
              </OnlineFrame>
            }
          />
          <Route path="/join" element={<JoinRoute />} />
          <Route path="/games/:gameId" element={<GameRoute />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export function OnlineApp() {
  const { t } = useLanguage()
  if (!firebaseConfigured) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#efe7d8] px-6 text-center text-[#30291f]">
        <div className="max-w-md space-y-3">
          <h1 className="font-serif text-3xl font-semibold">{t('online.configTitle')}</h1>
          <p className="text-sm text-[#806f57]">{t('online.configDescription')}</p>
        </div>
      </div>
    )
  }
  return (
    <AuthProvider>
      <BrowserRouter>
        <OnlineRoutes />
      </BrowserRouter>
    </AuthProvider>
  )
}
