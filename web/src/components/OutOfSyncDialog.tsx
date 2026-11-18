import { useEffect, useState } from 'react'
import { Dialog } from 'radix-ui'

import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'

/** Time the screen may lag behind the server before the player is told. */
export const OUT_OF_SYNC_GRACE_MS = 8000

export interface OutOfSyncDialogProps {
  /** Revision announced by the live game document, when known. */
  serverRevision: number | undefined
  /** Revision of the state the screen currently displays. */
  displayedRevision: number | undefined
  /** The live connection is broken: updates can no longer arrive. */
  connectionLost?: boolean
  onReload?: () => void
}

/**
 * Invites the player to reload when the displayed game no longer follows the
 * server (the live document announces a newer revision than the state shown,
 * or the live connection failed). Missing updates are otherwise silent: orders
 * intentions, for instance, simply stop matching the game. A short grace delay
 * absorbs the normal gap between the two live documents.
 */
export function OutOfSyncDialog({
  serverRevision,
  displayedRevision,
  connectionLost = false,
  onReload = () => window.location.reload(),
}: OutOfSyncDialogProps) {
  const { t } = useLanguage()
  const behind =
    serverRevision !== undefined &&
    displayedRevision !== undefined &&
    serverRevision > displayedRevision
  const stale = behind || connectionLost
  const [confirmed, setConfirmed] = useState(false)
  // Revision the player chose to ignore; a further desync asks again.
  const revisionKey = serverRevision ?? -1
  const [dismissedAt, setDismissedAt] = useState<number | null>(null)

  useEffect(() => {
    if (!stale) {
      setConfirmed(false)
      return
    }
    const timer = window.setTimeout(() => setConfirmed(true), OUT_OF_SYNC_GRACE_MS)
    return () => window.clearTimeout(timer)
  }, [stale, serverRevision])

  const open = stale && confirmed && dismissedAt !== revisionKey
  if (!open) return null

  return (
    <Dialog.Root
      open
      onOpenChange={(next) => {
        if (!next) setDismissedAt(revisionKey)
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-ink-900/50" />
        <Dialog.Content
          aria-describedby="out-of-sync-description"
          className="fixed top-1/2 left-1/2 z-50 w-[calc(100vw-1rem)] max-w-md -translate-x-1/2 -translate-y-1/2 space-y-3 rounded-xl border border-line bg-paper-50 p-4 shadow-xl"
        >
          <Dialog.Title className="font-display text-lg font-semibold text-ink-900">
            {t('online.outOfSync.title')}
          </Dialog.Title>
          <p id="out-of-sync-description" className="text-sm text-ink-700">
            {t('online.outOfSync.description')}
          </p>
          <div className="flex flex-col gap-2 sm:flex-row-reverse">
            <Button type="button" className="sm:flex-1" onClick={onReload}>
              {t('online.outOfSync.reload')}
            </Button>
            <Dialog.Close asChild>
              <Button type="button" variant="secondary" className="sm:flex-1">
                {t('online.outOfSync.later')}
              </Button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
