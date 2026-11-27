import { useCallback, useEffect, useRef, useState } from 'react'
import { IconBrandDiscord, IconX } from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useLanguage } from '@/i18n/LanguageContext'
import { ApiError, type TokenProvider } from '@/lib/api'
import { errorText } from '@/lib/api-error'
import {
  deleteDiscordWebhook,
  fetchDiscordWebhook,
  previewDiscordTemplate,
  saveDiscordWebhook,
  testDiscordWebhook,
} from '@/lib/discord-api'
import type {
  DiscordEvent,
  DiscordLanguage,
  DiscordPreview,
  DiscordTemplates,
  DiscordWebhookStatus,
} from '@/types'

/** Delay after the last keystroke before the preview is requested. */
export const PREVIEW_DEBOUNCE_MS = 400

/** What the host does to the webhook of the game. */
export interface DiscordSettingsProps {
  gameId: string
  tokenProvider: TokenProvider
  /** Called with whether the game announces to Discord after each change. */
  onActiveChange?: (active: boolean) => void
}

const EVENTS: DiscordEvent[] = ['newTurn', 'publicMessage']

const TEXTAREA_CLASS =
  'mt-1 w-full rounded-lg border border-line-strong bg-paper-50 px-3 py-2 font-mono text-sm outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20'

type Feedback = { kind: 'success' | 'error'; text: string }

function isActive(status: DiscordWebhookStatus): boolean {
  return status.configured && !status.disabled
}

/** The server words a rejected template with the unknown variable's name. */
function discordErrorText(
  failure: unknown,
  t: ReturnType<typeof useLanguage>['t'],
): string {
  if (failure instanceof ApiError && failure.code === 'invalid_template') {
    const variable = (failure.payload as { variable?: unknown } | null)?.variable
    if (typeof variable === 'string' && variable !== '') {
      return t('discord.invalidTemplate', { variable })
    }
  }
  return errorText(failure, t, t('error.requestFailed', { status: 0 }))
}

/**
 * Header button opening the Discord settings of a game, for its host. The
 * webhook URL is a write credential on a channel: it is typed here but never
 * shown back.
 */
export function DiscordSettingsDialog({
  open,
  onOpenChange,
  ...props
}: DiscordSettingsProps & { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useLanguage()
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      {open && (
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-50 bg-ink-900/50" />
          <Dialog.Content
            aria-describedby="discord-intro"
            className="fixed top-1/2 left-1/2 z-50 flex max-h-[90dvh] w-[calc(100vw-1rem)] max-w-2xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-line bg-paper-50 shadow-xl"
          >
            <header className="flex items-center gap-3 border-b border-line/60 px-4 py-3">
              <IconBrandDiscord aria-hidden="true" className="size-5 text-ink-500" />
              <Dialog.Title className="font-display text-lg font-semibold text-ink-900">
                {t('discord.title')}
              </Dialog.Title>
              <Dialog.Close asChild>
                <button
                  type="button"
                  aria-label={t('discord.close')}
                  className="ml-auto rounded-lg p-1.5 text-ink-700 hover:bg-paper-100"
                >
                  <IconX aria-hidden="true" className="size-5" />
                </button>
              </Dialog.Close>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
              <DiscordSettings {...props} />
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      )}
    </Dialog.Root>
  )
}

export function DiscordSettings({
  gameId,
  tokenProvider,
  onActiveChange,
}: DiscordSettingsProps) {
  const { t, language: appLanguage } = useLanguage()
  const [status, setStatus] = useState<DiscordWebhookStatus | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [url, setUrl] = useState('')
  const [language, setLanguage] = useState<DiscordLanguage>(appLanguage)
  const [templates, setTemplates] = useState<DiscordTemplates>({
    newTurn: '',
    publicMessage: '',
  })
  const [busy, setBusy] = useState<'save' | 'test' | 'remove' | null>(null)
  const [confirmingRemove, setConfirmingRemove] = useState(false)
  const [feedback, setFeedback] = useState<Feedback | null>(null)

  // The token provider of a caller may change identity between renders.
  const providerRef = useRef(tokenProvider)
  providerRef.current = tokenProvider
  const activeChangeRef = useRef(onActiveChange)
  activeChangeRef.current = onActiveChange

  const apply = useCallback((next: DiscordWebhookStatus) => {
    setStatus(next)
    setUrl('')
    if (next.language) setLanguage(next.language)
    setTemplates(next.templates)
    activeChangeRef.current?.(isActive(next))
  }, [])

  useEffect(() => {
    let cancelled = false
    fetchDiscordWebhook(providerRef.current, gameId).then(
      (next) => {
        if (!cancelled) apply(next)
      },
      (failure: unknown) => {
        if (!cancelled) setLoadError(discordErrorText(failure, t))
      },
    )
    return () => {
      cancelled = true
    }
    // The load happens once per game; the text of an error follows the
    // language at the time it happened.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameId, apply])

  const run = async (
    action: 'save' | 'test' | 'remove',
    call: () => Promise<DiscordWebhookStatus>,
    success: string,
  ) => {
    setBusy(action)
    setFeedback(null)
    try {
      apply(await call())
      setFeedback({ kind: 'success', text: success })
    } catch (failure) {
      setFeedback({ kind: 'error', text: discordErrorText(failure, t) })
      // Discord refusing the webhook disables it: read the state back.
      if (failure instanceof ApiError && failure.code === 'webhook_gone') {
        void fetchDiscordWebhook(providerRef.current, gameId).then(apply, () => undefined)
      }
    } finally {
      setBusy(null)
      setConfirmingRemove(false)
    }
  }

  // The test goes to the webhook typed in the form when there is one, else to
  // the stored one. It stores nothing and leaves the form as it is.
  const sendTest = async () => {
    const typed = url.trim()
    setBusy('test')
    setFeedback(null)
    try {
      await testDiscordWebhook(
        tokenProvider,
        gameId,
        typed !== '' ? { url: typed, language } : undefined,
      )
      setFeedback({ kind: 'success', text: t('discord.tested') })
    } catch (failure) {
      const refusedForm =
        typed !== '' && failure instanceof ApiError && failure.code === 'webhook_gone'
      setFeedback({
        kind: 'error',
        text: refusedForm ? t('discord.testRefused') : discordErrorText(failure, t),
      })
      // Discord refusing the stored webhook disables it: read the state back,
      // without touching what the host is typing.
      if (typed === '' && failure instanceof ApiError && failure.code === 'webhook_gone') {
        void fetchDiscordWebhook(providerRef.current, gameId).then((next) => {
          setStatus(next)
          activeChangeRef.current?.(isActive(next))
        }, () => undefined)
      }
    } finally {
      setBusy(null)
    }
  }

  const save = () => {
    const trimmed = url.trim()
    if (!status?.configured && trimmed === '') {
      setFeedback({ kind: 'error', text: t('discord.urlRequired') })
      return
    }
    void run(
      'save',
      () =>
        saveDiscordWebhook(tokenProvider, gameId, {
          // Left out, the stored URL stays.
          ...(trimmed !== '' ? { url: trimmed } : {}),
          language,
          templates,
        }),
      t('discord.saved'),
    )
  }

  if (loadError) return <Alert role="alert">{loadError}</Alert>
  if (!status) {
    return <p className="text-sm text-ink-500">{t('discord.loading')}</p>
  }

  const working = busy !== null

  return (
    <div className="space-y-4">
      <p id="discord-intro" className="text-sm text-ink-700">
        {t('discord.intro')}
      </p>
      <p className="text-xs text-ink-500">{t('discord.privacy')}</p>

      {status.disabled && (
        <Alert role="alert" tone="error">
          {t('discord.disabled')}
        </Alert>
      )}
      <p className="text-sm font-semibold text-ink-900">
        {status.configured ? t('discord.configured') : t('discord.notConfigured')}
      </p>

      <div>
        <label htmlFor="discord-url" className="label-caps text-2xs text-ink-500">
          {status.configured ? t('discord.urlLabelReplace') : t('discord.urlLabel')}
        </label>
        <Input
          id="discord-url"
          type="password"
          autoComplete="off"
          spellCheck={false}
          value={url}
          disabled={working}
          onChange={(event) => setUrl(event.target.value)}
          className="mt-1"
        />
        <p className="mt-1 text-xs text-ink-500">{t('discord.urlHint')}</p>
      </div>

      <div>
        <label htmlFor="discord-language" className="label-caps text-2xs text-ink-500">
          {t('discord.languageLabel')}
        </label>
        <select
          id="discord-language"
          value={language}
          disabled={working}
          onChange={(event) => setLanguage(event.target.value as DiscordLanguage)}
          className="mt-1 h-10 w-full rounded-lg border border-line-strong bg-paper-50 px-3 text-sm"
        >
          <option value="fr">Français</option>
          <option value="en">English</option>
        </select>
      </div>

      <section className="space-y-3">
        <h3 className="label-caps text-2xs text-ink-500">
          {t('discord.templatesTitle')}
        </h3>
        <p className="text-xs text-ink-500">{t('discord.templatesHelp')}</p>
        {EVENTS.map((event) => (
          <TemplateEditor
            key={event}
            event={event}
            value={templates[event]}
            placeholder={status.defaults[language]?.[event] ?? ''}
            language={language}
            gameId={gameId}
            tokenProvider={tokenProvider}
            disabled={working}
            onChange={(value) =>
              setTemplates((current) => ({ ...current, [event]: value }))
            }
          />
        ))}
      </section>

      {feedback && (
        <p
          role={feedback.kind === 'error' ? 'alert' : 'status'}
          className={
            feedback.kind === 'error'
              ? 'rounded-md border border-vermilion-700/30 bg-vermilion-100 px-3 py-2 text-sm text-vermilion-700'
              : 'text-sm text-success-700'
          }
        >
          {feedback.text}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" disabled={working} onClick={save}>
          {busy === 'save' ? t('discord.saving') : t('discord.save')}
        </Button>
        {(status.configured || url.trim() !== '') && (
          <Button
            type="button"
            variant="outline"
            disabled={working}
            onClick={() => void sendTest()}
          >
            {busy === 'test' ? t('discord.testing') : t('discord.test')}
          </Button>
        )}
        {status.configured && (
          <>
            {confirmingRemove ? (
              <span className="flex flex-wrap items-center gap-2 text-sm text-ink-700">
                {t('discord.removeConfirm')}
                <Button
                  type="button"
                  variant="outline"
                  disabled={working}
                  onClick={() =>
                    void run(
                      'remove',
                      () => deleteDiscordWebhook(tokenProvider, gameId),
                      t('discord.removed'),
                    )
                  }
                >
                  {t('discord.removeYes')}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  disabled={working}
                  onClick={() => setConfirmingRemove(false)}
                >
                  {t('discord.cancel')}
                </Button>
              </span>
            ) : (
              <Button
                type="button"
                variant="ghost"
                disabled={working}
                onClick={() => setConfirmingRemove(true)}
              >
                {t('discord.remove')}
              </Button>
            )}
          </>
        )}
      </div>
    </div>
  )
}

type PreviewState =
  | { kind: 'idle' }
  | { kind: 'ok'; preview: DiscordPreview }
  | { kind: 'error'; text: string }

function TemplateEditor({
  event,
  value,
  placeholder,
  language,
  gameId,
  tokenProvider,
  disabled,
  onChange,
}: {
  event: DiscordEvent
  value: string
  placeholder: string
  language: DiscordLanguage
  gameId: string
  tokenProvider: TokenProvider
  disabled: boolean
  onChange: (value: string) => void
}) {
  const { t } = useLanguage()
  const [preview, setPreview] = useState<PreviewState>({ kind: 'idle' })
  const providerRef = useRef(tokenProvider)
  providerRef.current = tokenProvider
  // The answer of an older request never replaces a newer one.
  const requestRef = useRef(0)
  const textId = `discord-template-${event}`

  useEffect(() => {
    const request = ++requestRef.current
    const timer = setTimeout(() => {
      previewDiscordTemplate(providerRef.current, gameId, event, language, value).then(
        (next) => {
          if (request === requestRef.current) setPreview({ kind: 'ok', preview: next })
        },
        (failure: unknown) => {
          if (request !== requestRef.current) return
          setPreview({ kind: 'error', text: discordErrorText(failure, t) })
        },
      )
    }, PREVIEW_DEBOUNCE_MS)
    return () => clearTimeout(timer)
    // A language change of the interface only rewords errors: it does not
    // refresh the preview.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [event, gameId, language, value])

  return (
    <div className="space-y-1">
      <label htmlFor={textId} className="block text-sm font-semibold text-ink-900">
        {t(`discord.event.${event}`)}
      </label>
      <p className="text-xs text-ink-500">{t(`discord.eventHelp.${event}`)}</p>
      <textarea
        id={textId}
        rows={3}
        maxLength={2000}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={(change) => onChange(change.target.value)}
        className={TEXTAREA_CLASS}
      />
      {value !== '' && (
        <button
          type="button"
          disabled={disabled}
          onClick={() => onChange('')}
          className="text-xs font-semibold text-brass-700 underline underline-offset-4"
        >
          {t('discord.useDefault')}
        </button>
      )}
      <div
        aria-label={`${t('discord.previewLabel')} ${t(`discord.event.${event}`)}`}
        className="rounded-md border border-line/60 bg-paper-100 px-3 py-2 text-sm text-ink-900"
      >
        {preview.kind === 'ok' && (
          <>
            <p className="whitespace-pre-wrap break-words">{preview.preview.content}</p>
            <p className="mt-1 text-xs text-ink-500">
              {t('discord.previewLength', { length: preview.preview.length })}
              {' · '}
              {preview.preview.mentionEveryone
                ? t('discord.previewEveryone')
                : preview.preview.mentionRoles.length > 0
                  ? t('discord.previewRoles', {
                      count: preview.preview.mentionRoles.length,
                    })
                  : t('discord.previewNoMention')}
            </p>
          </>
        )}
        {preview.kind === 'error' && (
          <p className="text-xs text-vermilion-700">
            {t('discord.previewFailed', { reason: preview.text })}
          </p>
        )}
      </div>
    </div>
  )
}
