import { type ReactNode } from 'react'
import {
  BrowserRouter,
  Link,
  Navigate,
  Outlet,
  Route,
  NavLink,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import { AuthProvider, useAuth } from '@/auth/AuthProvider'
import { BrandMark } from '@/components/BrandMark'
import { FinishPage, ProfilePage, SignInPage } from '@/online/AuthPages'
import { GamePage } from '@/online/GamePage'
import { HomePage } from '@/online/HomePage'
import { JoinPage } from '@/online/JoinPage'
import { InfoPage, type InfoPageKind } from '@/components/InfoPage'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { VersionBadge } from '@/components/VersionBadge'
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
    <header className="z-30 shrink-0 border-b border-[#b7a786]/60 bg-[#fffaf0]/95 px-3 py-2 shadow-sm backdrop-blur-sm sm:px-6 lg:sticky lg:top-0">
      <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-2 sm:gap-4">
        <Link to="/" className="flex items-center gap-2.5" aria-label="Crown & Borough">
          <BrandMark className="size-9 sm:size-11" />
          <span>
            <span className="block font-serif text-base font-semibold tracking-tight sm:text-xl">
              Crown &amp; Borough
            </span>
            <span className="hidden text-[10px] uppercase tracking-[0.18em] text-[#806f57] min-[420px]:block">
              {t('app.tagline')}
            </span>
            <VersionBadge />
          </span>
        </Link>
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <nav aria-label={t('nav.primary')} className="flex items-center gap-1">
            <NavLink
              to="/rules"
              className={({ isActive }) =>
                `rounded-md px-2.5 py-1.5 text-sm font-semibold transition ${isActive ? 'bg-[#f3ead9] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:bg-[#f3ead9] hover:text-[#30291f]'}`
              }
            >
              {t('nav.rules')}
            </NavLink>
            <NavLink
              to="/faq"
              className={({ isActive }) =>
                `rounded-md px-2.5 py-1.5 text-sm font-semibold transition ${isActive ? 'bg-[#f3ead9] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:bg-[#f3ead9] hover:text-[#30291f]'}`
              }
            >
              {t('nav.faq')}
            </NavLink>
          </nav>
          {profile && (
            <Link
              to="/profile"
              className="max-w-32 truncate rounded-md px-2 py-1 text-sm font-medium text-[#594b3c] underline-offset-4 hover:text-[#a84632] hover:underline sm:max-w-none"
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
  const location = useLocation()
  const isGame = /^\/games\//.test(location.pathname)
  return (
    <div
      lang={language}
      className={`flex flex-col bg-[#efe7d8] text-[#30291f] ${
        isGame ? 'h-dvh overflow-hidden' : 'min-h-dvh'
      }`}
    >
      <OnlineHeader />
      <main
        className={`mx-auto flex w-full min-h-0 max-w-[1500px] flex-1 flex-col p-3 sm:p-4 lg:p-6 ${
          isGame ? 'overflow-hidden' : ''
        }`}
      >
        {children}
      </main>
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

function InfoRoute({ kind }: { kind: InfoPageKind }) {
  return (
    <OnlineFrame>
      <InfoPage kind={kind} />
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
          <Route path="/rules" element={<InfoRoute kind="rules" />} />
          <Route path="/faq" element={<InfoRoute kind="faq" />} />
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
