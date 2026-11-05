import { useEffect, useState } from 'react'
import { IconX } from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { useAuth } from '@/auth/useAuth'
import { deployedVersion } from '@/components/VersionBadge'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import {
  fetchReleaseNotes,
  hasPendingReleaseNotes,
  RELEASES_URL,
  type ReleaseNotesEntry,
} from '@/lib/release-notes'

/**
 * Shows the notes of the versions deployed since the player last looked, once
 * per session. Whether to open is decided locally; the notes are fetched from
 * the GitHub releases once the dialog opens. Closing records the current
 * version on the account; the checkbox opts out of future dialogs.
 */
export function ReleaseNotesDialog() {
  const { profile, updateReleaseNotes } = useAuth()
  const { t } = useLanguage()
  const current = deployedVersion()
  const [dismissed, setDismissed] = useState(false)
  const [hideFuture, setHideFuture] = useState(false)
  // null while loading; a failed fetch shows the same fallback as no notes.
  const [entries, setEntries] = useState<ReleaseNotesEntry[] | null>(null)

  const lastSeen = profile?.lastSeenVersion ?? ''
  const open =
    !!profile &&
    !profile.hideReleaseNotes &&
    !dismissed &&
    hasPendingReleaseNotes(lastSeen, current)

  useEffect(() => {
    if (!open) return
    const controller = new AbortController()
    setEntries(null)
    fetchReleaseNotes(lastSeen, current, controller.signal)
      .catch(() => [])
      .then((fetched) => {
        if (!controller.signal.aborted) setEntries(fetched)
      })
    return () => controller.abort()
  }, [open, lastSeen, current])

  if (!open) return null

  const close = () => {
    setDismissed(true)
    void updateReleaseNotes({
      lastSeenVersion: current,
      hideReleaseNotes: hideFuture,
    }).catch(() => undefined)
  }

  return (
    <Dialog.Root
      open
      onOpenChange={(open) => {
        if (!open) close()
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-ink-900/50" />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed top-1/2 left-1/2 z-50 flex max-h-[85dvh] w-[calc(100vw-1rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-line bg-paper-50 shadow-xl"
        >
          <header className="flex items-center gap-3 border-b border-line/60 px-4 py-3">
            <Dialog.Title className="font-display text-lg font-semibold text-ink-900">
              {t('releaseNotes.title')}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label={t('releaseNotes.close')}
                className="ml-auto rounded-lg p-1.5 text-ink-700 hover:bg-paper-100"
              >
                <IconX aria-hidden="true" className="size-5" />
              </button>
            </Dialog.Close>
          </header>
          <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-4 py-4">
            {!entries && (
              <p className="text-sm text-ink-500">{t('releaseNotes.loading')}</p>
            )}
            {entries?.length === 0 && (
              <p className="text-sm text-ink-700">{t('releaseNotes.unavailable')}</p>
            )}
            {entries?.map((entry) => (
              <section key={entry.version} aria-label={entry.version}>
                <h3 className="font-display text-base font-semibold text-ink-900">
                  {t('app.version', { version: entry.version })}
                  {entry.date && (
                    <span className="ml-2 text-xs font-normal text-ink-500">
                      {entry.date}
                    </span>
                  )}
                </h3>
                {entry.sections.map((section) => (
                  <div key={section.title} className="mt-2">
                    <h4 className="label-caps text-2xs text-ink-500">{section.title}</h4>
                    <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-ink-700">
                      {section.items.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                  </div>
                ))}
              </section>
            ))}
          </div>
          <footer className="space-y-3 border-t border-line/60 px-4 py-3">
            <a
              href={RELEASES_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="block text-sm font-semibold text-brass-700 underline underline-offset-4"
            >
              {t('releaseNotes.allReleases')}
            </a>
            <label className="flex items-center gap-2 text-sm text-ink-700">
              <input
                type="checkbox"
                checked={hideFuture}
                onChange={(event) => setHideFuture(event.target.checked)}
                className="size-4 accent-brass-700"
              />
              {t('releaseNotes.hideFuture')}
            </label>
            <Button type="button" className="w-full" onClick={close}>
              {t('releaseNotes.close')}
            </Button>
          </footer>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
