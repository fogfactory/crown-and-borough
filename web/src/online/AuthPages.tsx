import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { MISSING_EMAIL_ERROR, useAuth } from '@/auth/AuthProvider'
import { Seal } from '@/components/Seal'
import { Alert } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { errorText } from '@/lib/api-error'
import { cn } from '@/lib/utils'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import {
  NO_NOTIFICATIONS,
  type NotificationPreferences,
  type NotificationsUpdate,
} from '@/types'

function safeRedirect(value: string | null): string {
  return value?.startsWith('/') ? value : '/'
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
      setError(errorText(submitError, t, t('error.authRequired')))
    } finally {
      setSending(false)
    }
  }

  return (
    <Card className="mx-auto max-w-md border-line">
      <CardHeader>
        <Seal variant="full" className="mx-auto mb-2 size-32" />
        <CardTitle className="font-display text-3xl font-bold text-ink-900">
          {t('auth.title')}
        </CardTitle>
        <CardDescription className="text-ink-500">
          {sent ? t('auth.linkSentHint') : t('profile.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {sent ? (
          <div className="space-y-4" role="status" aria-live="polite">
            <Alert tone="success">
              {t('auth.linkSent')}: <strong>{email}</strong>
            </Alert>
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
              <span className="text-sm font-semibold text-ink-700">
                {t('auth.email')}
              </span>
              <Input
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="h-11"
              />
            </label>
            {error && <Alert role="alert">{error}</Alert>}
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
  const [email, setEmail] = useState('')
  const [emailRequired, setEmailRequired] = useState(false)
  const [retrying, setRetrying] = useState(false)
  const started = useRef(false)
  const redirect = safeRedirect(searchParams.get('redirect'))

  const finishSignIn = useCallback(
    async (emailOverride?: string) => {
      setRetrying(true)
      setError(null)
      try {
        const completed = await completeSignIn(window.location.href, emailOverride)
        if (completed) navigate(redirect, { replace: true })
        else setError(t('auth.invalidLink'))
      } catch (signInError: unknown) {
        if (signInError instanceof Error && signInError.message === MISSING_EMAIL_ERROR) {
          setEmailRequired(true)
          setError(t('auth.emailMissingHint'))
        } else {
          setError(errorText(signInError, t, t('auth.invalidLink')))
        }
      } finally {
        setRetrying(false)
      }
    },
    [completeSignIn, navigate, redirect, t],
  )

  useEffect(() => {
    if (started.current) return
    started.current = true
    if (status === 'signed-in') {
      navigate(redirect, { replace: true })
      return
    }
    void finishSignIn()
  }, [finishSignIn, navigate, redirect, status])

  return (
    <Card className="mx-auto max-w-md border-line">
      <CardHeader>
        <CardTitle className="font-display text-2xl text-ink-900">
          {t('auth.verifyLink')}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {error && (
          <Alert role="alert" className="mb-4">
            {error}
          </Alert>
        )}
        {emailRequired && (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              void finishSignIn(email.trim())
            }}
          >
            <label className="block space-y-1.5">
              <span className="text-sm font-semibold text-ink-700">
                {t('auth.emailMissing')}
              </span>
              <Input
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="h-11"
              />
            </label>
            <Button type="submit" className="w-full" disabled={retrying || !email.trim()}>
              {retrying ? t('auth.verifyLink') : t('auth.completeLink')}
            </Button>
          </form>
        )}
        {error && (
          <div className="mt-4">
            <Link
              to="/signin"
              className="block text-center text-sm font-semibold text-brass-700 underline underline-offset-4"
            >
              {t('auth.backToSignIn')}
            </Link>
          </div>
        )}
        {!error && !emailRequired && (
          <p className="text-sm text-ink-500">{t('online.loading')}</p>
        )}
      </CardContent>
    </Card>
  )
}

/** A checkbox with its label, a description and an optional hint. */
function ToggleRow({
  label,
  description,
  hint,
  checked,
  disabled,
  className,
  onChange,
}: {
  label: string
  description: string
  hint?: string
  checked: boolean
  disabled?: boolean
  className?: string
  onChange: (checked: boolean) => void
}) {
  return (
    <label className={cn('flex items-start gap-3', className)}>
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        className="mt-0.5 size-4 accent-brass-700"
      />
      <span>
        <span className="block font-semibold text-ink-700">{label}</span>
        <span className="mt-1 block text-xs text-ink-500">{description}</span>
        {hint && <span className="mt-1 block text-xs italic text-ink-500">{hint}</span>}
      </span>
    </label>
  )
}

type NotificationToggle = 'newTurn' | 'privateMessage' | 'publicMessage'

const NOTIFICATION_TOGGLES: ReadonlyArray<{
  key: NotificationToggle
  label: MessageKey
  description: MessageKey
}> = [
  {
    key: 'newTurn',
    label: 'profile.notifications.newTurn',
    description: 'profile.notifications.newTurnDescription',
  },
  {
    key: 'privateMessage',
    label: 'profile.notifications.privateMessage',
    description: 'profile.notifications.privateMessageDescription',
  },
  {
    key: 'publicMessage',
    label: 'profile.notifications.publicMessage',
    description: 'profile.notifications.publicMessageDescription',
  },
]

export function ProfilePage() {
  const {
    profile,
    profileLoading,
    updateProfile,
    updateReleaseNotes,
    updateNotifications,
  } = useAuth()
  const { t, language } = useLanguage()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [displayName, setDisplayName] = useState(profile?.displayName ?? '')
  const [saving, setSaving] = useState(false)
  const [togglingReleaseNotes, setTogglingReleaseNotes] = useState(false)
  const [savingNotifications, setSavingNotifications] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const redirect = safeRedirect(searchParams.get('redirect'))
  // Profiles saved before the notifications existed carry none: all off.
  const notifications: NotificationPreferences = {
    ...NO_NOTIFICATIONS,
    ...profile?.notifications,
  }
  const anyNotification =
    notifications.newTurn || notifications.privateMessage || notifications.publicMessage

  useEffect(() => {
    if (profile) setDisplayName(profile.displayName)
  }, [profile])

  if (profileLoading && !profile) {
    return (
      <p className="py-20 text-center font-display text-lg italic text-ink-500">
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
      setError(errorText(submitError, t, t('error.profileRequired')))
    } finally {
      setSaving(false)
    }
  }

  const handleReleaseNotesToggle = async (hide: boolean) => {
    setTogglingReleaseNotes(true)
    setError(null)
    try {
      await updateReleaseNotes({ hideReleaseNotes: hide })
    } catch (toggleError) {
      setError(errorText(toggleError, t, t('profile.hideReleaseNotesFailed')))
    } finally {
      setTogglingReleaseNotes(false)
    }
  }

  // Until the player picks one, the emails follow the app language.
  const emailLanguage =
    notifications.language === 'en' || notifications.language === 'fr'
      ? notifications.language
      : language

  const saveNotifications = async (update: NotificationsUpdate) => {
    setSavingNotifications(true)
    setError(null)
    try {
      // The language shown is recorded with the first setting saved.
      await updateNotifications(
        notifications.language ? update : { language: emailLanguage, ...update },
      )
    } catch (toggleError) {
      setError(errorText(toggleError, t, t('profile.notifications.failed')))
    } finally {
      setSavingNotifications(false)
    }
  }

  return (
    <Card className="mx-auto max-w-xl border-line">
      <CardHeader>
        <CardTitle className="font-display text-3xl font-bold text-ink-900">
          {t('profile.title')}
        </CardTitle>
        <CardDescription className="text-ink-500">
          {t('profile.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
          <label className="block space-y-1.5">
            <span className="text-sm font-semibold text-ink-700">
              {t('profile.displayName')}
            </span>
            <Input
              type="text"
              required
              minLength={1}
              maxLength={32}
              value={displayName}
              placeholder={t('profile.displayNamePlaceholder')}
              onChange={(event) => setDisplayName(event.target.value)}
              className="h-11"
            />
          </label>
          {error && <Alert role="alert">{error}</Alert>}
          <Button
            type="submit"
            className="w-full"
            disabled={saving || !displayName.trim()}
          >
            {saving ? t('profile.saving') : t('profile.save')}
          </Button>
        </form>
        <ToggleRow
          className="mt-6 rounded-lg border border-line/70 bg-paper-100 px-3 py-3 text-sm"
          label={t('profile.hideReleaseNotes')}
          description={t('profile.hideReleaseNotesDescription')}
          checked={profile?.hideReleaseNotes ?? false}
          disabled={!profile || togglingReleaseNotes}
          onChange={(checked) => void handleReleaseNotesToggle(checked)}
        />
        <fieldset
          className="mt-6 space-y-2 rounded-lg border border-line/70 bg-paper-100 px-3 py-3 text-sm"
          disabled={!profile || savingNotifications}
        >
          <legend className="px-1 font-semibold text-ink-700">
            {t('profile.notifications.title')}
          </legend>
          <p className="text-xs text-ink-500">{t('profile.notifications.description')}</p>
          {NOTIFICATION_TOGGLES.map(({ key, label, description }) => (
            <ToggleRow
              key={key}
              label={t(label)}
              description={t(description)}
              checked={notifications[key]}
              onChange={(checked) => void saveNotifications({ [key]: checked })}
            />
          ))}
          <p className="text-xs text-ink-500">
            {t('profile.notifications.messageLimit')}
          </p>
          <ToggleRow
            className={anyNotification ? undefined : 'opacity-60'}
            label={t('profile.notifications.reduceWhenActive')}
            description={t('profile.notifications.reduceWhenActiveDescription')}
            hint={
              anyNotification
                ? undefined
                : t('profile.notifications.reduceWhenActiveHint')
            }
            checked={notifications.reduceWhenActive}
            disabled={!anyNotification}
            onChange={(checked) => void saveNotifications({ reduceWhenActive: checked })}
          />
          <label className="block space-y-1.5 pt-1">
            <span className="block font-semibold text-ink-700">
              {t('profile.notifications.language')}
            </span>
            <select
              value={emailLanguage}
              onChange={(event) =>
                void saveNotifications({ language: event.target.value as 'en' | 'fr' })
              }
              className="h-10 w-full rounded-lg border border-line bg-paper-50 px-3 text-sm text-ink-900"
            >
              <option value="fr">{t('language.french')}</option>
              <option value="en">{t('language.english')}</option>
            </select>
          </label>
        </fieldset>
      </CardContent>
    </Card>
  )
}
