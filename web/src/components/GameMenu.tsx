import { useState, type ReactNode } from 'react'
import { Popover } from 'radix-ui'
import { IconChevronDown, IconMenu2 } from '@tabler/icons-react'

import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { VersionBadge } from '@/components/VersionBadge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useLanguage } from '@/i18n/LanguageContext'
import { useIsDesktop } from '@/lib/use-media-query'

const MIN_PLAYERS = 2
const MAX_PLAYERS = 16
const PLAYER_COUNT_OPTIONS = Array.from(
  { length: MAX_PLAYERS - MIN_PLAYERS + 1 },
  (_, index) => MIN_PLAYERS + index,
)

/** Game creation controls (hotseat): players, seed and years. */
export interface GameMenuCreation {
  playerCount: number
  years: number
  seed: string
  creating: boolean
  error: string | null
  canCreate: boolean
  onPlayerCountChange: (count: number) => void
  onYearsChange: (years: number) => void
  onSeedChange: (seed: string) => void
  onCreate: () => Promise<void> | void
}

/** Forced resolution of the current turn. */
export interface GameMenuResolution {
  disabled: boolean
  /** Ask for a confirmation before resolving (online games). */
  confirm?: boolean
  onResolve: () => Promise<void> | void
}

interface GameMenuProps {
  creation?: GameMenuCreation
  resolution?: GameMenuResolution
  /** Show the language switcher (when the page header does not). */
  language?: boolean
  /** Bar items moved here when the bar is narrow (phones, tablets). */
  compact?: ReactNode
}

const SECTION_TITLE =
  'mb-2 text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500'
const FIELD_LABEL = 'block text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500'
const FIELD_INPUT =
  'mt-1 h-8 w-full rounded-md border border-line-strong bg-paper-50 px-2.5 text-sm text-ink-900 outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20'

/**
 * "Game" menu of the top bar (charte-graphique.md § Disposition sur
 * ordinateur): game creation, forced resolution and language, so the bar
 * holds on one line.
 */
export function GameMenu({
  creation,
  resolution,
  language = false,
  compact,
}: GameMenuProps) {
  const { t } = useLanguage()
  const desktop = useIsDesktop()
  const [open, setOpen] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) setConfirming(false)
  }

  const handleCreate = async () => {
    await creation?.onCreate()
    setOpen(false)
  }

  const handleResolve = async () => {
    if (resolution?.confirm && !confirming) {
      setConfirming(true)
      return
    }
    setConfirming(false)
    setOpen(false)
    await resolution?.onResolve()
  }

  return (
    <Popover.Root open={open} onOpenChange={handleOpenChange}>
      <Popover.Trigger asChild>
        {desktop ? (
          <button
            type="button"
            className="inline-flex h-8 shrink-0 items-center gap-1 rounded-md border border-line-strong bg-paper-50 px-2.5 text-sm font-semibold text-ink-700 transition hover:bg-paper-100 hover:text-ink-900"
          >
            {t('app.gameMenu')}
            <IconChevronDown aria-hidden="true" className="size-3.5" />
          </button>
        ) : (
          <button
            type="button"
            aria-label={t('app.gameMenu')}
            title={t('app.gameMenu')}
            className="inline-flex size-11 shrink-0 items-center justify-center rounded-md border border-line-strong bg-paper-50 text-ink-700 transition hover:bg-paper-100 hover:text-ink-900"
          >
            <IconMenu2 aria-hidden="true" className="size-5" />
          </button>
        )}
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          aria-label={t('app.gameMenu')}
          className="z-50 w-64 space-y-4 rounded-md border border-line bg-paper-50 p-4 shadow-panel"
        >
          {compact && !desktop && <section className="space-y-3">{compact}</section>}
          {resolution && (
            <section>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full"
                disabled={resolution.disabled}
                title={t('app.resolveTitle')}
                onClick={() => void handleResolve()}
              >
                {confirming ? t('online.forceResolve') : t('app.resolve')}
              </Button>
              {confirming && (
                <div className="mt-2 space-y-2 rounded-md border border-warning-700/40 bg-warning-100 p-2 text-xs text-warning-700">
                  <p>{t('online.forceResolveConfirm')}</p>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => setConfirming(false)}
                  >
                    {t('online.cancel')}
                  </Button>
                </div>
              )}
            </section>
          )}
          {creation && (
            <section className="space-y-3">
              <p className={SECTION_TITLE}>{t('app.gameSetup')}</p>
              <div>
                <label htmlFor="setup-player-count" className={FIELD_LABEL}>
                  {t('app.players')}
                </label>
                <Select
                  value={String(creation.playerCount)}
                  onValueChange={(value) => creation.onPlayerCountChange(Number(value))}
                >
                  <SelectTrigger
                    id="setup-player-count"
                    className="mt-1 w-full border-line bg-paper-50 text-ink-900"
                  >
                    <SelectValue placeholder={t('app.players')} />
                  </SelectTrigger>
                  <SelectContent>
                    {PLAYER_COUNT_OPTIONS.map((count) => (
                      <SelectItem key={count} value={String(count)}>
                        {count}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label htmlFor="setup-game-seed" className={FIELD_LABEL}>
                  {t('app.seed')}
                </label>
                <input
                  id="setup-game-seed"
                  type="text"
                  value={creation.seed}
                  onChange={(event) => creation.onSeedChange(event.target.value)}
                  placeholder={t('app.seedPlaceholder')}
                  className={FIELD_INPUT}
                />
              </div>
              <div>
                <label htmlFor="setup-game-years" className={FIELD_LABEL}>
                  {t('home.gameYears')}
                </label>
                <input
                  id="setup-game-years"
                  type="number"
                  min={1}
                  max={50}
                  value={creation.years}
                  onChange={(event) => creation.onYearsChange(Number(event.target.value))}
                  className={FIELD_INPUT}
                />
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full"
                disabled={creation.creating || !creation.canCreate}
                title={t('app.newGameTitle')}
                onClick={() => void handleCreate()}
              >
                {creation.creating ? t('app.creating') : t('app.newGame')}
              </Button>
              {creation.error && (
                <p
                  role="alert"
                  className="rounded-md border border-vermilion-700/30 bg-vermilion-100 px-2 py-1 text-xs text-vermilion-700"
                >
                  {creation.error}
                </p>
              )}
            </section>
          )}
          {language && (
            <section className="flex items-center justify-between gap-2">
              <span className="text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500">
                {t('language.label')}
              </span>
              <LanguageSwitcher />
            </section>
          )}
          <VersionBadge />
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
