import { useMemo, useState } from 'react'
import { IconCalculator, IconX } from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import {
  emptyScenario,
  useVictorySimulation,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import type {
  Noble,
  Player,
  PlayerId,
  VictoryProjectionStatus,
  VictoryReading,
  VictoryScenario,
} from '@/types'

type Kind = 'marriage' | 'marriageEnd' | 'death' | 'claim'

const KIND_KEYS: Record<Kind, MessageKey> = {
  marriage: 'sim.kindMarriage',
  marriageEnd: 'sim.kindMarriageEnd',
  death: 'sim.kindDeath',
  claim: 'sim.kindClaim',
}

const STATUS_KEYS: Record<VictoryProjectionStatus, MessageKey> = {
  major: 'score.outcomeMajor',
  minor: 'score.outcomeMinor',
  failure: 'score.outcomeFailure',
  ongoing: 'sim.statusOngoing',
}

const STATUS_STYLES: Record<VictoryProjectionStatus, string> = {
  major: 'border-[#b8860b] bg-[#f8e8ae] text-[#6b4e0a]',
  minor: 'border-[#8a929b] bg-[#e4e7ea] text-[#3f464d]',
  failure: 'border-[#a84632]/50 bg-[#f4d9d2] text-[#7a2b1c]',
  ongoing: 'border-[#b7a786] bg-[#f3ead9] text-[#594b3c]',
}

/**
 * Dialog opened from the scoreboard: the player stacks hypotheses (marriage,
 * end of marriage, death, honoured claim) and reads the projected score and
 * victory status computed by the server, without committing anything.
 */
export function VictorySimulatorDialog({
  players,
  nobles,
  playerId,
  request,
}: {
  players: Player[]
  nobles: Noble[]
  playerId: PlayerId
  request: VictorySimulationRequest | null
}) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  const [scenario, setScenario] = useState<VictoryScenario>(emptyScenario)
  const { simulation, loading, failed } = useVictorySimulation(scenario, request, open)

  const noblesByCode = useMemo(() => new Map(nobles.map((n) => [n.code, n])), [nobles])
  const nobleName = (code: string) => noblesByCode.get(code)?.name || code
  const playerName = (id: string) => players.find((p) => p.id === id)?.name || id
  const empty =
    scenario.marriages.length +
      scenario.marriageEnds.length +
      scenario.deaths.length +
      scenario.claims.length ===
    0

  const lines: Array<{ id: string; text: string; remove: () => void }> = [
    ...scenario.marriageEnds.map((m, index) => ({
      id: `end-${index}`,
      text: t('sim.marriageEnd', { noble: nobleName(m.noble), spouse: nobleName(m.spouse) }),
      remove: () =>
        setScenario((s) => ({ ...s, marriageEnds: s.marriageEnds.filter((_, i) => i !== index) })),
    })),
    ...scenario.deaths.map((code, index) => ({
      id: `death-${index}`,
      text: t('sim.death', { noble: nobleName(code) }),
      remove: () => setScenario((s) => ({ ...s, deaths: s.deaths.filter((_, i) => i !== index) })),
    })),
    ...scenario.marriages.map((m, index) => ({
      id: `marriage-${index}`,
      text: t('sim.marriage', { noble: nobleName(m.noble), spouse: nobleName(m.spouse) }),
      remove: () =>
        setScenario((s) => ({ ...s, marriages: s.marriages.filter((_, i) => i !== index) })),
    })),
    ...scenario.claims.map((c, index) => ({
      id: `claim-${index}`,
      text: t('sim.claim', { heir: nobleName(c.heir), target: nobleName(c.target) }),
      remove: () => setScenario((s) => ({ ...s, claims: s.claims.filter((_, i) => i !== index) })),
    })),
  ]

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button
          type="button"
          aria-label={t('sim.open')}
          title={t('sim.open')}
          className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border border-[#b7a786] bg-[#fffaf0] px-2 text-xs font-semibold text-[#594b3c] transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
        >
          <IconCalculator aria-hidden="true" className="size-4" />
          {t('sim.open')}
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-[#30291f]/50" />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed inset-2 z-50 flex flex-col overflow-y-auto rounded-xl border border-[#b7a786] bg-[#fffaf0] shadow-xl sm:inset-x-auto sm:left-1/2 sm:top-6 sm:bottom-6 sm:w-[44rem] sm:max-w-[calc(100vw-3rem)] sm:-translate-x-1/2"
        >
          <header className="flex items-center gap-3 border-b border-[#b7a786]/60 px-4 py-3">
            <Dialog.Title className="font-serif text-lg font-semibold text-[#30291f]">
              {t('sim.title')}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label={t('sim.close')}
                className="ml-auto rounded-lg p-1.5 text-[#594b3c] hover:bg-[#f3ead9]"
              >
                <IconX aria-hidden="true" className="size-5" />
              </button>
            </Dialog.Close>
          </header>
          <div className="grid gap-4 p-4">
            <p className="text-sm text-[#594b3c]">{t('sim.intro')}</p>
            <HypothesisForm
              nobles={nobles}
              playerId={playerId}
              playerName={playerName}
              onAdd={(next) => setScenario((s) => next(s))}
            />
            <div>
              {empty ? (
                <p className="text-sm italic text-[#806f57]">{t('sim.empty')}</p>
              ) : (
                <>
                  <ul className="grid gap-1.5" aria-label={t('sim.kind')}>
                    {lines.map((line) => (
                      <li
                        key={line.id}
                        className="flex items-center gap-2 rounded-lg border border-[#b7a786]/50 bg-[#f8f0e2] px-2.5 py-1.5 text-sm"
                      >
                        <span className="min-w-0 flex-1">{line.text}</span>
                        <button
                          type="button"
                          aria-label={`${t('sim.remove')}: ${line.text}`}
                          onClick={line.remove}
                          className="rounded p-1 text-[#594b3c] hover:bg-[#f3ead9]"
                        >
                          <IconX aria-hidden="true" className="size-4" />
                        </button>
                      </li>
                    ))}
                  </ul>
                  <button
                    type="button"
                    onClick={() => setScenario(emptyScenario)}
                    className="mt-2 text-xs font-semibold text-[#a84632] hover:underline"
                  >
                    {t('sim.clear')}
                  </button>
                </>
              )}
            </div>
            <div aria-live="polite" className="grid gap-2">
              {failed && (
                <p role="alert" className="text-sm font-semibold text-[#7a2b1c]">
                  {t('sim.error')}
                </p>
              )}
              {loading && <p className="text-xs text-[#806f57]">{t('sim.loading')}</p>}
              {simulation && !failed && (
                <Comparison
                  current={simulation.current}
                  projected={simulation.projected}
                  playerName={playerName}
                />
              )}
            </div>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function Comparison({
  current,
  projected,
  playerName,
}: {
  current: VictoryReading
  projected: VictoryReading
  playerName: (id: string) => string
}) {
  const { t } = useLanguage()
  const delta = projected.score.total - current.score.total
  const columns: Array<[MessageKey, VictoryReading]> = [
    ['sim.current', current],
    ['sim.projected', projected],
  ]
  return (
    <div className="grid grid-cols-2 gap-3" data-testid="sim-result">
      {columns.map(([labelKey, reading]) => {
        const { score, victory } = reading
        const progress =
          victory.required > 0 ? Math.min(100, (score.total / victory.required) * 100) : 0
        return (
          <section
            key={labelKey}
            aria-label={t(labelKey)}
            className="rounded-lg border border-[#b7a786]/50 bg-[#f8f0e2] p-3"
          >
            <h3 className="text-[10px] font-bold uppercase tracking-[0.1em] text-[#806f57]">
              {t(labelKey)}
            </h3>
            <p className="mt-1 font-serif text-2xl font-semibold text-[#a84632]">
              {score.total}
              <span className="text-sm text-[#806f57]"> / {victory.required}</span>
              {labelKey === 'sim.projected' && delta !== 0 && (
                <span
                  data-testid="sim-delta"
                  aria-label={t('sim.delta')}
                  className={`ml-2 text-sm ${delta > 0 ? 'text-[#2f6f3f]' : 'text-[#7a2b1c]'}`}
                >
                  {delta > 0 ? `+${delta}` : delta}
                </span>
              )}
            </p>
            <div
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={victory.required}
              aria-valuenow={Math.min(score.total, victory.required)}
              className="mt-1 h-1.5 overflow-hidden rounded bg-[#e6dac2]"
            >
              <div className="h-full bg-[#b8860b]" style={{ width: `${progress}%` }} />
            </div>
            <dl className="mt-2 grid grid-cols-[1fr_auto] gap-x-3 gap-y-1 text-xs">
              <dt className="text-[#806f57]">{t('score.titles')}</dt>
              <dd className="text-right font-medium">{score.titles ?? 0}</dd>
              {(score.alliance ?? 0) > 0 && (
                <>
                  <dt className="text-[#806f57]">{t('score.alliance')}</dt>
                  <dd className="text-right font-medium">{score.alliance}</dd>
                </>
              )}
              {victory.mode === 'alliance' && (
                <>
                  <dt className="text-[#806f57]">
                    {t('score.ally', {
                      partner: victory.partner ? playerName(victory.partner) : '',
                    })}
                  </dt>
                  <dd className="text-right font-medium">{score.ally ?? 0}</dd>
                </>
              )}
              <dt className="text-[#806f57]">{t('sim.missing')}</dt>
              <dd className="text-right font-medium">{reading.missing}</dd>
            </dl>
            <div className="mt-2 flex flex-wrap items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.1em]">
              <span
                className={`rounded border px-1.5 py-0.5 ${
                  victory.mode === 'alliance'
                    ? 'border-[#8a929b] bg-[#e4e7ea] text-[#3f464d]'
                    : 'border-[#b8860b] bg-[#f8e8ae] text-[#6b4e0a]'
                }`}
              >
                {t(victory.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo')}
              </span>
              <span
                data-testid={`sim-status-${labelKey === 'sim.current' ? 'current' : 'projected'}`}
                className={`rounded border px-1.5 py-0.5 ${STATUS_STYLES[reading.status]}`}
              >
                {t(STATUS_KEYS[reading.status])}
              </span>
            </div>
          </section>
        )
      })}
    </div>
  )
}

function HypothesisForm({
  nobles,
  playerId,
  playerName,
  onAdd,
}: {
  nobles: Noble[]
  playerId: PlayerId
  playerName: (id: string) => string
  onAdd: (apply: (scenario: VictoryScenario) => VictoryScenario) => void
}) {
  const { t } = useLanguage()
  const [kind, setKind] = useState<Kind>('marriage')
  const [first, setFirst] = useState('')
  const [second, setSecond] = useState('')

  const own = nobles.filter((n) => n.owner === playerId)
  const foreign = nobles.filter((n) => n.owner !== playerId)
  const married = nobles.filter((n) => n.spouse)
  const firstOptions =
    kind === 'claim' ? own : kind === 'marriageEnd' ? married : nobles
  const secondOptions =
    kind === 'claim'
      ? foreign
      : kind === 'marriageEnd'
        ? nobles.filter((n) => n.code === nobles.find((m) => m.code === first)?.spouse)
        : nobles.filter((n) => n.code !== first)
  const needsSecond = kind !== 'death'
  const ready = first !== '' && (!needsSecond || second !== '')

  const labels: Record<Kind, [MessageKey, MessageKey]> = {
    marriage: ['sim.noble', 'sim.spouse'],
    marriageEnd: ['sim.noble', 'sim.spouse'],
    death: ['sim.noble', 'sim.spouse'],
    claim: ['sim.heir', 'sim.target'],
  }
  const option = (n: Noble) => (
    <option key={n.code} value={n.code}>
      {n.name || n.code} ({playerName(n.owner)})
    </option>
  )
  const selectClass =
    'h-8 min-w-0 rounded-lg border border-[#b7a786] bg-white px-2 text-sm text-[#30291f]'

  function add() {
    if (!ready) return
    onAdd((s) => {
      switch (kind) {
        case 'marriage':
          return { ...s, marriages: [...s.marriages, { noble: first, spouse: second }] }
        case 'marriageEnd':
          return { ...s, marriageEnds: [...s.marriageEnds, { noble: first, spouse: second }] }
        case 'death':
          return s.deaths.includes(first) ? s : { ...s, deaths: [...s.deaths, first] }
        case 'claim':
          return { ...s, claims: [...s.claims, { heir: first, target: second }] }
      }
    })
    setFirst('')
    setSecond('')
  }

  return (
    <form
      className="flex flex-wrap items-end gap-2"
      onSubmit={(event) => {
        event.preventDefault()
        add()
      }}
    >
      <label className="grid gap-1 text-xs text-[#594b3c]">
        {t('sim.kind')}
        <select
          value={kind}
          className={selectClass}
          onChange={(event) => {
            setKind(event.target.value as Kind)
            setFirst('')
            setSecond('')
          }}
        >
          {(Object.keys(KIND_KEYS) as Kind[]).map((k) => (
            <option key={k} value={k}>
              {t(KIND_KEYS[k])}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1 text-xs text-[#594b3c]">
        {t(labels[kind][0])}
        <select
          value={first}
          className={selectClass}
          onChange={(event) => {
            setFirst(event.target.value)
            setSecond('')
          }}
        >
          <option value="" />
          {firstOptions.map(option)}
        </select>
      </label>
      {needsSecond && (
        <label className="grid gap-1 text-xs text-[#594b3c]">
          {t(labels[kind][1])}
          <select
            value={second}
            className={selectClass}
            onChange={(event) => setSecond(event.target.value)}
          >
            <option value="" />
            {secondOptions.map(option)}
          </select>
        </label>
      )}
      <button
        type="submit"
        disabled={!ready}
        className="h-8 rounded-lg bg-[#a84632] px-3 text-sm font-semibold text-white disabled:opacity-40"
      >
        {t('sim.add')}
      </button>
    </form>
  )
}
