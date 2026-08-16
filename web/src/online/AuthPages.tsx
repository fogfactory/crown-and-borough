import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ApiError } from '@/lib/api'
import { useLanguage } from '@/i18n/LanguageContext'

function safeRedirect(value: string | null): string {
  return value?.startsWith('/') ? value : '/'
}

function errorText(error: unknown, fallback: string): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return fallback
}

export function SignInPage() {
  const { status, sendSignInLink } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const redirect = safeRedirect(searchParams.get('redirect'))

  useEffect(() => {
    if (status === 'signed-in') navigate(redirect, { replace: true })
  }, [navigate, redirect, status])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const value = email.trim()
    if (!value) return
    setSending(true)
    setError(null)
    try {
      await sendSignInLink(value, redirect)
      setSent(true)
    } catch (submitError) {
      setError(errorText(submitError, t('error.authRequired')))
    } finally {
      setSending(false)
    }
  }

  return (
    <Card className="mx-auto max-w-md border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader>
        <Link
          to="/"
          className="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-[#a84632]"
        >
          Crown &amp; Borough
        </Link>
        <CardTitle className="font-serif text-3xl text-[#30291f]">
          {t('auth.title')}
        </CardTitle>
        <CardDescription className="text-[#806f57]">
          {sent ? t('auth.linkSentHint') : t('profile.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {sent ? (
          <div className="space-y-4" role="status" aria-live="polite">
            <p className="rounded-lg border border-[#376341]/30 bg-[#e8f1e3] px-3 py-3 text-sm text-[#376341]">
              {t('auth.linkSent')}: <strong>{email}</strong>
            </p>
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={() => setSent(false)}
            >
              {t('auth.sendLink')}
            </Button>
          </div>
        ) : (
          <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
            <label className="block space-y-1.5">
              <span className="text-sm font-semibold text-[#594b3c]">
                {t('auth.email')}
              </span>
              <input
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="h-11 w-full rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
              />
            </label>
            {error && (
              <p
                role="alert"
                className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
              >
                {error}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={sending || !email.trim()}>
              {sending ? t('auth.sendingLink') : t('auth.sendLink')}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  )
}

export function FinishPage() {
  const { completeSignIn, status } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [error, setError] = useState<string | null>(null)
  const started = useRef(false)
  const redirect = safeRedirect(searchParams.get('redirect'))

  useEffect(() => {
    if (started.current) return
    started.current = true
    if (status === 'signed-in') {
      navigate(redirect, { replace: true })
      return
    }
    void completeSignIn()
      .then((completed) => {
        if (completed) navigate(redirect, { replace: true })
        else setError(t('auth.invalidLink'))
      })
      .catch((signInError: unknown) => {
        setError(errorText(signInError, t('auth.invalidLink')))
      })
  }, [completeSignIn, navigate, redirect, status, t])

  return (
    <Card className="mx-auto max-w-md border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader>
        <CardTitle className="font-serif text-2xl text-[#30291f]">
          {t('auth.verifyLink')}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {error ? (
          <div className="space-y-4">
            <p
              role="alert"
              className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
            >
              {error}
            </p>
            <Link
              to="/signin"
              className="block text-center text-sm font-semibold text-[#a84632] underline underline-offset-4"
            >
              {t('auth.backToSignIn')}
            </Link>
          </div>
        ) : (
          <p className="text-sm text-[#806f57]">{t('online.loading')}</p>
        )}
      </CardContent>
    </Card>
  )
}

export function ProfilePage() {
  const { profile, profileLoading, updateProfile } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [displayName, setDisplayName] = useState(profile?.displayName ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const redirect = safeRedirect(searchParams.get('redirect'))

  useEffect(() => {
    if (profile) setDisplayName(profile.displayName)
  }, [profile])

  if (profileLoading && !profile) {
    return (
      <p className="py-20 text-center font-serif text-lg italic text-[#806f57]">
        {t('online.loading')}
      </p>
    )
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      await updateProfile(displayName.trim())
      navigate(redirect, { replace: true })
    } catch (submitError) {
      setError(errorText(submitError, t('error.profileRequired')))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="mx-auto max-w-xl border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader>
        <CardTitle className="font-serif text-3xl text-[#30291f]">
          {t('profile.title')}
        </CardTitle>
        <CardDescription className="text-[#806f57]">
          {t('profile.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
          <label className="block space-y-1.5">
            <span className="text-sm font-semibold text-[#594b3c]">
              {t('profile.displayName')}
            </span>
            <input
              type="text"
              required
              minLength={1}
              maxLength={32}
              value={displayName}
              placeholder={t('profile.displayNamePlaceholder')}
              onChange={(event) => setDisplayName(event.target.value)}
              className="h-11 w-full rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
            />
          </label>
          {error && (
            <p
              role="alert"
              className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
            >
              {error}
            </p>
          )}
          <Button
            type="submit"
            className="w-full"
            disabled={saving || !displayName.trim()}
          >
            {saving ? t('profile.saving') : t('profile.save')}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
