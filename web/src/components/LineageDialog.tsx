import { memo, useCallback, type MutableRefObject, useEffect, useMemo, useRef, useState } from 'react'
import {
  IconChevronDown,
  IconChevronUp,
  IconFlask,
  IconHierarchy2,
  IconX,
} from '@tabler/icons-react'
import { Dialog } from 'radix-ui'

import { ShieldIcon } from '@/components/ArmsIcons'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { playerArms, SHIELD_PATH } from '@/lib/heraldry'
import {
  buildAlliancePairs,
  buildHouses,
  buildLinks,
  countLines,
  layoutTree,
  MEDALLION_W,
  placeTree,
  visibleNobles,
  wrapLines,
  type AlliancePair,
  type ClaimRef,
  type EdgeKind,
  type House,
  type HouseMember,
  type LineageLink,
  type TreeBox,
} from '@/lib/lineage'
import {
  affectedFiefs,
  exiledFiefs,
  fiefOutcomes,
  hasClaimedHolding,
  killVariants,
  sameOutcomes,
  usurpedFiefs,
  type FiefOutcome,
  type KillVariant,
} from '@/lib/kill-variants'
import { playerDisplayName } from '@/lib/player-label'
import { simulationLog, useSimulationHistory } from '@/lib/simulation-history'
import { useActionPreviews, type ActionPreviewCache } from '@/lib/use-action-previews'
import { useElementSize } from '@/lib/use-element-size'
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
  endAngle,
  ivyLeaves,
  pathPoints,
  roundedPath,
  type Point,
} from '@/lib/lineage-art'
import {
  LINEAGE_BAND,
  LINEAGE_BRASS,
  LINEAGE_BRASS_LIGHT,
  LINEAGE_RANK_DIGIT,
  LINEAGE_CLOTH,
  LINEAGE_ENDED,
  LINEAGE_HEAD_BROWN,
  LINEAGE_INK,
  LINEAGE_IVY,
  LINEAGE_PAPER,
  LINEAGE_SCROLL_SHADE,
  LINEAGE_SECONDARY,
  LINEAGE_WEAVE,
  MAP_INK,
  MAP_PAPER,
  NEUTRAL_ARMY_COLOR,
  SELECTION_HALO,
  SELECTION_STROKE,
} from '@/lib/map-palette'

/**
 * Stroke of a marriage by kind. The head marriage is a brown outlined bar
 * filled with gold, grown over with ivy; the secondary one is a double line (a
 * wide ink stroke with a thin line in the colour of the ground over it, a null
 * inner colour); an ended one is hatched.
 */
const EDGE_STYLE: Record<
  EdgeKind,
  { stroke: string; width: number; inner: { color: string | null; width: number } | null }
> = {
  head: {
    stroke: LINEAGE_HEAD_BROWN,
    width: 8,
    inner: { color: LINEAGE_BRASS_LIGHT, width: 4.5 },
  },
  secondary: {
    stroke: LINEAGE_SECONDARY,
    width: 5,
    inner: { color: null, width: 1.5 },
  },
  ended: { stroke: 'url(#lineage-hatch)', width: 5, inner: null },
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
  /** `garrison` kills `from` once the noble garrisoning the capital is clicked. */
  kind: 'marry' | 'claim' | 'garrison'
  from: string
}

/** Wiring of the tree's cards and marriage lines to the simulation. */
interface TreeEdit {
  selected: Selection | null
  pending: Pending | null
  onNoble: (code: string) => void
  onLink: (a: string, b: string) => void
}

type EdgeShape = { d: string } | { x1: number; y1: number; x2: number; y2: number }

/**
 * A marriage drawn as its main stroke and, when its kind has one, the thin
 * inner stroke over it; `ground` is the colour of what the bond lies on, which
 * the inner line of a secondary marriage takes.
 */
function EdgeStroke({
  kind,
  shape,
  ground = LINEAGE_PAPER,
}: {
  kind: EdgeKind
  shape: EdgeShape
  ground?: string
}) {
  const style = EDGE_STYLE[kind]
  const draw = (stroke: string, strokeWidth: number) =>
    'd' in shape ? (
      <path d={shape.d} fill="none" stroke={stroke} strokeWidth={strokeWidth} />
    ) : (
      <line {...shape} stroke={stroke} strokeWidth={strokeWidth} />
    )
  return (
    <>
      {draw(style.stroke, style.width)}
      {style.inner && draw(style.inner.color ?? ground, style.inner.width)}
    </>
  )
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
        <line x1="0" y1="0" x2="0" y2="6" stroke={LINEAGE_ENDED} strokeWidth="3" />
      </pattern>
    </defs>
  )
}

/** Margin of the tapestry border around the tree, in pixels. */
const FRAME_PAD = 36
/** Spread of a leaf at the end of a claim's vine. */
const LEAF = 'M0,0 C-3,-5 -11,-5 -16,0 C-11,5 -3,5 0,0 Z'

/** Cloth, woven threads and border of the tapestry the tree is drawn on. */
function TapestryFrame({ width, height }: { width: number; height: number }) {
  const w = width + 2 * FRAME_PAD
  const h = height + 2 * FRAME_PAD
  const band = (inset: number) =>
    `M${inset},${inset} H${w - inset} V${h - inset} H${inset} Z`
  const corners = [
    [10, 10],
    [w - 10, 10],
    [10, h - 10],
    [w - 10, h - 10],
  ]
  return (
    <svg
      aria-hidden="true"
      data-testid="lineage-tapestry"
      className="pointer-events-none absolute inset-0"
      width={w}
      height={h}
    >
      <defs>
        <pattern id="lineage-weave" patternUnits="userSpaceOnUse" width="8" height="8">
          <rect width="8" height="8" fill={LINEAGE_CLOTH} />
          <path
            d="M0,1.5 H4 M4,5.5 H8 M1.5,4 V8 M5.5,0 V4"
            stroke={LINEAGE_WEAVE}
            strokeWidth="1.6"
            fill="none"
          />
        </pattern>
      </defs>
      <rect width={w} height={h} fill="url(#lineage-weave)" />
      <path
        d={`${band(3)} ${band(17)}`}
        fillRule="evenodd"
        fill={LINEAGE_BAND}
        stroke={LINEAGE_INK}
        strokeWidth="1.5"
      />
      <path
        d={band(10)}
        fill="none"
        stroke={LINEAGE_ENDED}
        strokeWidth="1"
        strokeDasharray="5 3"
      />
      <path d={band(21)} fill="none" stroke={LINEAGE_BRASS_LIGHT} strokeWidth="1" />
      <path d={band(17)} fill="none" stroke={LINEAGE_BRASS} strokeWidth="1.5" />
      {corners.map(([x, y]) => (
        <g key={`${x}-${y}`} transform={`translate(${x} ${y})`}>
          <circle r="8.5" fill={LINEAGE_CLOTH} stroke={LINEAGE_BRASS} strokeWidth="1.5" />
          {[
            [0, -3.6],
            [3.6, 0],
            [0, 3.6],
            [-3.6, 0],
          ].map(([cx, cy]) => (
            <circle
              key={`${cx}${cy}`}
              cx={cx}
              cy={cy}
              r="2.6"
              fill={LINEAGE_BRASS_LIGHT}
              stroke={LINEAGE_INK}
              strokeWidth="0.8"
            />
          ))}
          <circle r="1.3" fill={LINEAGE_INK} />
        </g>
      ))}
    </svg>
  )
}

/** Small pointed ivy leaf, rooted at the origin and pointing along +x. */
const IVY_LEAF = 'M0,0 C1,-2.4 3.6,-3 6,0 C3.6,3 1,2.4 0,0Z'

/**
 * Ivy growing on the gold bar of a head marriage: many small leaves on both
 * sides of the stem, rooted at its edge and tilted forward. The shape (not the
 * colour alone) tells the head marriage from the others.
 */
function IvyLeaves({ points, radius = 10 }: { points: Point[]; radius?: number }) {
  const edge = EDGE_STYLE.head.width / 2 - 0.5
  return (
    <g data-ivy="">
      {ivyLeaves(points, radius).map((leaf, i) => {
        const nx = -Math.sin(leaf.angle) * leaf.side * edge
        const ny = Math.cos(leaf.angle) * leaf.side * edge
        return (
          <path
            key={i}
            d={IVY_LEAF}
            transform={`translate(${leaf.x + nx} ${leaf.y + ny}) rotate(${(leaf.angle * 180) / Math.PI + leaf.side * (50 + leaf.tilt)})`}
            fill={LINEAGE_IVY}
          />
        )
      })}
    </g>
  )
}

/**
 * The knot tying two cartouches: a ring in the colour of the bond (brown and gold
 * for the head marriage, ink for a secondary one); an ended marriage has a broken ring
 * crossed by a slash.
 */
function MarriageKnot({
  x,
  y,
  kind,
  fill = LINEAGE_CLOTH,
}: {
  x: number
  y: number
  kind: EdgeKind
  fill?: string
}) {
  const ended = kind === 'ended'
  const color = ended ? LINEAGE_ENDED : EDGE_STYLE[kind].stroke
  return (
    <g data-knot={kind} transform={`translate(${x} ${y})`}>
      <circle
        r="6.5"
        fill={fill}
        stroke={color}
        strokeWidth="2"
        strokeDasharray={ended ? '4 3' : undefined}
      />
      {ended ? (
        <line x1="-4.5" y1="4.5" x2="4.5" y2="-4.5" stroke={color} strokeWidth="1.5" />
      ) : (
        <circle r="2.3" fill={kind === 'head' ? LINEAGE_BRASS_LIGHT : color} />
      )}
    </g>
  )
}

/** Where the knot of a marriage sits: the middle of its line, or of its arch. */
function knotPosition(points: Point[], aligned: boolean) {
  if (points.length < 2) return null
  const [first, last] = [points[0], points[points.length - 1]]
  if (aligned || points.length < 4)
    return { x: (first.x + last.x) / 2, y: (first.y + last.y) / 2 }
  return { x: (points[1].x + points[2].x) / 2, y: points[1].y }
}

/**
 * Header button opening a full-screen dialog on the houses' lineage: a
 * genealogical tree (couples side by side, the heirs of a claim hung under
 * their couple) to understand the whole board, with the alliance graph as an
 * inset to plan the victory.
 */
export function LineageDialog(props: LineageDialogProps) {
  const { t } = useLanguage()
  const [focus, setFocus] = useState<PlayerId | null>(props.defaultFocus ?? null)
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
  const previewCache = useRef<ActionPreviewCache>(new Map())
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
    previewCache.current.clear()
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
  const { push: pushAction } = history
  const onNoble = useCallback(
    (code: string) => {
      setRejection(null)
      if (pending) {
        if (code === pending.from) return
        pushAction(
          pending.kind === 'garrison'
            ? { type: 'kill', noble: pending.from, garrison: code }
            : { type: pending.kind, noble: pending.from, other: code },
        )
        setSelection(null)
        setPending(null)
        return
      }
      setSelection({ type: 'noble', code })
    },
    [pending, pushAction],
  )
  const onLink = useCallback(
    (a: string, b: string) => {
      if (pending) return
      setRejection(null)
      setSelection({ type: 'link', a, b })
    },
    [pending],
  )
  const edit = useMemo<TreeEdit | undefined>(
    () => (simulating ? { selected: selection, pending, onNoble, onLink } : undefined),
    [simulating, selection, pending, onNoble, onLink],
  )
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
      buildHouses(props.players, nobles, deceased ?? [], fiefs ?? [], claimList ?? [], {
        bishoprics: data.bishoprics,
        cardinals: data.cardinals,
        pope: data.pope,
        king: data.king,
        queen: data.queen,
        queenDowager: data.queenDowager,
        excommunicated: data.excommunicated,
      }),
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
    ],
  )
  const allLinks = useMemo(
    () => buildLinks(allHouses, marriages ?? []),
    [allHouses, marriages],
  )
  const focusId =
    focus && allHouses.some((house) => house.player.id === focus) ? focus : null
  // With a focus, the tree keeps that house and the nobles directly tied to it.
  const visible = useMemo(
    () => visibleNobles(allHouses, allLinks, claimList ?? [], focusId),
    [allHouses, allLinks, claimList, focusId],
  )
  const houses = useMemo(
    () =>
      visible
        ? allHouses
            .map((house) => ({
              ...house,
              members: house.members.filter((member) => visible.has(member.code)),
            }))
            .filter((house) => house.members.length > 0)
        : allHouses,
    [allHouses, visible],
  )
  const ranks = useMemo(() => {
    const byCode = new Map<string, number>()
    for (const house of allHouses)
      house.members.forEach((member, index) => byCode.set(member.code, index + 1))
    return byCode
  }, [allHouses])
  const links = useMemo(
    () =>
      visible
        ? allLinks.filter((link) => visible.has(link.a) && visible.has(link.b))
        : allLinks,
    [allLinks, visible],
  )
  const claims = useMemo(
    () =>
      (claimList ?? []).filter(
        (claim) => !visible || (visible.has(claim.heir) && visible.has(claim.target)),
      ),
    [claimList, visible],
  )
  const playerName = (house: House) => house.player.name || house.player.id

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
                  {allHouses.map((house) => (
                    <option key={house.player.id} value={house.player.id}>
                      {playerName(house)}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            {props.simulationRequest && (
              <SimulationBar
                simulating={simulating}
                nobles={nobles}
                fiefs={fiefs ?? NO_FIEFS}
                claims={claimList ?? NO_CLAIMS}
                players={props.players}
                request={props.simulationRequest ?? null}
                previewCache={previewCache}
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
              ranks={ranks}
              focus={focusId}
              edit={edit}
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
              houses={allHouses}
              links={allLinks}
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

const NO_FIEFS: Fief[] = []
const NO_CLAIMS: Claim[] = []

function SimulationBar({
  simulating,
  nobles,
  fiefs,
  claims,
  players,
  request,
  previewCache,
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
  claims: Claim[]
  players: Player[]
  request: VictorySimulationRequest | null
  previewCache: MutableRefObject<ActionPreviewCache>
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
  const log = (action: SimulationAction) => {
    const line = simulationLog(action)
    return t(line.key as MessageKey, {
      noble: line.noble && nameOf(line.noble),
      other: line.other && nameOf(line.other),
      fief: line.fief,
    })
  }
  const ownerName = (id: string) => playerDisplayName(id, [players], id)
  const ownExiledFiefs = noble ? exiledFiefs(noble.code, fiefs) : []
  const contestedFiefs = noble
    ? [...ownExiledFiefs, ...usurpedFiefs(noble.code, fiefs)]
    : []
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
            {t(
              pending.kind === 'marry'
                ? 'sim.pickSpouse'
                : pending.kind === 'claim'
                  ? 'sim.pickTarget'
                  : 'sim.pickGarrison',
              { name: nameOf(pending.from) },
            )}
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
          <KillOptions
            code={selection.code}
            alive={alive}
            nobles={nobles}
            fiefs={fiefs}
            claims={claims}
            request={request}
            actions={actions}
            previewCache={previewCache}
            nameOf={nameOf}
            buttonClass={button}
            onAction={onAction}
            onPending={onPending}
          />
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
          {noble && (
            <select
              aria-label={t('sim.conquer')}
              className={button}
              disabled={noble.status !== 'free'}
              value=""
              onChange={(event) => {
                if (event.target.value)
                  onAction({
                    type: 'conquer',
                    noble: noble.code,
                    fief: event.target.value,
                  })
              }}
            >
              <option value="">{t('sim.conquer')}</option>
              {fiefs
                .filter((fief) => fief.owner !== noble.owner)
                .map((fief) => (
                  <option key={fief.capital} value={fief.capital}>
                    {t('sim.conquerOption', {
                      fief: fief.capital,
                      owner: ownerName(fief.owner),
                    })}
                  </option>
                ))}
            </select>
          )}
          {ownExiledFiefs.map((fief) => (
            <button
              key={`restore-${fief.capital}`}
              type="button"
              className={button}
              disabled={
                nobles.find((n) => n.code === fief.holder)?.owner === noble?.owner
              }
              onClick={() => onAction({ type: 'restore', fief: fief.capital })}
            >
              {t('sim.restore', { fief: fief.capital })}
            </button>
          ))}
          {contestedFiefs.map((fief) => (
            <button
              key={`renounce-${fief.capital}`}
              type="button"
              className={button}
              onClick={() =>
                onAction({
                  type: 'renounce_title',
                  noble: selection.code,
                  fief: fief.capital,
                })
              }
            >
              {t('sim.renounceTitle', { fief: fief.capital })}
            </button>
          ))}
          {claims.some((claim) => claim.heir === selection.code && !claim.ended) && (
            <button
              type="button"
              className={button}
              onClick={() => onAction({ type: 'renounce_claim', noble: selection.code })}
            >
              {t('sim.renounceClaim')}
            </button>
          )}
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

/**
 * The ways a noble can die: each option is one button, with the fiefs it
 * settles previewed from the server's projection of that option.
 */
function KillOptions({
  code,
  alive,
  nobles,
  fiefs,
  claims,
  request,
  actions,
  previewCache,
  nameOf,
  buttonClass,
  onAction,
  onPending,
}: {
  code: string
  alive: boolean
  nobles: Noble[]
  fiefs: Fief[]
  claims: Claim[]
  request: VictorySimulationRequest | null
  actions: SimulationAction[]
  previewCache: MutableRefObject<ActionPreviewCache>
  nameOf: (code: string) => string
  buttonClass: string
  onAction: (action: SimulationAction) => void
  onPending: (pending: Pending) => void
}) {
  const { t } = useLanguage()
  const variants = useMemo(
    () => killVariants(code, nobles, fiefs, claims),
    [code, nobles, fiefs, claims],
  )
  const settled = useMemo(() => affectedFiefs(code, fiefs), [code, fiefs])
  const previewed = useMemo(() => (settled.length > 0 ? variants : []), [settled, variants])
  const previewStates = useActionPreviews(request, actions, previewed, previewCache)
  const outcomesOf = (variant: KillVariant): FiefOutcome[] | undefined => {
    const state = previewStates[variant.key]
    if (!state) return undefined
    const capitals = new Set(settled.map((fief) => fief.capital))
    if (variant.partner)
      for (const fief of affectedFiefs(variant.partner, fiefs)) capitals.add(fief.capital)
    return fiefOutcomes(state, [...capitals], nameOf)
  }
  const describeOutcomes = (outcomes: FiefOutcome[]) =>
    outcomes
      .map(
        (outcome) =>
          t('sim.preview.fief', {
            fief: outcome.fief,
            holder: outcome.holder ?? t('sim.preview.vacant'),
          }) + (outcome.exile ? t('sim.preview.exile', { name: outcome.exile }) : ''),
      )
      .join(' ; ')
  const plainOutcomes = outcomesOf(variants[0])
  const label = (variant: KillVariant) =>
    variant.key === 'plain'
      ? t('sim.kill')
      : variant.partner
        ? t('sim.killTogether', { name: nameOf(variant.partner) })
        : t('sim.killNoGarrison')
  return (
    <>
      {variants.map((variant) => {
        const outcomes = outcomesOf(variant)
        // Without a garrison the death reads like the plain one: no option.
        if (
          variant.action.type === 'kill' &&
          variant.action.noGarrison &&
          (!outcomes || (plainOutcomes && sameOutcomes(outcomes, plainOutcomes)))
        )
          return null
        const text = label(variant)
        const previewId = `sim-preview-${variant.key.replace(/\W/g, '-')}`
        return (
          <button
            key={variant.key}
            type="button"
            className={cn(buttonClass, outcomes && 'inline-flex flex-col items-start')}
            disabled={!alive}
            aria-label={text}
            aria-describedby={outcomes ? previewId : undefined}
            onClick={() => onAction(variant.action)}
          >
            {text}
            {outcomes && (
              <span id={previewId} className="font-normal text-ink-500">
                {describeOutcomes(outcomes)}
              </span>
            )}
          </button>
        )
      })}
      {hasClaimedHolding(code, fiefs, claims) && (
        <button
          type="button"
          className={buttonClass}
          onClick={() => onPending({ kind: 'garrison', from: code })}
        >
          {t('sim.killGarrison')}
        </button>
      )}
    </>
  )
}

function Swatch({ kind }: { kind: EdgeKind | 'claim' | 'noble' }) {
  if (kind === 'noble')
    return (
      <svg aria-hidden="true" width="44" height="16" className="shrink-0">
        <g transform="scale(0.55)">
          <ScrollShape w={80} h={29} color={LINEAGE_BRASS} />
        </g>
        <path
          d={SHIELD_PATH}
          transform="translate(11 8) scale(0.5)"
          fill={LINEAGE_BRASS}
          stroke={LINEAGE_INK}
          strokeWidth="1.2"
        />
      </svg>
    )
  return (
    <svg
      aria-hidden="true"
      width="48"
      height="22"
      viewBox="-6 -4 48 22"
      className="shrink-0 overflow-visible"
    >
      <HatchDefs />
      {kind === 'claim' ? (
        <>
          <path
            d="M1,11 C12,11 14,4 26,4"
            fill="none"
            stroke={LINEAGE_SECONDARY}
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeDasharray="1 4"
          />
          <path d={LEAF} transform="translate(35 4)" fill={LINEAGE_SECONDARY} />
        </>
      ) : (
        <>
          <EdgeStroke kind={kind} shape={{ x1: 0, y1: 7, x2: 36, y2: 7 }} />
          {kind === 'head' && (
            <IvyLeaves
              points={[
                { x: 0, y: 7 },
                { x: 36, y: 7 },
              ]}
            />
          )}
          <MarriageKnot x={18} y={7} kind={kind} fill={LINEAGE_PAPER} />
        </>
      )}
    </svg>
  )
}

function Legend() {
  const { t } = useLanguage()
  const kinds: Array<[EdgeKind | 'claim' | 'noble', MessageKey]> = [
    ['noble', 'lineage.legend.noble'],
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

/** Characters of a name that fit beside the blazon of a cartouche before it wraps. */
const NAME_LINE_CHARS = 15

/** The titles, standing and deaths of a member, one text line each. */
function medallionTexts(member: HouseMember, t: Translate): MedallionLine[] {
  const lines: MedallionLine[] = []
  if (member.dead && member.cause)
    lines.push({
      key: 'cause',
      text: `✝ ${t(`lineage.cause.${member.cause}` as MessageKey)}`,
    })
  if (member.status && member.status !== 'free')
    lines.push({
      key: 'status',
      text: t(`orders.nobleStatus.${member.status}` as MessageKey),
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
    lines.push({
      key: `fief-${fief.capital}`,
      text: member.usurped.includes(fief.capital)
        ? `${titleLabel(t, fief)} · ${t('lineage.usurper')}`
        : titleLabel(t, fief),
    })
  for (const fief of member.exiled)
    lines.push({
      key: `exiled-${fief.capital}`,
      alert: true,
      text: `${t(`fief.exiledTitle.${fief.title}` as MessageKey)} · ${fief.capital}`,
    })
  for (const dignity of member.dignities)
    lines.push({ key: `dignity-${dignity}`, text: t(`dignity.${dignity}` as MessageKey) })
  if (member.claim?.outcome)
    lines.push({
      key: 'claim-outcome',
      text: t(
        member.claim.outcome === 'honoured'
          ? 'lineage.claimHonoured'
          : member.claim.outcome === 'renounced'
            ? 'lineage.claimRenounced'
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
  ranks,
  focus,
  edit,
}: {
  houses: House[]
  links: LineageLink[]
  claims: Claim[]
  /** Succession rank of each living noble in its whole house, whoever is shown. */
  ranks: Map<string, number>
  focus: PlayerId | null
  edit?: TreeEdit
}) {
  const { t } = useLanguage()
  const scroller = useRef<HTMLDivElement>(null)
  const armsOf = useMemo(
    () => new Map(houses.map((house) => [house.player.id, playerArms(house.player)])),
    [houses],
  )
  const claimRefs = useMemo<ClaimRef[]>(
    () =>
      claims.map(({ heir, target, spouse, ended, turn }) => ({
        heir,
        target,
        spouse,
        ended,
        turn,
      })),
    [claims],
  )
  const layout = useMemo(
    () => layoutTree(houses, links, focus, claimRefs),
    [houses, links, focus, claimRefs],
  )
  // The text lines of every member, and how many lines past the first its
  // cartouche takes (a wrapped name, then the written titles).
  const { texts, extraLines } = useMemo(() => {
    const texts = new Map<string, MedallionLine[]>()
    const extraLines = new Map<string, number>()
    for (const house of houses)
      for (const member of house.members) {
        const lines = medallionTexts(member, t)
        texts.set(member.code, lines)
        extraLines.set(
          member.code,
          wrapLines(`${member.name}${member.sex ? ' ♂' : ''}`, NAME_LINE_CHARS) -
            1 +
            countLines(lines.map((line) => line.text)),
        )
      }
    return { texts, extraLines }
  }, [houses, t])
  const placement = useMemo(
    () => placeTree(layout, (code) => extraLines.get(code) ?? 0),
    [layout, extraLines],
  )
  const { width, height, boxes } = placement

  // The frame follows the focus (and the opening), not the edits of a simulation:
  // the latest layout is read from a ref so a new one never moves the scroll.
  const latest = useRef({ houses, layout, boxes })
  useEffect(() => {
    latest.current = { houses, layout, boxes }
  })
  useEffect(() => {
    const element = scroller.current
    if (!element || !focus) return
    const { houses, layout, boxes } = latest.current
    const house = houses.find((entry) => entry.player.id === focus)
    const lefts = (house?.members ?? [])
      .map((member) => layout.x.get(member.code))
      .filter((left): left is number => left !== undefined)
    if (lefts.length === 0) return
    const first = Math.min(...lefts)
    const last = Math.max(...lefts) + MEDALLION_W
    // The house's cards centred, or its first card at the edge when they do not fit.
    element.scrollLeft = Math.min(
      FRAME_PAD + (first + last) / 2 - element.clientWidth / 2,
      FRAME_PAD + first - 24,
    )
    // The house's first row at the top, unless it already shows without scrolling.
    const tops = (house?.members ?? []).map(
      (member) => boxes.get(member.code)?.y ?? Infinity,
    )
    const top = FRAME_PAD + Math.min(...tops) - 24
    element.scrollTop = Number.isFinite(top) && top > element.clientHeight - 160 ? top : 0
  }, [focus])

  const isSelected = (link: LineageLink) =>
    edit?.selected?.type === 'link' &&
    ((edit.selected.a === link.a && edit.selected.b === link.b) ||
      (edit.selected.a === link.b && edit.selected.b === link.a))

  return (
    <div ref={scroller} className="min-h-0 flex-1 overflow-auto p-4">
      <div
        className="relative mx-auto"
        style={{ width: width + 2 * FRAME_PAD, height: height + 2 * FRAME_PAD }}
      >
        <TapestryFrame width={width} height={height} />
        <div
          className="absolute"
          style={{ left: FRAME_PAD, top: FRAME_PAD, width, height }}
        >
          <svg
            aria-hidden={edit ? undefined : true}
            className="pointer-events-none absolute inset-0 z-0"
            width={width}
            height={height}
          >
            <HatchDefs />
            {placement.claims.map(({ claim: index, d }) => {
              const claim = claims[index]
              const points = pathPoints(d)
              const end = points[points.length - 1]
              const angle = (endAngle(points) * 180) / Math.PI
              const ended = Boolean(claim.ended)
              return (
                <g
                  key={`claim-${index}-${claim.heir}-${claim.target}`}
                  opacity={ended ? 0.45 : 1}
                >
                  <path
                    data-claim={`${claim.heir}>${claim.target}`}
                    data-ended={ended}
                    d={roundedPath(points, 28)}
                    fill="none"
                    stroke={LINEAGE_SECONDARY}
                    strokeWidth={1.8}
                    strokeLinecap="round"
                    strokeDasharray="1 4"
                  />
                  {end && (
                    <path
                      d={LEAF}
                      transform={`translate(${end.x} ${end.y}) rotate(${angle})`}
                      fill={ended ? LINEAGE_CLOTH : LINEAGE_SECONDARY}
                      stroke={LINEAGE_SECONDARY}
                      strokeWidth={1.2}
                      strokeLinejoin="round"
                    />
                  )}
                </g>
              )
            })}
            {placement.marriages.map(({ link, aligned, d: edge }, position) => {
              const points = pathPoints(edge)
              const d = roundedPath(points, 10)
              const knot = knotPosition(points, aligned)
              return (
                <g
                  key={`${position}-${link.key}`}
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
                  <EdgeStroke kind={link.kind} shape={{ d }} ground={LINEAGE_CLOTH} />
                  {link.kind === 'head' && <IvyLeaves points={points} radius={10} />}
                  {knot && <MarriageKnot x={knot.x} y={knot.y} kind={link.kind} />}
                  {edit && link.active && (
                    <path
                      d={d}
                      data-testid={`link-${link.a}-${link.b}`}
                      role="button"
                      aria-label={`${link.a} - ${link.b}`}
                      fill="none"
                      stroke="transparent"
                      strokeWidth={16}
                      tabIndex={0}
                      aria-pressed={isSelected(link)}
                      className="outline-none focus-visible:stroke-brass-700/60"
                      style={{ pointerEvents: 'stroke', cursor: 'pointer' }}
                      onClick={() => edit.onLink(link.a, link.b)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          edit.onLink(link.a, link.b)
                        }
                      }}
                    />
                  )}
                </g>
              )
            })}
          </svg>
          {houses.map((house) => (
            <section
              key={house.player.id}
              aria-label={house.player.name || house.player.id}
              data-focused={house.player.id === focus}
              className="pointer-events-none absolute inset-0 z-10"
            >
              <ol>
                {house.members.map((member, index) => (
                  <Cartouche
                    key={member.code}
                    member={member}
                    rank={ranks.get(member.code) ?? index + 1}
                    arms={armsOf.get(house.player.id)!}
                    color={house.player.color || NEUTRAL_ARMY_COLOR}
                    box={boxes.get(member.code)!}
                    lines={texts.get(member.code) ?? []}
                    edit={edit}
                  />
                ))}
              </ol>
            </section>
          ))}
        </div>
      </div>
    </div>
  )
}

/** Rolls of a scroll: the width of each curled end. */
const ROLL = 18
/** Heading of a cartouche: the blazon, then the name beside it. */
const CARTOUCHE_PAD_X = ROLL + 8

/** One curled end of a scroll, `x` from its left edge. */
function ScrollRoll({ x, h, dash }: { x: number; h: number; dash?: string }) {
  const half = ROLL / 2
  return (
    <g transform={`translate(${x} 0)`}>
      <path
        d={`M0,5 A${half},4 0 0 1 ${ROLL},5 V${h - 5} A${half},4 0 0 1 0,${h - 5} Z`}
        fill={LINEAGE_BAND}
        stroke={LINEAGE_INK}
        strokeWidth="1.4"
        strokeDasharray={dash}
      />
      <ellipse
        cx={half}
        cy="5"
        rx={half}
        ry="4"
        fill={LINEAGE_SCROLL_SHADE}
        stroke={LINEAGE_INK}
        strokeWidth="1.2"
      />
      <path
        d={`M${half - 4},5 A4,2 0 1 1 ${half + 1},6.6 M${half},5 H${half + 4}`}
        fill="none"
        stroke={LINEAGE_INK}
        strokeWidth="0.8"
      />
      <path
        d={`M${half - 3},${h - 5} V10 M${half + 3.5},${h - 6} V12`}
        fill="none"
        stroke={LINEAGE_SCROLL_SHADE}
        strokeWidth="1.2"
        strokeLinecap="round"
      />
    </g>
  )
}

/**
 * The parchment scroll of a noble: a sheet unrolled between two curled ends,
 * inked at its edges, with a rule in the house colour along its top and bottom.
 */
function ScrollShape({
  w,
  h,
  color,
  dead = false,
}: {
  w: number
  h: number
  color: string
  dead?: boolean
}) {
  const half = ROLL / 2
  const dash = dead ? '4 3' : undefined
  return (
    <g opacity={dead ? 0.65 : 1}>
      <path
        d={`M${half},6 H${w - half} V${h - 6} H${half} Z`}
        fill={LINEAGE_PAPER}
        stroke={LINEAGE_INK}
        strokeWidth="1.4"
        strokeDasharray={dash}
      />
      <path
        d={`M${ROLL + 4},9.5 H${w - ROLL - 4} M${ROLL + 4},${h - 9.5} H${w - ROLL - 4}`}
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
      />
      <ScrollRoll x={0} h={h} dash={dash} />
      <ScrollRoll x={w - ROLL} h={h} dash={dash} />
    </g>
  )
}

/**
 * A noble: a parchment scroll carrying the house blazon in the colours of the
 * player, the name, the titles written beneath, and the succession rank on a
 * seal in the colour of the house.
 */
const Cartouche = memo(function Cartouche({
  member,
  rank,
  arms,
  color,
  box,
  lines,
  edit,
}: {
  member: HouseMember
  rank: number
  arms: ReturnType<typeof playerArms>
  color: string
  box: TreeBox
  lines: MedallionLine[]
  edit?: TreeEdit
}) {
  const { t } = useLanguage()
  const selected = edit?.selected?.type === 'noble' && edit.selected.code === member.code
  const pending = edit?.pending?.from === member.code
  // The estimated height holds unless the text wraps further than expected: the
  // parchment then follows the text, so nothing spills outside the scroll.
  const content = useRef<HTMLDivElement>(null)
  const h = Math.max(box.h, Math.ceil(useElementSize(content).height))
  const choose = edit ? () => edit.onNoble(member.code) : undefined
  return (
    <li
      data-noble={member.code}
      data-dead={member.dead || undefined}
      className={cn(
        'pointer-events-none absolute text-center text-xs',
        edit && 'cursor-pointer',
      )}
      onClick={edit ? () => edit.onNoble(member.code) : undefined}
      style={{ left: box.x, top: box.y, width: box.w, minHeight: box.h }}
    >
      <svg
        aria-hidden="true"
        className="absolute inset-0 overflow-visible"
        width={box.w}
        height={h}
        viewBox={`0 0 ${box.w} ${h}`}
      >
        {selected && (
          <>
            <rect
              x="-5"
              y="-1"
              width={box.w + 10}
              height={h + 2}
              rx="12"
              fill="none"
              stroke={SELECTION_HALO}
              strokeWidth="3"
            />
            <rect
              x="-3"
              y="1"
              width={box.w + 6}
              height={h - 2}
              rx="10"
              fill="none"
              stroke={SELECTION_STROKE}
              strokeWidth="1.5"
            />
          </>
        )}
        {pending && (
          <rect
            x="-4"
            y="-1"
            width={box.w + 8}
            height={h + 2}
            rx="11"
            fill="none"
            stroke={LINEAGE_SECONDARY}
            strokeWidth="2"
            strokeDasharray="4 3"
          />
        )}
        <ScrollShape w={box.w} h={h} color={color} dead={member.dead} />
        {!member.dead && (
          <g transform={`translate(${box.w - CARTOUCHE_PAD_X - 4} 6)`}>
            <circle r="8.5" fill={color} stroke={LINEAGE_INK} strokeWidth="1.25" />
            <text
              textAnchor="middle"
              dy="3.5"
              className="font-display text-2xs font-bold"
              fill={LINEAGE_RANK_DIGIT}
              stroke={LINEAGE_INK}
              strokeWidth="2.2"
              strokeLinejoin="round"
              paintOrder="stroke"
            >
              {rank}
            </text>
          </g>
        )}
      </svg>
      <div
        ref={content}
        className={cn(
          'pointer-events-auto relative flex flex-col',
          edit && 'rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-brass-700/60',
        )}
        style={{ minHeight: box.h, padding: `12px ${CARTOUCHE_PAD_X}px` }}
        role={edit ? 'button' : undefined}
        tabIndex={edit ? 0 : undefined}
        aria-pressed={edit ? selected : undefined}
        onKeyDown={
          choose
            ? (event) => {
                if (event.target !== event.currentTarget) return
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  choose()
                }
              }
            : undefined
        }
      >
        {!member.dead && (
          <span className="sr-only">{t('lineage.rank', { rank })}</span>
        )}
        <div className="flex min-h-8 items-center gap-1.5">
          <span
            className={cn(
              'shrink-0 [&>svg]:h-8 [&>svg]:w-7',
              member.dead && 'opacity-60',
            )}
          >
            <ShieldIcon id={`lineage-arms-${member.code}`} arms={arms} />
          </span>
          <p
            className={cn(
              'min-w-0 flex-1 break-words text-sm font-semibold leading-4',
              member.dead ? 'text-ink-700 line-through' : 'text-ink-900',
            )}
          >
            {member.name}
            {member.sex && (
              <span aria-label={member.sex} className="ml-1 font-normal text-ink-700">
                {member.sex === 'female' ? '♀' : '♂'}
              </span>
            )}
          </p>
        </div>
        {lines.length > 0 && (
          <div className="mt-1">
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
          </div>
        )}
      </div>
    </li>
  )
})

/**
 * The alliance graph as a corner inset of the tree: folded by default so
 * the tree keeps the room.
 */
const AllianceInset = memo(function AllianceInset(props: {
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
})

const AllianceGraph = memo(function AllianceGraph({
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
  const houseById = useMemo(
    () => new Map(houses.map((house) => [house.player.id, house])),
    [houses],
  )
  const nameByCode = useMemo(() => {
    const byCode = new Map<string, string>()
    for (const house of houses)
      for (const member of house.members)
        if (!byCode.has(member.code)) byCode.set(member.code, member.name)
    return byCode
  }, [houses])
  const houseName = (id: PlayerId) => houseById.get(id)?.player.name || id
  const nobleName = (code: string) => nameByCode.get(code) ?? code
  const marriageName = (a: string, b: string) => `${nobleName(a)} × ${nobleName(b)}`

  const position = useMemo(() => {
    const cx = 200
    const cy = 160
    const ring = (list: House[], at: Map<PlayerId, { x: number; y: number }>) =>
      list.forEach((house, index) => {
        const angle = (2 * Math.PI * index) / list.length - Math.PI / 2
        at.set(house.player.id, {
          x: cx + 125 * Math.cos(angle),
          y: cy + 115 * Math.sin(angle),
        })
      })
    const at = new Map<PlayerId, { x: number; y: number }>()
    if (focus) {
      at.set(focus, { x: cx, y: cy })
      ring(
        houses.filter((house) => house.player.id !== focus),
        at,
      )
    } else ring(houses, at)
    return at
  }, [houses, focus])
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
          return (
            <g key={pair.key} opacity={dim(pair) ? 0.2 : 1}>
              <EdgeStroke
                kind={pair.kind}
                shape={{ x1: a.x, y1: a.y, x2: b.x, y2: b.y }}
              />
              {pair.kind === 'head' && (
                <IvyLeaves
                  points={[
                    { x: a.x, y: a.y },
                    { x: b.x, y: b.y },
                  ]}
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
})
