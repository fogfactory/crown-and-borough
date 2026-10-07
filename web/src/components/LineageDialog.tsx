import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import { IconFlask, IconHierarchy2, IconX } from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  CARD_H,
  CARD_W,
  claimRoute,
  GAP_X,
  layoutColumns,
  marriageSegment,
  PITCH,
  type AlliancePair,
  type EdgeKind,
  type House,
  type HouseMember,
  type LineageLink,
} from '@/lib/lineage'
import { useSimulationHistory } from '@/lib/simulation-history'
import {
  useVictorySimulation,
  type VictorySimulationRequest,
} from '@/lib/use-victory-simulation'
import { cn } from '@/lib/utils'
import { VictoryStatusPanel } from '@/components/VictoryStatusPanel'
import type {
  Claim,
  DeceasedNoble,
  Fief,
  Marriage,
  Noble,
  Player,
  PlayerId,
  ScoreBreakdown,
  SimulationAction,
  VictoryStatus,
} from '@/types'

const GOLD = '#b8860b'
const SILVER = '#9aa3ad'
const CLAIM = '#7a4fa3'

/** Stroke of a marriage by kind: gold bold head, thin silver secondary, hatched ended. */
const EDGE_STYLE: Record<EdgeKind, { stroke: string; width: number; dash?: string }> = {
  head: { stroke: GOLD, width: 5 },
  secondary: { stroke: SILVER, width: 1.5 },
  ended: { stroke: 'url(#lineage-hatch)', width: 7 },
}

const HEADER_H = 44

/** A card dragged away from its automatic place, in pixels. */
interface Offset {
  dx: number
  dy: number
}

interface LineageDialogProps {
  players: Player[]
  nobles: Noble[]
  deceased?: DeceasedNoble[]
  fiefs?: Fief[]
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
type Selection =
  | { type: 'noble'; code: string }
  | { type: 'link'; a: string; b: string }

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

type View = 'tree' | 'graph'

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
        <rect width="6" height="6" fill="#fffaf0" />
        <line x1="0" y1="0" x2="0" y2="6" stroke="#8c8372" strokeWidth="3" />
      </pattern>
      <marker
        id="lineage-arrow"
        viewBox="0 0 10 10"
        refX="9"
        refY="5"
        markerWidth="7"
        markerHeight="7"
        orient="auto"
      >
        <path d="M0,0 L10,5 L0,10 z" fill={CLAIM} />
      </marker>
    </defs>
  )
}

/**
 * Header button opening a full-screen dialog on the houses' lineage: a tree
 * (one column per house, joined by their marriages and claims) to understand
 * the whole board, and an alliance graph to plan the victory. The focus house
 * and the filters apply to both views.
 */
export function LineageDialog(props: LineageDialogProps) {
  const { t } = useLanguage()
  const [view, setView] = useState<View>('tree')
  const [focus, setFocus] = useState<PlayerId | null>(props.defaultFocus ?? null)
  const [hidden, setHidden] = useState<Set<PlayerId>>(new Set())
  const [showEnded, setShowEnded] = useState(true)
  const [showClaims, setShowClaims] = useState(true)
  const [moves, setMoves] = useState<Record<string, Offset>>({})

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
    () => buildHouses(props.players, nobles, deceased ?? [], fiefs ?? [], claimList ?? []),
    [props.players, nobles, deceased, fiefs, claimList],
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
      (claim) => visible.has(claim.heir) && visible.has(claim.target),
    )
  }, [houses, claimList, showClaims])
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
    <Dialog.Root onOpenChange={(open) => !open && setMoves({})}>
      <Dialog.Trigger asChild>
        <button
          type="button"
          aria-label={t('lineage.open')}
          title={t('lineage.open')}
          className="inline-flex h-8 shrink-0 items-center rounded-lg border border-[#b7a786] bg-[#fffaf0] px-2 text-[#594b3c] transition hover:bg-[#f3ead9] hover:text-[#30291f] focus-visible:ring-2 focus-visible:ring-[#a84632]/40 focus-visible:outline-none"
        >
          <IconHierarchy2 aria-hidden="true" className="size-4" />
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-[#30291f]/50" />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed inset-2 z-50 flex flex-col overflow-hidden rounded-xl border border-[#b7a786] bg-[#fffaf0] shadow-xl sm:inset-6"
        >
          <header className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-[#b7a786]/60 px-4 py-3">
            <Dialog.Title className="font-serif text-lg font-semibold text-[#30291f]">
              {t('lineage.title')}
            </Dialog.Title>
            <div
              role="tablist"
              className="flex rounded-lg border border-[#b7a786] p-0.5 text-sm"
            >
              {(['tree', 'graph'] as const).map((key) => (
                <button
                  key={key}
                  type="button"
                  role="tab"
                  aria-selected={view === key}
                  onClick={() => setView(key)}
                  className={cn(
                    'rounded-md px-3 py-1 font-semibold',
                    view === key
                      ? 'bg-[#a84632] text-white'
                      : 'text-[#594b3c] hover:bg-[#f3ead9]',
                  )}
                >
                  {t(key === 'tree' ? 'lineage.tree' : 'lineage.graph')}
                </button>
              ))}
            </div>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label={t('lineage.close')}
                className="ml-auto rounded-lg p-1.5 text-[#594b3c] hover:bg-[#f3ead9]"
              >
                <IconX aria-hidden="true" className="size-5" />
              </button>
            </Dialog.Close>
            <div className="flex w-full flex-wrap items-center gap-x-4 gap-y-2 text-xs text-[#594b3c]">
              <label className="flex items-center gap-1.5">
                {t('lineage.focus')}
                <select
                  value={focusId ?? ''}
                  onChange={(event) => setFocus(event.target.value || null)}
                  className="rounded-md border border-[#b7a786] bg-[#fffaf0] px-1.5 py-1"
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
                          ? 'border-[#30291f]/40 bg-[#f3ead9]'
                          : 'border-[#b7a786]/60 opacity-50',
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
              {view === 'tree' && (
                <>
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
                  <button
                    type="button"
                    disabled={Object.keys(moves).length === 0}
                    onClick={() => setMoves({})}
                    className="rounded-md border border-[#b7a786] px-2 py-0.5 font-semibold enabled:hover:bg-[#f3ead9] disabled:opacity-40"
                  >
                    {t('lineage.resetLayout')}
                  </button>
                </>
              )}
            </div>
            {props.simulationRequest && (
              <SimulationBar
                simulating={simulating}
                view={view}
                nobles={nobles}
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
          {view === 'tree' ? (
            <LineageTree
              houses={houses}
              links={links}
              claims={claims}
              focus={focusId}
              edit={
                simulating
                  ? { selected: selection, pending, onNoble, onLink }
                  : undefined
              }
              moves={moves}
              onMove={(code, offset) =>
                setMoves((current) => ({ ...current, [code]: offset }))
              }
            />
          ) : (
            <AllianceGraph
              houses={houses}
              links={links}
              focus={focusId}
              onFocus={setFocus}
              scores={scores}
              victory={victory}
            />
          )}
          <VictoryStatusPanel
            players={props.players}
            scores={scores}
            victory={victory}
            simulation={projected}
          />
          <Legend withClaims={view === 'tree'} />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function SimulationBar({
  simulating,
  view,
  nobles,
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
  view: View
  nobles: Noble[]
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
    'rounded-md border border-[#b7a786] bg-[#fffaf0] px-2 py-0.5 text-xs font-semibold enabled:hover:bg-[#f3ead9] disabled:opacity-40'
  if (!simulating) {
    return (
      <div className="flex w-full">
        <button type="button" onClick={onStart} className={cn(button, 'inline-flex items-center gap-1')}>
          <IconFlask aria-hidden="true" className="size-4" />
          {t('sim.start')}
        </button>
      </div>
    )
  }
  const noble = selection?.type === 'noble' ? nobles.find((n) => n.code === selection.code) : undefined
  const alive = noble !== undefined
  const log = (action: SimulationAction) =>
    t(`sim.log.${action.type}` as MessageKey, {
      noble: nameOf(action.noble),
      other: 'other' in action ? nameOf(action.other) : '',
    })
  return (
    <div className="flex w-full flex-col gap-2 rounded-lg border border-[#b8860b]/60 bg-[#f8e8ae]/40 p-2 text-xs">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-semibold text-[#6b4e0a]">{t('sim.banner')}</span>
        <button type="button" className={button} disabled={!canUndo} onClick={onUndo}>
          {t('sim.undo')}
        </button>
        <button type="button" className={button} disabled={!canRedo} onClick={onRedo}>
          {t('sim.redo')}
        </button>
        <button type="button" className={button} disabled={actions.length === 0} onClick={onReset}>
          {t('sim.reset')}
        </button>
        <button type="button" className={cn(button, 'ml-auto')} onClick={onStop}>
          {t('sim.stop')}
        </button>
      </div>
      {view === 'graph' ? (
        <p>{t('sim.graphHint')}</p>
      ) : pending ? (
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
            onClick={() => onAction({ type: 'divorce', noble: selection.a, other: selection.b })}
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
        <p role="alert" className="font-semibold text-[#7a2b1c]">
          {t('sim.rejected', { reason: rejection })}
        </p>
      )}
      {failed && (
        <p role="alert" className="font-semibold text-[#7a2b1c]">
          {t('sim.failed')}
        </p>
      )}
      {loading && <p className="text-[#806f57]">{t('sim.loading')}</p>}
      {actions.length > 0 && (
        <ol className="flex flex-wrap gap-1.5" aria-label={t('sim.panel')}>
          {actions.map((action, index) => (
            <li key={index} className="rounded-full border border-[#b8860b]/60 bg-[#fffaf0] px-2 py-0.5">
              {log(action)}
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}

function Swatch({ kind }: { kind: EdgeKind | 'claim' }) {
  const style =
    kind === 'claim' ? { stroke: CLAIM, width: 2, dash: '2 3' } : EDGE_STYLE[kind]
  return (
    <svg aria-hidden="true" width="28" height="10" className="shrink-0">
      <HatchDefs />
      <line
        x1="1"
        y1="5"
        x2="27"
        y2="5"
        stroke={style.stroke}
        strokeWidth={Math.min(style.width, 8)}
        strokeDasharray={style.dash}
      />
    </svg>
  )
}

function Legend({ withClaims }: { withClaims: boolean }) {
  const { t } = useLanguage()
  const kinds: Array<[EdgeKind | 'claim', MessageKey]> = [
    ['head', 'lineage.legend.head'],
    ['secondary', 'lineage.legend.secondary'],
    ['ended', 'lineage.legend.ended'],
  ]
  if (withClaims) kinds.push(['claim', 'lineage.legend.claim'])
  return (
    <footer className="flex flex-wrap gap-x-4 gap-y-1 border-t border-[#b7a786]/60 px-4 py-2 text-xs text-[#594b3c]">
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

function Badge({
  children,
  className,
  style,
}: {
  children: ReactNode
  className?: string
  style?: React.CSSProperties
}) {
  return (
    <span
      className={cn(
        'rounded border px-1 text-[10px] font-semibold uppercase leading-4',
        className,
      )}
      style={style}
    >
      {children}
    </span>
  )
}

function LineageTree({
  houses,
  links,
  claims,
  focus,
  edit,
  moves,
  onMove,
}: {
  houses: House[]
  links: LineageLink[]
  claims: Claim[]
  focus: PlayerId | null
  edit?: TreeEdit
  moves: Record<string, Offset>
  onMove: (code: string, offset: Offset) => void
}) {
  const { t } = useLanguage()
  const scroller = useRef<HTMLDivElement>(null)
  const drag = useRef<{ code: string; x: number; y: number; from: Offset } | null>(null)
  const claimRefs = useMemo(
    () => claims.map(({ heir, target, spouse }) => ({ heir, target, spouse })),
    [claims],
  )
  const { columns, rows } = useMemo(
    () => layoutColumns(houses, links, focus, claimRefs),
    [houses, links, focus, claimRefs],
  )

  // Automatic position of each card, then the user's own displacement.
  const home = new Map<string, { x: number; y: number }>()
  columns.forEach((column, columnIndex) => {
    column.house.members.forEach((member, index) =>
      home.set(member.code, {
        x: columnIndex * (CARD_W + GAP_X),
        y: HEADER_H + (column.offset + index) * PITCH,
      }),
    )
  })
  const position = new Map<string, { x: number; y: number }>()
  for (const [code, at] of home) {
    const moved = moves[code]
    position.set(code, { x: at.x + (moved?.dx ?? 0), y: at.y + (moved?.dy ?? 0) })
  }
  const width = Math.max(
    columns.length * (CARD_W + GAP_X) - GAP_X,
    ...[...position.values()].map((at) => at.x + CARD_W),
  )
  const height = Math.max(
    HEADER_H + rows * PITCH,
    ...[...position.values()].map((at) => at.y + CARD_H + 8),
  )

  useEffect(() => {
    const element = scroller.current
    if (!element || !focus) return
    const index = columns.findIndex((column) => column.house.player.id === focus)
    if (index < 0) return
    element.scrollLeft = index * (CARD_W + GAP_X) + CARD_W / 2 - element.clientWidth / 2
  }, [focus, columns])

  const segment = (from: { x: number; y: number }, to: { x: number; y: number }) => {
    const { x1, y1, x2, y2 } = marriageSegment(from, to)
    return `M${x1},${y1} L${x2},${y2}`
  }

  const startDrag = (event: ReactPointerEvent<HTMLButtonElement>, code: string) => {
    event.currentTarget.setPointerCapture(event.pointerId)
    drag.current = {
      code,
      x: event.clientX,
      y: event.clientY,
      from: { dx: moves[code]?.dx ?? 0, dy: moves[code]?.dy ?? 0 },
    }
  }
  const dragTo = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const current = drag.current
    const at = current && home.get(current.code)
    if (!current || !at) return
    onMove(current.code, {
      dx: Math.max(-at.x, current.from.dx + event.clientX - current.x),
      dy: Math.max(-at.y, current.from.dy + event.clientY - current.y),
    })
  }

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
          {links.map((link) => {
            const from = position.get(link.a)
            const to = position.get(link.b)
            if (!from || !to) return null
            const style = EDGE_STYLE[link.kind]
            const selected =
              edit?.selected?.type === 'link' &&
              ((edit.selected.a === link.a && edit.selected.b === link.b) ||
                (edit.selected.a === link.b && edit.selected.b === link.a))
            return (
              <g key={link.key}>
                {selected && (
                  <path
                    d={segment(from, to)}
                    fill="none"
                    stroke="#a84632"
                    strokeWidth={style.width + 6}
                    strokeOpacity={0.35}
                  />
                )}
                <path
                  d={segment(from, to)}
                  fill="none"
                  stroke={style.stroke}
                  strokeWidth={style.width}
                  strokeDasharray={style.dash}
                />
                {edit && link.active && (
                  <path
                    d={segment(from, to)}
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
          {claims.map((claim) => {
            const heir = position.get(claim.heir)
            const target = position.get(claim.target)
            const spouse = position.get(claim.spouse)
            if (!heir || !target || !spouse) return null
            // Without a visible marriage line (ended marriages hidden), the
            // filiation points at the target's card instead.
            const shown = links.some(
              (l) =>
                (l.a === claim.target && l.b === claim.spouse) ||
                (l.b === claim.target && l.a === claim.spouse),
            )
            const [start, corner, end] = claimRoute(heir, target, shown ? spouse : target)
            return (
              <path
                key={`claim-${claim.heir}`}
                d={`M${start[0]},${start[1]} L${corner[0]},${corner[1]} L${end[0]},${end[1]}`}
                fill="none"
                stroke={CLAIM}
                strokeWidth={2}
                strokeDasharray="2 3"
                markerEnd="url(#lineage-arrow)"
              />
            )
          })}
        </svg>
        {columns.map((column, columnIndex) => {
          const house = column.house
          return (
            <section
              key={house.player.id}
              aria-label={house.player.name || house.player.id}
              className="pointer-events-none absolute inset-0 z-10"
            >
              <h3
                className={cn(
                  'absolute top-0 flex items-center gap-2 rounded-md px-2 py-1 text-sm font-semibold text-[#30291f]',
                  house.player.id === focus && 'bg-[#f3ead9] ring-1 ring-[#30291f]/30',
                )}
                style={{
                  left: columnIndex * (CARD_W + GAP_X),
                  width: CARD_W,
                  height: HEADER_H - 8,
                }}
              >
                <span
                  aria-hidden="true"
                  className="size-3 rounded-full border border-[#30291f]/30"
                  style={{ backgroundColor: house.player.color }}
                />
                {house.player.name || house.player.id}
              </h3>
              <ol>
                {house.members.map((member, index) => {
                  const at = position.get(member.code)!
                  const memberLinks = links.filter(
                    (l) =>
                      l.kind !== 'ended' && (l.a === member.code || l.b === member.code),
                  )
                  return (
                    <li
                      key={member.code}
                      data-noble={member.code}
                      className={cn(
                        'pointer-events-auto absolute overflow-y-auto rounded-lg border bg-[#f8f0e2] p-2 text-xs',
                        member.dead
                          ? 'border-dashed border-[#b7a786]/70 opacity-60'
                          : 'border-[#b7a786]',
                        moves[member.code] && 'z-20 shadow-md',
                        edit && 'cursor-pointer',
                        edit?.selected?.type === 'noble' &&
                          edit.selected.code === member.code &&
                          'ring-2 ring-[#a84632]',
                        edit?.pending?.from === member.code &&
                          'outline-2 outline-dashed outline-[#7a4fa3]',
                      )}
                      onClick={edit ? () => edit.onNoble(member.code) : undefined}
                      style={{
                        width: CARD_W,
                        height: CARD_H,
                        left: at.x,
                        top: at.y,
                        borderTopColor: house.player.color,
                        borderTopWidth: 4,
                      }}
                    >
                      <div className="flex items-start gap-1.5">
                        <button
                          type="button"
                          aria-label={t('lineage.move', { name: member.name })}
                          title={t('lineage.move', { name: member.name })}
                          onPointerDown={(event) => startDrag(event, member.code)}
                          onPointerMove={dragTo}
                          onPointerUp={() => (drag.current = null)}
                          onPointerCancel={() => (drag.current = null)}
                          className="-ml-1 cursor-grab touch-none px-0.5 leading-none text-[#806f57] active:cursor-grabbing"
                        >
                          ⠿
                        </button>
                        <span className="font-serif text-sm text-[#806f57]">
                          {index + 1}
                        </span>
                        <span
                          className={cn(
                            'min-w-0 flex-1 font-semibold',
                            member.dead && 'line-through',
                          )}
                        >
                          {member.name}
                        </span>
                        {member.sex && (
                          <span
                            aria-label={member.sex}
                            className={
                              member.sex === 'female'
                                ? 'text-[#a84632]'
                                : 'text-[#2f6f9f]'
                            }
                          >
                            {member.sex === 'female' ? '♀' : '♂'}
                          </span>
                        )}
                      </div>
                      <div className="mt-1 flex flex-wrap gap-1">
                        {member.dead && member.cause && (
                          <Badge className="border-[#9a8f7c]">
                            ✝ {t(`lineage.cause.${member.cause}` as MessageKey)}
                          </Badge>
                        )}
                        {member.fiefs.map((fief) => (
                          <Badge key={fief.capital} className="border-[#b7a786]">
                            {titleLabel(t, fief)}
                          </Badge>
                        ))}
                        {member.dignities.map((dignity) => (
                          <Badge
                            key={dignity}
                            className="border-[#815f1e] bg-[#f8e8ae] text-[#6b4e0a]"
                          >
                            {t(`dignity.${dignity}` as MessageKey)}
                          </Badge>
                        ))}
                        {member.claim && (
                          <Badge
                            className="text-white"
                            style={{ backgroundColor: CLAIM, borderColor: CLAIM }}
                          >
                            {t('lineage.claimBadge', {
                              target: member.claim.targetFiefs[0]
                                ? titleLabel(t, member.claim.targetFiefs[0])
                                : member.claim.targetName,
                              rank: member.claim.rank,
                            })}
                          </Badge>
                        )}
                        {member.claimedBy > 0 && (
                          <Badge className="border-[#7a4fa3] text-[#7a4fa3]">
                            {t('lineage.claimedBadge', { count: member.claimedBy })}
                          </Badge>
                        )}
                        {memberLinks.map((link) => (
                          <Badge
                            key={link.key}
                            className={
                              link.kind === 'secondary' ? 'text-[#30291f]' : 'text-white'
                            }
                            style={{
                              backgroundColor:
                                link.kind === 'secondary' ? '#dfe3e7' : GOLD,
                              borderColor: link.kind === 'secondary' ? SILVER : GOLD,
                            }}
                          >
                            {t('lineage.marriageBadge', {
                              kind: t(kindLabel(link)),
                              weight: link.weight ?? 0,
                            })}
                            {link.activeHeadFor.length > 0 ? ' ★' : ''}
                          </Badge>
                        ))}
                      </div>
                    </li>
                  )
                })}
              </ol>
            </section>
          )
        })}
      </div>
    </div>
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
    <div className="grid min-h-0 flex-1 gap-4 overflow-auto p-4 lg:grid-cols-[minmax(0,28rem)_1fr]">
      <svg
        viewBox="0 0 400 330"
        role="img"
        aria-label={t('lineage.graph')}
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
                strokeWidth={style.width + 1}
                strokeDasharray={style.dash}
                strokeLinecap={style.dash ? 'butt' : 'round'}
              />
              <text
                x={(a.x + b.x) / 2}
                y={(a.y + b.y) / 2 - 6}
                textAnchor="middle"
                className="fill-[#30291f] text-[10px] font-bold"
                stroke="#fffaf0"
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
                stroke={isFocus ? '#30291f' : '#fffaf0'}
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
                className="fill-[#30291f] text-[11px] font-semibold"
              >
                {house.player.name || house.player.id}
              </text>
            </g>
          )
        })}
      </svg>
      <div className="grid content-start gap-3">
        <h3 className="font-serif text-base font-semibold text-[#30291f]">
          {t('lineage.alliances')}
        </h3>
        {pairs.length === 0 && (
          <p className="text-sm text-[#806f57]">{t('lineage.noAlliance')}</p>
        )}
        <ul className="grid gap-2">
          {pairs.map((pair) => {
            const carrier = pair.carrier
            const others = pair.links.filter((link) => link !== carrier)
            return (
              <li
                key={pair.key}
                className={cn(
                  'rounded-lg border bg-[#f8f0e2] p-2.5 text-sm',
                  focus && pair.houses.includes(focus)
                    ? 'border-[#30291f]'
                    : 'border-[#b7a786]/60',
                  dim(pair) && 'opacity-50',
                )}
              >
                <div className="flex flex-wrap items-center gap-2 font-semibold">
                  <Swatch kind={pair.kind} />
                  {houseName(pair.houses[0])} ⟷ {houseName(pair.houses[1])}
                  <span className="text-xs font-normal text-[#806f57]">
                    {t(kindLabel(carrier))} ·{' '}
                    {t('lineage.weight', { weight: pair.weight })}
                  </span>
                </div>
                <dl className="mt-1.5 grid gap-1 text-xs">
                  <div>
                    <dt className="inline text-[#806f57]">{t('lineage.toCancel')} : </dt>
                    <dd className="inline font-medium">
                      {marriageName(carrier.a, carrier.b)}
                    </dd>
                  </div>
                  {carrier.headSuccessors.map((successor) => (
                    <div key={successor.player}>
                      <dt className="inline text-[#806f57]">
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
                      <dt className="inline text-[#806f57]">
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
                  className="size-3 rounded-full border border-[#30291f]/30"
                  style={{ backgroundColor: house.player.color }}
                />
                <span className="font-semibold">
                  {house.player.name || house.player.id}
                </span>
                <span className="font-serif text-[#a84632]">
                  {scores?.[house.player.id]?.total ?? 0}
                  {goal ? ` / ${goal.required}` : ''}
                </span>
                {goal && (
                  <span className="rounded border border-[#b8860b] bg-[#f8e8ae] px-1.5 text-[10px] font-bold uppercase text-[#6b4e0a]">
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
