import { useState } from 'react'
import { Popover } from 'radix-ui'
import { IconAdjustmentsHorizontal } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useLanguage } from '@/i18n/LanguageContext'

const MIN_PLAYERS = 2
const MAX_PLAYERS = 16
const PLAYER_COUNT_OPTIONS = Array.from(
  { length: MAX_PLAYERS - MIN_PLAYERS + 1 },
  (_, index) => MIN_PLAYERS + index,
)

interface GameSetupMenuProps {
  playerCount: number
  years: number
  seed: string
  creating: boolean
  createError: string | null
  canCreate: boolean
  onPlayerCountChange: (count: number) => void
  onYearsChange: (years: number) => void
  onSeedChange: (seed: string) => void
  onCreate: () => Promise<void> | void
}

/**
 * Collapsible game creation controls used in the compact header on phones.
 */
export function GameSetupMenu({
  playerCount,
  years,
  seed,
  creating,
  createError,
  canCreate,
  onPlayerCountChange,
  onYearsChange,
  onSeedChange,
  onCreate,
}: GameSetupMenuProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)

  const handleCreate = async () => {
    await onCreate()
    setOpen(false)
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          type="button"
          aria-label={t('app.gameSetup')}
          title={t('app.gameSetup')}
          className="flex size-9 items-center justify-center rounded-lg border border-line bg-paper-50 text-ink-700 transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
        >
          <IconAdjustmentsHorizontal aria-hidden="true" className="size-4" />
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={8}
          className="z-50 w-64 rounded-xl border border-line bg-paper-50 p-4 shadow-panel-strong"
        >
          <p className="mb-3 text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500">
            {t('app.gameSetup')}
          </p>
          <div className="space-y-3">
            <div>
              <label
                htmlFor="setup-player-count"
                className="block text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500"
              >
                {t('app.players')}
              </label>
              <Select
                value={String(playerCount)}
                onValueChange={(value) => onPlayerCountChange(Number(value))}
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
              <label
                htmlFor="setup-game-seed"
                className="block text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500"
              >
                {t('app.seed')}
              </label>
              <input
                id="setup-game-seed"
                type="text"
                value={seed}
                onChange={(event) => onSeedChange(event.target.value)}
                placeholder={t('app.seedPlaceholder')}
                className="mt-1 h-8 w-full rounded-lg border border-line-strong bg-paper-50 px-2.5 text-sm text-ink-900 outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20"
              />
            </div>
            <div>
              <label
                htmlFor="setup-game-years"
                className="block text-2xs font-semibold uppercase tracking-[0.2em] text-ink-500"
              >
                {t('home.gameYears')}
              </label>
              <input
                id="setup-game-years"
                type="number"
                min={1}
                max={50}
                value={years}
                onChange={(event) => onYearsChange(Number(event.target.value))}
                className="mt-1 h-8 w-full rounded-lg border border-line-strong bg-paper-50 px-2.5 text-sm text-ink-900 outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20"
              />
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full"
              disabled={creating || !canCreate}
              title={t('app.newGameTitle')}
              onClick={() => void handleCreate()}
            >
              {creating ? t('app.creating') : t('app.newGame')}
            </Button>
            {createError && (
              <p
                role="alert"
                className="rounded-md border border-vermilion-700/30 bg-vermilion-100 px-2 py-1 text-xs text-vermilion-700"
              >
                {createError}
              </p>
            )}
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
