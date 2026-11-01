import { useEffect, useMemo, useRef, useState } from 'react'
import {
  IconChevronDown,
  IconChevronUp,
  IconFlask,
  IconHierarchy2,
  IconX,
} from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { BannerIcon } from '@/components/ArmsIcons'
import { NoblePawnIcon } from '@/components/NobleHolder'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { playerArms } from '@/lib/heraldry'
import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  countLines,
  layoutTree,
  MEDALLION_ANCHOR_Y,
  placeTree,
  type AlliancePair,
  type ClaimRef,
  type EdgeKind,
  type House,
  type HouseMember,
  type LineageLink,
} from '@/lib/lineage'
import { nobleAttributesByCode, type NobleAttributes } from '@/lib/noble-attributes'
import { useSimulationHistory } from '@/lib/simulation-history'
import {
  useVictorySimulation,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import { cn } from '@/lib/utils'
import { VictoryStatusPanel } from '@/components/VictoryStatusPanel'
import type {
  Bishopric,
  Claim,
  DeceasedNoble,
  Excommunication,
  Fief,
  Marriage,
  Noble,
  Player,
  PlayerId,
  ScoreBreakdown,
  SimulationAction,
  VictoryStatus,
} from '@/types'
import {
  LINEAGE_CLAIM,
  LINEAGE_ENDED,
  LINEAGE_HEAD,
  LINEAGE_PAPER,
  LINEAGE_SECONDARY,
  MAP_INK,
  MAP_PAPER,
  NEUTRAL_ARMY_COLOR,
  SELECTION_HALO,
} from '@/lib/map-palette'

/**
 * Stroke of a marriage by kind. Head and secondary marriages are double lines
 * (a wide stroke with a thin paper line over it); an ended one is hatched.
 */
const EDGE_STYLE: Record<EdgeKind, { stroke: string; double: boolean }> = {
  head: { stroke: LINEAGE_HEAD, double: true },
  secondary: { stroke: LINEAGE_SECONDARY, double: true },
  ended: { stroke: 'url(#lineage-hatch)', double: false },
}
const LINE_WIDTH = 5

interface LineageDialogProps {
  /** Controlled opening, for a trigger outside the header (Realm bookmark). */
  open?: boolean
  onOpenChange?: (open: boolean) => void
  players: Player[]
  nobles: Noble[]
  deceased?: DeceasedNoble[]
  fiefs?: Fief[]
  bishoprics?: Bishopric[]
  cardinals?: string[]
  pope?: string
  king?: string
  queen?: string
  queenDowager?: boolean
  excommunicated?: Excommunication[]
  marriages?: Marriage[]
  claims?: Claim[]
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
  /** House focused when the dialog opens (the viewer's own). */
  defaultFocus?: PlayerId | null
  /** Enables the simulation mode: projects hypothetical actions on the server. */
  simulationRequest?: VictorySimulationRequest | null
  /** Changes when the real state moves on; the simulation is then reset. */
  simulationKey?: string
}

/** What a click selected while simulating. */
type Selection = { type: 'noble'; code: string } | { type: 'link'; a: string; b: string }

/** A started two-step action waiting for the second click. */
interface Pending {
  kind: 'marry' | 'claim'
  from: string
}

/** Wiring of the tree's cards and marriage lines to the simulation. */
interface TreeEdit {
  selected: Selection | null
  pending: Pending | null
  onNoble: (code: string) => void
  onLink: (a: string, b: string) => void
}

function HatchDefs() {
  return (
    <defs>
      <pattern
        id="lineage-hatch"
        patternUnits="userSpaceOnUse"
        width="6"
        height="6"
        patternTransform="rotate(45)"
      >
        <rect width="6" height="6" fill={LINEAGE_PAPER} />
        <line x1="0" y1="0" x2="0" y2="6" stroke={LINEAGE_ENDED} strokeWidth="3" />
      </pattern>
    </defs>
  )
}

/**
 * Header button opening a full-screen dialog on the houses' lineage: a tree
 * (one column per house, joined by their marriages and claims) to understand
 * the whole board, with the alliance graph as an inset to plan the victory.
 */
export function LineageDialog(props: LineageDialogProps) {
  const { t } = useLanguage()
  const [focus, setFocus] = useState<PlayerId | null>(props.defaultFocus ?? null)
  const [hidden, setHidden] = useState<Set<PlayerId>>(new Set())
  const [showEnded, setShowEnded] = useState(true)
  const [showClaims, setShowClaims] = useState(true)
  const [ownOpen, setOwnOpen] = useState(false)
  const open = props.open ?? ownOpen
  const handleOpenChange = (next: boolean) => {
    setOwnOpen(next)
    props.onOpenChange?.(next)
  }

  const [simulating, setSimulating] = useState(false)
  const [selection, setSelection] = useState<Selection | null>(null)
  const [pending, setPending] = useState<Pending | null>(null)
  const [rejection, setRejection] = useState<string | null>(null)
  const history = useSimulationHistory()
  const sim = useVictorySimulation(
    history.actions,
    props.simulationRequest ?? null,
    simulating,
  )
  const projected = simulating ? sim.simulation : null
  const data = projected?.state ?? props
  const nobles = data.nobles
  const deceased = data.deceased
  const fiefs = data.fiefs
  const marriages = data.marriages
  const claimList = data.claims
  const scores = projected?.state.scores ?? props.scores
  const victory = projected?.state.victory ?? props.victory

  const { dropLast, reset: resetHistory } = history
  useEffect(() => {
    if (sim.rejection && history.actions.length > 0) {
      setRejection(sim.rejection)
      dropLast()
    }
    // Only a new refusal matters; the actions it refers to are already dropped.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sim.rejection])
  useEffect(() => {
    resetHistory()
    setSelection(null)
    setPending(null)
    setRejection(null)
  }, [props.simulationKey, resetHistory])

  const clearTransient = () => {
    setSelection(null)
    setPending(null)
    setRejection(null)
  }
  const apply = (action: SimulationAction) => {
    history.push(action)
    clearTransient()
  }
  const onNoble = (code: string) => {
    setRejection(null)
    if (pending) {
      if (code === pending.from) return
      apply({ type: pending.kind, noble: pending.from, other: code })
      return
    }
    setSelection({ type: 'noble', code })
  }
  const onLink = (a: string, b: string) => {
    if (pending) return
    setRejection(null)
    setSelection({ type: 'link', a, b })
  }
  const stopSimulating = () => {
    setSimulating(false)
    clearTransient()
  }
  const nameOf = (code: string) =>
    nobles.find((n) => n.code === code)?.name ||
    (deceased ?? []).find((n) => n.code === code)?.name ||
    code

  const allHouses = useMemo(
    () =>
      buildHouses(
        props.players,
        nobles,
        deceased ?? [],
        fiefs ?? [],
        (claimList ?? []).filter((claim) => showEnded || !claim.ended),
        {
          bishoprics: data.bishoprics,
          cardinals: data.cardinals,
          pope: data.pope,
          king: data.king,
          queen: data.queen,
          queenDowager: data.queenDowager,
          excommunicated: data.excommunicated,
        },
      ),
    [
      props.players,
      nobles,
      deceased,
      fiefs,
      claimList,
      data.bishoprics,
      data.cardinals,
      data.pope,
      data.king,
      data.queen,
      data.queenDowager,
      data.excommunicated,
      showEnded,
    ],
  )
  const houses = useMemo(
    () => allHouses.filter((house) => !hidden.has(house.player.id)),
    [allHouses, hidden],
  )
  const links = useMemo(() => {
    const visible = new Set(
      houses.flatMap((house) => house.members.map((member) => member.code)),
    )
    return buildLinks(allHouses, marriages ?? []).filter(
      (link) => visible.has(link.a) && visible.has(link.b) && (showEnded || link.active),
    )
  }, [allHouses, houses, marriages, showEnded])
  const claims = useMemo(() => {
    if (!showClaims) return []
    const visible = new Set(
      houses.flatMap((house) => house.members.map((member) => member.code)),
    )
    return (claimList ?? []).filter(
      (claim) =>
        visible.has(claim.heir) &&
        visible.has(claim.target) &&
        (showEnded || !claim.ended),
    )
  }, [houses, claimList, showClaims, showEnded])
  const attributes = useMemo(
    () =>
      nobleAttributesByCode({
        nobles,
        fiefs,
        bishoprics: data.bishoprics,
        cardinals: data.cardinals,
        pope: data.pope,
        king: data.king,
        queen: data.queen,
        queenDowager: data.queenDowager,
        excommunicated: data.excommunicated,
      }),
    [
      nobles,
      fiefs,
      data.bishoprics,
      data.cardinals,
      data.pope,
      data.king,
      data.queen,
      data.queenDowager,
      data.excommunicated,
    ],
  )
  const focusId =
    focus && houses.some((house) => house.player.id === focus) ? focus : null
  const playerName = (house: House) => house.player.name || house.player.id

  const toggleHouse = (id: PlayerId) =>
    setHidden((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <Dialog.Trigger asChild>
        <button
          type="button"
          aria-label={t('lineage.open')}
          title={t('lineage.open')}
          className="inline-flex h-8 shrink-0 items-center rounded-lg border border-line bg-paper-50 px-2 text-ink-700 transition hover:bg-paper-100 hover:text-ink-900 focus-visible:ring-2 focus-visible:ring-brass-700/40 focus-visible:outline-none"
        >
          <IconHierarchy2 aria-hidden="true" className="size-4" />
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-ink-900/50" />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed inset-2 z-50 flex flex-col overflow-hidden rounded-xl border border-line bg-paper-50 shadow-xl sm:inset-6"
        >
          <header className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line/60 px-4 py-3">
            <Dialog.Title className="font-display text-lg font-semibold text-ink-900">
              {t('lineage.title')}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label={t('lineage.close')}
                className="ml-auto rounded-lg p-1.5 text-ink-700 hover:bg-paper-100"
              >
                <IconX aria-hidden="true" className="size-5" />
              </button>
            </Dialog.Close>
            <div className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-xs text-ink-700">
              <label className="flex items-center gap-1.5">
                {t('lineage.focus')}
                <select
                  value={focusId ?? ''}
                  onChange={(event) => setFocus(event.target.value || null)}
                  className="rounded-md border border-line bg-paper-50 px-1.5 py-1"
                >
                  <option value="">{t('lineage.focusNone')}</option>
                  {houses.map((house) => (
                    <option key={house.player.id} value={house.player.id}>
                      {playerName(house)}
                    </option>
                  ))}
                </select>
              </label>
              <div
                role="group"
                aria-label={t('lineage.families')}
                className="flex flex-wrap items-center gap-1.5"
              >
                {allHouses.map((house) => {
                  const on = !hidden.has(house.player.id)
                  return (
                    <button
                      key={house.player.id}
                      type="button"
                      aria-pressed={on}
                      onClick={() => toggleHouse(house.player.id)}
                      className={cn(
                        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 font-semibold',
                        on
                          ? 'border-ink-900/40 bg-paper-100'
                          : 'border-line/60 opacity-50',
                      )}
                    >
                      <span
                        aria-hidden="true"
                        className="size-2.5 rounded-full"
                        style={{ backgroundColor: house.player.color }}
                      />
                      {playerName(house)}
                    </button>
                  )
                })}
              </div>
              <label className="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={showEnded}
                  onChange={(e) => setShowEnded(e.target.checked)}
                />
                {t('lineage.showEnded')}
              </label>
              <label className="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={showClaims}
                  onChange={(e) => setShowClaims(e.target.checked)}
                />
                {t('lineage.showClaims')}
              </label>
            </div>
            {props.simulationRequest && (
              <SimulationBar
                simulating={simulating}
                nobles={nobles}
                fiefs={fiefs ?? []}
                king={data.king}
                selection={selection}
                pending={pending}
                actions={history.actions}
                canUndo={history.canUndo}
                canRedo={history.canRedo}
                loading={sim.loading}
                failed={sim.failed}
                rejection={rejection}
                nameOf={nameOf}
                onStart={() => setSimulating(true)}
                onStop={stopSimulating}
                onUndo={() => {
                  history.undo()
                  clearTransient()
                }}
                onRedo={() => {
                  history.redo()
                  clearTransient()
                }}
                onReset={() => {
                  history.reset()
                  clearTransient()
                }}
                onAction={apply}
                onPending={(next) => {
                  setPending(next)
                  setSelection(null)
                }}
                onCancel={clearTransient}
              />
            )}
          </header>
          <div className="relative flex min-h-0 flex-1 flex-col">
            <LineageTree
              houses={houses}
              links={links}
              claims={claims}
              focus={focusId}
              attributes={attributes}
              edit={
                simulating ? { selected: selection, pending, onNoble, onLink } : undefined
              }
            />
            {simulating && (
              <div
                aria-hidden="true"
                data-testid="lineage-watermark"
                className="pointer-events-none absolute inset-0 z-20 flex select-none items-center justify-center overflow-hidden"
              >
                <span className="-rotate-12 font-display text-[min(16vw,8rem)] font-semibold uppercase tracking-widest text-ink-500/15">
                  {t('lineage.watermark')}
                </span>
              </div>
            )}
            <AllianceInset
              houses={houses}
              links={links}
              focus={focusId}
              onFocus={setFocus}
              scores={scores}
              victory={victory}
            />
          </div>
          <VictoryStatusPanel
            players={props.players}
            scores={scores}
            victory={victory}
            simulation={projected}
          />
          <Legend />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function SimulationBar({
  simulating,
  nobles,
  fiefs,
  king,
  selection,
  pending,
  actions,
  canUndo,
  canRedo,
  loading,
  failed,
  rejection,
  nameOf,
  onStart,
  onStop,
  onUndo,
  onRedo,
  onReset,
  onAction,
  onPending,
  onCancel,
}: {
  simulating: boolean
  nobles: Noble[]
  fiefs: Fief[]
  king?: string
  selection: Selection | null
  pending: Pending | null
  actions: SimulationAction[]
  canUndo: boolean
  canRedo: boolean
  loading: boolean
  failed: boolean
  rejection: string | null
  nameOf: (code: string) => string
  onStart: () => void
  onStop: () => void
  onUndo: () => void
  onRedo: () => void
  onReset: () => void
  onAction: (action: SimulationAction) => void
  onPending: (pending: Pending) => void
  onCancel: () => void
}) {
  const { t } = useLanguage()
  const button =
    'rounded-md border border-line bg-paper-50 px-2 py-0.5 text-xs font-semibold enabled:hover:bg-paper-100 disabled:opacity-40'
  if (!simulating) {
    return (
      <div className="flex w-full">
        <button
          type="button"
          onClick={onStart}
          className={cn(button, 'inline-flex items-center gap-1')}
        >
          <IconFlask aria-hidden="true" className="size-4" />
          {t('sim.start')}
        </button>
      </div>
    )
  }
  const noble =
    selection?.type === 'noble'
      ? nobles.find((n) => n.code === selection.code)
      : undefined
  const alive = noble !== undefined
  const heldFiefs = noble ? fiefs.filter((fief) => fief.holder === noble.code) : []
  const log = (action: SimulationAction) =>
    t(`sim.log.${action.type}` as MessageKey, {
      noble: 'noble' in action ? nameOf(action.noble) : '',
      other: 'other' in action ? nameOf(action.other) : '',
      fief: 'fief' in action ? action.fief : '',
    })
  return (
    <div className="flex w-full flex-col gap-2 rounded-lg border border-brass-500/60 bg-brass-300/30 p-2 text-xs">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-semibold text-brass-700">{t('sim.banner')}</span>
        <button type="button" className={button} disabled={!canUndo} onClick={onUndo}>
          {t('sim.undo')}
        </button>
        <button type="button" className={button} disabled={!canRedo} onClick={onRedo}>
          {t('sim.redo')}
        </button>
        <button
          type="button"
          className={button}
          disabled={actions.length === 0}
          onClick={onReset}
        >
          {t('sim.reset')}
        </button>
        <button type="button" className={cn(button, 'ml-auto')} onClick={onStop}>
          {t('sim.stop')}
        </button>
      </div>
      {pending ? (
        <div className="flex flex-wrap items-center gap-2" role="status">
          <span className="font-semibold">
            {t(pending.kind === 'marry' ? 'sim.pickSpouse' : 'sim.pickTarget', {
              name: nameOf(pending.from),
            })}
          </span>
          <button type="button" className={button} onClick={onCancel}>
            {t('sim.cancel')}
          </button>
        </div>
      ) : selection?.type === 'noble' ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">
            {t('sim.selectedNoble', { name: nameOf(selection.code) })}
          </span>
          <button
            type="button"
            className={button}
            disabled={!alive}
            onClick={() => onAction({ type: 'kill', noble: selection.code })}
          >
            {t('sim.kill')}
          </button>
          <button
            type="button"
            className={button}
            disabled={!alive || Boolean(noble?.spouse)}
            onClick={() => onPending({ kind: 'marry', from: selection.code })}
          >
            {t('sim.marry')}
          </button>
          <button
            type="button"
            className={button}
            disabled={!alive}
            onClick={() => onPending({ kind: 'claim', from: selection.code })}
          >
            {t('sim.claim')}
          </button>
          {heldFiefs.map((fief) => (
            <button
              key={fief.capital}
              type="button"
              className={button}
              disabled={fief.title === 'duchy'}
              onClick={() => onAction({ type: 'rank', fief: fief.capital })}
            >
              {t('sim.rank', { fief: fief.capital })}
            </button>
          ))}
          {noble && noble.code !== king && (
            <button
              type="button"
              className={button}
              onClick={() => onAction({ type: 'crown', noble: selection.code })}
            >
              {t('sim.crown')}
            </button>
          )}
          {noble && noble.code === king && (
            <button
              type="button"
              className={button}
              onClick={() => onAction({ type: 'dethrone' })}
            >
              {t('sim.dethrone')}
            </button>
          )}
          <button type="button" className={button} onClick={onCancel}>
            {t('sim.cancel')}
          </button>
        </div>
      ) : selection?.type === 'link' ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">
            {t('sim.selectedLink', { a: nameOf(selection.a), b: nameOf(selection.b) })}
          </span>
          <button
            type="button"
            className={button}
            onClick={() =>
              onAction({ type: 'divorce', noble: selection.a, other: selection.b })
            }
          >
            {t('sim.divorce')}
          </button>
          <button type="button" className={button} onClick={onCancel}>
            {t('sim.cancel')}
          </button>
        </div>
      ) : (
        <p>{t('sim.hint')}</p>
      )}
      {rejection && (
        <p role="alert" className="font-semibold text-vermilion-700">
          {t('sim.rejected', { reason: rejection })}
        </p>
      )}
      {failed && (
        <p role="alert" className="font-semibold text-vermilion-700">
          {t('sim.failed')}
        </p>
      )}
      {loading && <p className="text-ink-500">{t('sim.loading')}</p>}
      {actions.length > 0 && (
        <ol className="flex flex-wrap gap-1.5" aria-label={t('sim.panel')}>
          {actions.map((action, index) => (
            <li
              key={index}
              className="rounded-full border border-brass-500/60 bg-paper-50 px-2 py-0.5"
            >
              {log(action)}
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}

function Swatch({ kind }: { kind: EdgeKind | 'claim' }) {
  return (
    <svg aria-hidden="true" width="28" height="10" className="shrink-0">
      <HatchDefs />
      {kind === 'claim' ? (
        <line
          x1="1"
          y1="5"
          x2="27"
          y2="5"
          stroke={LINEAGE_CLAIM}
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeDasharray="1 4"
        />
      ) : (
        <>
          <line
            x1="0"
            y1="5"
            x2="28"
            y2="5"
            stroke={EDGE_STYLE[kind].stroke}
            strokeWidth={LINE_WIDTH}
          />
          {EDGE_STYLE[kind].double && (
            <line x1="0" y1="5" x2="28" y2="5" stroke={LINEAGE_PAPER} strokeWidth="1.5" />
          )}
        </>
      )}
    </svg>
  )
}

function Legend() {
  const { t } = useLanguage()
  const kinds: Array<[EdgeKind | 'claim', MessageKey]> = [
    ['head', 'lineage.legend.head'],
    ['secondary', 'lineage.legend.secondary'],
    ['ended', 'lineage.legend.ended'],
    ['claim', 'lineage.legend.claim'],
  ]
  return (
    <footer className="flex flex-wrap gap-x-4 gap-y-1 border-t border-line/60 px-4 py-2 text-xs text-ink-700">
      {kinds.map(([kind, label]) => (
        <span key={kind} className="inline-flex items-center gap-1.5">
          <Swatch kind={kind} />
          {t(label)}
        </span>
      ))}
    </footer>
  )
}

function kindLabel(link: LineageLink): MessageKey {
  return `lineage.kind.${link.kind}` as MessageKey
}

function titleLabel(
  t: ReturnType<typeof useLanguage>['t'],
  member: HouseMember['fiefs'][number],
) {
  return `${t(`fief.title.${member.title}` as MessageKey)} · ${member.capital}`
}

type Translate = ReturnType<typeof useLanguage>['t']

/** One text line under a noble's name. */
interface MedallionLine {
  key: string
  text: string
  /** Assassinations and excommunication are written in the error ink. */
  alert?: boolean
  testId?: string
}

/** Characters of a name that fit beside the pawn and the rank before it wraps. */
const NAME_LINE_CHARS = 13

/** The titles, standing and deaths of a member, one text line each. */
function medallionTexts(member: HouseMember, t: Translate): MedallionLine[] {
  const lines: MedallionLine[] = []
  if (member.dead && member.cause)
    lines.push({
      key: 'cause',
      text: `✝ ${t(`lineage.cause.${member.cause}` as MessageKey)}`,
    })
  if (member.commissioner)
    lines.push({
      key: 'commissioner',
      alert: true,
      testId: 'assassination-commissioner',
      text: t('lineage.assassinatedBy', {
        name: member.commissioner.name,
        code: member.commissioner.code,
      }),
    })
  for (const victim of member.commissioned)
    lines.push({
      key: `commissioned-${victim.code}`,
      alert: true,
      testId: 'assassination-victim',
      text: t('lineage.commissionedKilling', { name: victim.name, code: victim.code }),
    })
  for (const held of member.religious)
    lines.push({
      key: `religious-${held.title}-${held.seat ?? ''}`,
      text: t(`lineage.religious.${held.title}` as MessageKey, { seat: held.seat ?? '' }),
    })
  if (member.excommunicated)
    lines.push({ key: 'excommunicated', alert: true, text: t('lineage.excommunicated') })
  for (const fief of member.fiefs)
    lines.push({ key: `fief-${fief.capital}`, text: titleLabel(t, fief) })
  for (const dignity of member.dignities)
    lines.push({ key: `dignity-${dignity}`, text: t(`dignity.${dignity}` as MessageKey) })
  if (member.claim?.outcome)
    lines.push({
      key: 'claim-outcome',
      text: t(
        member.claim.outcome === 'honoured'
          ? 'lineage.claimHonoured'
          : 'lineage.claimLapsed',
        { target: member.claim.targetName },
      ),
    })
  return lines
}

function LineageTree({
  houses,
  links,
  claims,
  focus,
  attributes,
  edit,
}: {
  houses: House[]
  links: LineageLink[]
  claims: Claim[]
  focus: PlayerId | null
  attributes: Map<string, NobleAttributes>
  edit?: TreeEdit
}) {
  const { t } = useLanguage()
  const scroller = useRef<HTMLDivElement>(null)
  const claimRefs = useMemo<ClaimRef[]>(
    () =>
      claims.map(({ heir, target, spouse, ended }) => ({ heir, target, spouse, ended })),
    [claims],
  )
  const layout = useMemo(
    () => layoutTree(houses, links, focus, claimRefs),
    [houses, links, focus, claimRefs],
  )
  const texts = useMemo(() => {
    const byCode = new Map<string, MedallionLine[]>()
    for (const house of houses)
      for (const member of house.members)
        byCode.set(member.code, medallionTexts(member, t))
    return byCode
  }, [houses, t])
  const placement = useMemo(() => {
    const names = new Map<string, number>()
    for (const house of houses)
      for (const member of house.members)
        names.set(
          member.code,
          Math.max(0, Math.ceil(member.name.length / NAME_LINE_CHARS) - 1),
        )
    return placeTree(
      layout,
      (code) =>
        (names.get(code) ?? 0) +
        countLines((texts.get(code) ?? []).map((line) => line.text)),
    )
  }, [layout, houses, texts])
  const { width, height, boxes, columnX } = placement

  useEffect(() => {
    const element = scroller.current
    if (!element || !focus) return
    const index = layout.columns.findIndex((house) => house.player.id === focus)
    if (index < 0) return
    element.scrollLeft = columnX[index] + 88 - element.clientWidth / 2
    element.scrollTop = 0
  }, [focus, layout, columnX])

  const isSelected = (link: LineageLink) =>
    edit?.selected?.type === 'link' &&
    ((edit.selected.a === link.a && edit.selected.b === link.b) ||
      (edit.selected.a === link.b && edit.selected.b === link.a))

  return (
    <div ref={scroller} className="min-h-0 flex-1 overflow-auto p-4">
      <div className="relative" style={{ width, height }}>
        <svg
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 z-0"
          width={width}
          height={height}
        >
          <HatchDefs />
          {placement.marriages.map(({ link, aligned, d }) => {
            const style = EDGE_STYLE[link.kind]
            return (
              <g
                key={link.key}
                data-marriage={link.key}
                data-kind={link.kind}
                data-aligned={aligned}
              >
                {isSelected(link) && (
                  <path
                    d={d}
                    fill="none"
                    stroke={SELECTION_HALO}
                    strokeWidth={LINE_WIDTH + 8}
                    strokeOpacity={0.6}
                  />
                )}
                <path d={d} fill="none" stroke={style.stroke} strokeWidth={LINE_WIDTH} />
                {style.double && (
                  <path d={d} fill="none" stroke={LINEAGE_PAPER} strokeWidth={1.5} />
                )}
                {edit && link.active && (
                  <path
                    d={d}
                    data-testid={`link-${link.a}-${link.b}`}
                    role="button"
                    aria-label={`${link.a} - ${link.b}`}
                    fill="none"
                    stroke="transparent"
                    strokeWidth={16}
                    style={{ pointerEvents: 'stroke', cursor: 'pointer' }}
                    onClick={() => edit.onLink(link.a, link.b)}
                  />
                )}
              </g>
            )
          })}
          {placement.claims.map(({ claim: index, d }) => {
            const claim = claims[index]
            return (
              <path
                key={`claim-${claim.heir}-${claim.turn}-${claim.target}`}
                data-claim={`${claim.heir}>${claim.target}`}
                data-ended={Boolean(claim.ended)}
                d={d}
                fill="none"
                stroke={LINEAGE_CLAIM}
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeDasharray="1 4"
                opacity={claim.ended ? 0.4 : 1}
              />
            )
          })}
        </svg>
        {layout.columns.map((house, columnIndex) => (
          <section
            key={house.player.id}
            aria-label={house.player.name || house.player.id}
            className="pointer-events-none absolute inset-0 z-10"
          >
            <h3
              className={cn(
                'absolute top-0 flex items-center gap-2 rounded-md px-2 py-1 text-sm font-semibold text-ink-900',
                house.player.id === focus && 'bg-paper-100 ring-1 ring-ink-900/30',
              )}
              style={{ left: columnX[columnIndex], width: 176 }}
            >
              <BannerIcon
                id={`lineage-arms-${house.player.id}`}
                arms={playerArms(house.player)}
              />
              {house.player.name || house.player.id}
            </h3>
            <ol>
              {house.members.map((member, index) => {
                const box = boxes.get(member.code)!
                const lines = texts.get(member.code) ?? []
                return (
                  <li
                    key={member.code}
                    data-noble={member.code}
                    className={cn(
                      'pointer-events-auto absolute rounded-xl border bg-paper-50 px-2 pb-1.5 pt-1 text-xs',
                      member.dead
                        ? 'border-dashed border-line/70 opacity-60'
                        : 'border-line',
                      edit && 'cursor-pointer',
                      edit?.selected?.type === 'noble' &&
                        edit.selected.code === member.code &&
                        'ring-2 ring-brass-700',
                      edit?.pending?.from === member.code &&
                        'outline-2 outline-dashed outline-ink-700',
                    )}
                    onClick={edit ? () => edit.onNoble(member.code) : undefined}
                    style={{ left: box.x, top: box.y, width: box.w, minHeight: box.h }}
                  >
                    <div
                      className="flex items-start gap-1.5"
                      style={{ minHeight: MEDALLION_ANCHOR_Y * 2 - 4 }}
                    >
                      <span className="text-2xl leading-none">
                        <NoblePawnIcon
                          noble={{ status: member.status ?? 'free' }}
                          color={house.player.color || NEUTRAL_ARMY_COLOR}
                          attributes={attributes.get(member.code)}
                        />
                      </span>
                      {!member.dead && (
                        <span className="w-3 shrink-0 pt-1.5 text-right font-display text-sm leading-4 text-ink-500">
                          {index + 1}
                        </span>
                      )}
                      <span
                        className={cn(
                          'min-w-0 flex-1 break-words pt-1.5 text-sm font-semibold leading-4',
                          member.dead && 'line-through',
                        )}
                      >
                        {member.name}
                      </span>
                      {member.sex && (
                        <span
                          aria-label={member.sex}
                          className={cn(
                            'pt-1.5 leading-4',
                            member.sex === 'female'
                              ? 'text-brass-700'
                              : 'text-winter-700',
                          )}
                        >
                          {member.sex === 'female' ? '♀' : '♂'}
                        </span>
                      )}
                    </div>
                    {lines.map((line) => (
                      <p
                        key={line.key}
                        data-testid={line.testId}
                        className={cn(
                          'break-words leading-4',
                          line.alert ? 'text-vermilion-700' : 'text-ink-700',
                        )}
                      >
                        {line.text}
                      </p>
                    ))}
                  </li>
                )
              })}
            </ol>
          </section>
        ))}
      </div>
    </div>
  )
}

/**
 * The alliance graph as a corner inset of the tree: folded by default so
 * the tree keeps the room.
 */
function AllianceInset(props: {
  houses: House[]
  links: LineageLink[]
  focus: PlayerId | null
  onFocus: (id: PlayerId | null) => void
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
}) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)
  return (
    <aside className="pointer-events-none absolute bottom-3 right-3 z-30 flex max-h-[70%] w-[min(26rem,calc(100%-1.5rem))] flex-col items-end gap-1">
      <button
        type="button"
        aria-expanded={open}
        aria-controls="lineage-alliances"
        onClick={() => setOpen((current) => !current)}
        className="pointer-events-auto inline-flex items-center gap-1.5 rounded-lg border border-line bg-paper-50 px-2.5 py-1 text-xs font-semibold text-ink-700 shadow-sm hover:bg-paper-100"
      >
        {open ? (
          <IconChevronDown aria-hidden="true" className="size-3.5" />
        ) : (
          <IconChevronUp aria-hidden="true" className="size-3.5" />
        )}
        {t('lineage.alliances')}
      </button>
      {open && (
        <div
          id="lineage-alliances"
          className="pointer-events-auto min-h-0 w-full overflow-auto rounded-xl border border-line bg-paper-50/95 p-3 shadow-lg"
        >
          <AllianceGraph {...props} />
        </div>
      )}
    </aside>
  )
}

function AllianceGraph({
  houses,
  links,
  focus,
  onFocus,
  scores,
  victory,
}: {
  houses: House[]
  links: LineageLink[]
  focus: PlayerId | null
  onFocus: (id: PlayerId | null) => void
  scores?: Record<string, ScoreBreakdown>
  victory?: VictoryStatus
}) {
  const { t } = useLanguage()
  const pairs = useMemo(() => buildAlliancePairs(links), [links])
  const houseName = (id: PlayerId) => {
    const house = houses.find((h) => h.player.id === id)
    return house?.player.name || id
  }
  const nobleName = (code: string) => {
    for (const house of houses) {
      const member = house.members.find((m) => m.code === code)
      if (member) return member.name
    }
    return code
  }
  const marriageName = (a: string, b: string) => `${nobleName(a)} × ${nobleName(b)}`

  const cx = 200
  const cy = 160
  const others = houses.filter((house) => house.player.id !== focus)
  const position = new Map<PlayerId, { x: number; y: number }>()
  if (focus) {
    position.set(focus, { x: cx, y: cy })
    others.forEach((house, index) => {
      const angle = (2 * Math.PI * index) / others.length - Math.PI / 2
      position.set(house.player.id, {
        x: cx + 125 * Math.cos(angle),
        y: cy + 115 * Math.sin(angle),
      })
    })
  } else {
    houses.forEach((house, index) => {
      const angle = (2 * Math.PI * index) / houses.length - Math.PI / 2
      position.set(house.player.id, {
        x: cx + 125 * Math.cos(angle),
        y: cy + 115 * Math.sin(angle),
      })
    })
  }
  const dim = (pair: AlliancePair) => focus !== null && !pair.houses.includes(focus)

  return (
    <div className="grid gap-3">
      <svg
        viewBox="0 0 400 330"
        role="img"
        aria-label={t('lineage.alliances')}
        className="w-full self-start"
      >
        <HatchDefs />
        {pairs.map((pair) => {
          const a = position.get(pair.houses[0])
          const b = position.get(pair.houses[1])
          if (!a || !b) return null
          const style = EDGE_STYLE[pair.kind]
          return (
            <g key={pair.key} opacity={dim(pair) ? 0.2 : 1}>
              <line
                x1={a.x}
                y1={a.y}
                x2={b.x}
                y2={b.y}
                stroke={style.stroke}
                strokeWidth={LINE_WIDTH}
              />
              {style.double && (
                <line
                  x1={a.x}
                  y1={a.y}
                  x2={b.x}
                  y2={b.y}
                  stroke={LINEAGE_PAPER}
                  strokeWidth={1.5}
                />
              )}
              <text
                x={(a.x + b.x) / 2}
                y={(a.y + b.y) / 2 - 6}
                textAnchor="middle"
                className="fill-ink-900 text-2xs font-bold"
                stroke={MAP_PAPER}
                strokeWidth={3}
                paintOrder="stroke"
              >
                {pair.head ? '★ ' : ''}×{pair.links.length} · {pair.weight}
              </text>
            </g>
          )
        })}
        {houses.map((house) => {
          const p = position.get(house.player.id)!
          const total = scores?.[house.player.id]?.total ?? 0
          const goal = victory?.players[house.player.id]
          const isFocus = focus === house.player.id
          return (
            <g
              key={house.player.id}
              role="button"
              tabIndex={0}
              aria-label={house.player.name || house.player.id}
              aria-pressed={isFocus}
              className="cursor-pointer outline-none"
              onClick={() => onFocus(isFocus ? null : house.player.id)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onFocus(isFocus ? null : house.player.id)
                }
              }}
            >
              <circle
                cx={p.x}
                cy={p.y}
                r={isFocus ? 28 : 24}
                fill={house.player.color}
                stroke={isFocus ? MAP_INK : MAP_PAPER}
                strokeWidth={isFocus ? 4 : 3}
              />
              <text
                x={p.x}
                y={p.y + 4}
                textAnchor="middle"
                className="fill-white text-xs font-bold"
              >
                {total}
                {goal ? `/${goal.required}` : ''}
              </text>
              <text
                x={p.x}
                y={p.y + 42}
                textAnchor="middle"
                className="fill-ink-900 text-2xs font-semibold"
              >
                {house.player.name || house.player.id}
              </text>
            </g>
          )
        })}
      </svg>
      <div className="grid content-start gap-3">
        {pairs.length === 0 && (
          <p className="text-sm text-ink-500">{t('lineage.noAlliance')}</p>
        )}
        <ul className="grid gap-2">
          {pairs.map((pair) => {
            const carrier = pair.carrier
            const others = pair.links.filter((link) => link !== carrier)
            return (
              <li
                key={pair.key}
                className={cn(
                  'rounded-lg border bg-paper-100 p-2.5 text-sm',
                  focus && pair.houses.includes(focus)
                    ? 'border-ink-900'
                    : 'border-line/60',
                  dim(pair) && 'opacity-50',
                )}
              >
                <div className="flex flex-wrap items-center gap-2 font-semibold">
                  <Swatch kind={pair.kind} />
                  {houseName(pair.houses[0])} ⟷ {houseName(pair.houses[1])}
                  <span className="text-xs font-normal text-ink-500">
                    {t(kindLabel(carrier))} ·{' '}
                    {t('lineage.weight', { weight: pair.weight })}
                  </span>
                </div>
                <dl className="mt-1.5 grid gap-1 text-xs">
                  <div>
                    <dt className="inline text-ink-500">{t('lineage.toCancel')} : </dt>
                    <dd className="inline font-medium">
                      {marriageName(carrier.a, carrier.b)}
                    </dd>
                  </div>
                  {carrier.headSuccessors.map((successor) => (
                    <div key={successor.player}>
                      <dt className="inline text-ink-500">
                        {successor.marriage
                          ? t('lineage.takesOver', { house: houseName(successor.player) })
                          : t('lineage.noSuccessor', {
                              house: houseName(successor.player),
                            })}
                        {successor.marriage ? ' : ' : ''}
                      </dt>
                      {successor.marriage && (
                        <dd className="inline font-medium">
                          {marriageName(
                            successor.marriage.nobleA,
                            successor.marriage.nobleB,
                          )}
                        </dd>
                      )}
                    </div>
                  ))}
                  {others.length > 0 && (
                    <div>
                      <dt className="inline text-ink-500">
                        {t('lineage.otherMarriages')} :{' '}
                      </dt>
                      <dd className="inline">
                        {others.map((link) => marriageName(link.a, link.b)).join(', ')}
                      </dd>
                    </div>
                  )}
                </dl>
              </li>
            )
          })}
        </ul>
        <ul className="grid gap-2">
          {houses.map((house) => {
            const goal = victory?.players[house.player.id]
            return (
              <li
                key={house.player.id}
                className="flex flex-wrap items-center gap-2 text-sm"
              >
                <span
                  aria-hidden="true"
                  className="size-3 rounded-full border border-ink-900/30"
                  style={{ backgroundColor: house.player.color }}
                />
                <span className="font-semibold">
                  {house.player.name || house.player.id}
                </span>
                <span className="font-display text-brass-700">
                  {scores?.[house.player.id]?.total ?? 0}
                  {goal ? ` / ${goal.required}` : ''}
                </span>
                {goal && (
                  <span className="rounded border border-brass-500 bg-brass-300 px-1.5 text-2xs font-bold uppercase text-ink-900">
                    {t(
                      goal.mode === 'alliance' ? 'score.modeAlliance' : 'score.modeSolo',
                    )}
                    {goal.partner
                      ? ` · ${t('score.ally', { partner: houseName(goal.partner) })}`
                      : ''}
                  </span>
                )}
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
