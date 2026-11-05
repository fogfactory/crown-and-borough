import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { IconBug, IconX } from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { useAuth } from '@/auth/useAuth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useLanguage } from '@/i18n/LanguageContext'
import {
  fetchBugReportsEnabled,
  fetchGameView,
  sendBugReport,
} from '@/lib/bug-report-api'
import {
  captureBugContext,
  contextJson,
  currentGameId,
  githubIssueUrl,
  githubSummary,
  maskText,
} from '@/lib/bug-report'
import { cn } from '@/lib/utils'

type Status =
  | { kind: 'idle' }
  | { kind: 'confirming' }
  | { kind: 'sending' }
  | { kind: 'sent'; url: string }
  | { kind: 'copied'; copied: boolean }
  | { kind: 'error' }

// Mirror bugTitleMaxRunes and bugDescriptionRunes in internal/api/bugs.go.
const TITLE_MAX = 120
const DESCRIPTION_MAX = 4000

/**
 * Header button opening the bug report dialog. The report is either filed
 * directly (public GitHub issue, after an explicit confirmation) or pre-filled
 * on GitHub with the captured context copied to the clipboard.
 */
export function BugReportButton({ className }: { className?: string }) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn('size-11 text-ink-500 lg:size-8', className)}
          aria-label={t('bugReport.open')}
          title={t('bugReport.open')}
        >
          <IconBug aria-hidden="true" className="size-5" />
        </Button>
      </Dialog.Trigger>
      {open && <BugReportContent onClose={() => setOpen(false)} />}
    </Dialog.Root>
  )
}

function BugReportContent({ onClose }: { onClose: () => void }) {
  const { getIdToken } = useAuth()
  const { t, language } = useLanguage()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [enabled, setEnabled] = useState<boolean | null>(null)
  const [status, setStatus] = useState<Status>({ kind: 'idle' })
  // Captured once when the dialog opens: what the player sees is what is sent.
  const [context] = useState(() => captureBugContext(language))
  const gameId = currentGameId(useLocation().pathname)
  // The player's own game view, fetched like the game page does, for the
  // preview only: on "Send" the server stores its own copy privately and never
  // trusts one from the browser.
  const [game, setGame] = useState<unknown>(null)
  const json = useMemo(() => contextJson(context, game), [context, game])

  useEffect(() => {
    if (!gameId) return
    let cancelled = false
    void fetchGameView({ getIdToken }, gameId).then((view) => {
      if (!cancelled) setGame(view)
    })
    return () => {
      cancelled = true
    }
  }, [gameId, getIdToken])

  useEffect(() => {
    let cancelled = false
    void fetchBugReportsEnabled({ getIdToken }).then((value) => {
      if (!cancelled) setEnabled(value)
    })
    return () => {
      cancelled = true
    }
  }, [getIdToken])

  const filled = title.trim() !== '' && description.trim() !== ''
  const busy = status.kind === 'sending'
  const canSend = filled && enabled === true && !busy

  const send = async () => {
    setStatus({ kind: 'sending' })
    try {
      const url = await sendBugReport(
        { getIdToken },
        { title: title.trim(), description: description.trim(), gameId, context },
      )
      setStatus({ kind: 'sent', url })
    } catch {
      setStatus({ kind: 'error' })
    }
  }

  const fillOnGithub = async () => {
    // navigator.clipboard is missing outside secure contexts.
    const copied = await Promise.resolve()
      // The issue is public: copy the capture alone, never the game view.
      .then(() => navigator.clipboard.writeText(JSON.stringify(context, null, 2)))
      .then(() => true)
      .catch(() => false)
    const url = githubIssueUrl(
      maskText(title.trim()),
      githubSummary(description.trim(), context, t('bugReport.pasteHint')),
    )
    window.open(url, '_blank', 'noopener,noreferrer')
    setStatus({ kind: 'copied', copied })
  }

  const finished = status.kind === 'sent'

  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-50 bg-ink-900/50" />
      <Dialog.Content
        aria-describedby="bug-report-intro"
        className="fixed top-1/2 left-1/2 z-50 flex max-h-[90dvh] w-[calc(100vw-1rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-line bg-paper-50 shadow-xl"
      >
        <header className="flex items-center gap-3 border-b border-line/60 px-4 py-3">
          <Dialog.Title className="font-display text-lg font-semibold text-ink-900">
            {t('bugReport.title')}
          </Dialog.Title>
          <Dialog.Close asChild>
            <button
              type="button"
              aria-label={t('bugReport.close')}
              className="ml-auto rounded-lg p-1.5 text-ink-700 hover:bg-paper-100"
            >
              <IconX aria-hidden="true" className="size-5" />
            </button>
          </Dialog.Close>
        </header>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4">
          <p id="bug-report-intro" className="text-sm text-ink-700">
            {t('bugReport.intro')}
          </p>

          {finished ? (
            <p role="status" className="text-sm text-ink-900">
              {t('bugReport.sent')}{' '}
              <a
                href={status.url}
                target="_blank"
                rel="noopener noreferrer"
                className="font-semibold text-brass-700 underline underline-offset-4"
              >
                {t('bugReport.viewIssue')}
              </a>
            </p>
          ) : (
            <>
              <div>
                <label
                  htmlFor="bug-report-title"
                  className="label-caps text-2xs text-ink-500"
                >
                  {t('bugReport.summaryLabel')}
                </label>
                <Input
                  id="bug-report-title"
                  value={title}
                  maxLength={TITLE_MAX}
                  disabled={busy}
                  onChange={(event) => setTitle(event.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <label
                  htmlFor="bug-report-description"
                  className="label-caps text-2xs text-ink-500"
                >
                  {t('bugReport.descriptionLabel')}
                </label>
                <textarea
                  id="bug-report-description"
                  value={description}
                  maxLength={DESCRIPTION_MAX}
                  disabled={busy}
                  rows={5}
                  placeholder={t('bugReport.descriptionPlaceholder')}
                  onChange={(event) => setDescription(event.target.value)}
                  className="mt-1 w-full rounded-lg border border-line-strong bg-paper-50 px-3 py-2 text-sm outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20"
                />
              </div>
              <details className="rounded-lg border border-line/60 bg-paper-100 px-3 py-2">
                <summary className="cursor-pointer text-sm font-semibold text-ink-700">
                  {t('bugReport.contextSummary')}
                </summary>
                <p className="mt-2 text-xs text-ink-500">{t('bugReport.contextHelp')}</p>
                <pre
                  aria-label={t('bugReport.contextPreview')}
                  className="mt-2 max-h-48 overflow-auto rounded-md bg-paper-50 p-2 font-mono text-2xs text-ink-700"
                >
                  {json}
                </pre>
              </details>
              {enabled === false && (
                <p className="text-xs text-ink-500">{t('bugReport.sendDisabled')}</p>
              )}
              {status.kind === 'confirming' && (
                <p
                  role="alert"
                  className="rounded-md border border-brass-700/40 bg-brass-100 px-3 py-2 text-sm text-ink-900"
                >
                  {t('bugReport.confirm')}
                </p>
              )}
              {status.kind === 'error' && (
                <p
                  role="alert"
                  className="rounded-md border border-vermilion-700/30 bg-vermilion-100 px-3 py-2 text-sm text-vermilion-700"
                >
                  {t('bugReport.error')}
                </p>
              )}
              {status.kind === 'copied' && (
                <p role="status" className="text-sm text-ink-900">
                  {status.copied ? t('bugReport.copied') : t('bugReport.copyFailed')}
                </p>
              )}
              <p className="text-xs text-ink-500">{t('bugReport.publicNotice')}</p>
            </>
          )}
        </div>

        <footer className="flex flex-wrap justify-end gap-2 border-t border-line/60 px-4 py-3">
          {finished ? (
            <Button type="button" onClick={onClose}>
              {t('bugReport.close')}
            </Button>
          ) : status.kind === 'confirming' ? (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => setStatus({ kind: 'idle' })}
              >
                {t('bugReport.back')}
              </Button>
              <Button type="button" onClick={() => void send()}>
                {t('bugReport.confirmSend')}
              </Button>
            </>
          ) : (
            <>
              <Button
                type="button"
                variant="outline"
                disabled={!filled || busy}
                onClick={() => void fillOnGithub()}
              >
                {t('bugReport.fillOnGithub')}
              </Button>
              <Button
                type="button"
                disabled={!canSend}
                title={enabled === false ? t('bugReport.sendDisabled') : undefined}
                onClick={() => setStatus({ kind: 'confirming' })}
              >
                {busy ? t('bugReport.sending') : t('bugReport.send')}
              </Button>
            </>
          )}
        </footer>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
