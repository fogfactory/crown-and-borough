import { OrderLauncher, type OrderFieldOption } from '@/components/OrderDialog'
import { formatCardLabel } from '@/lib/card-hand'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { appendDraftLine, draftLines } from '@/lib/winter-draft'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import type { CardKind, Noble, PlayerId, Region, Season, StateData } from '@/types'

const PLAYABLE_KINDS: CardKind[] = [
  'fair_weather',
  'abundant_harvest',
  'revolt',
  'seigneurial_tax',
  'trial',
]

function nobleOption(noble: Noble): OrderFieldOption {
  return { value: noble.code, label: `${noble.code} · ${noble.name} (${noble.owner})` }
}

function codeOption(code: string): OrderFieldOption {
  return { value: code, label: code }
}

function regionOptions(regions: Region[]): OrderFieldOption[] {
  return regions.map((region) => ({
    value: region.seed,
    label: region.name ? `${region.seed} · ${region.name}` : region.seed,
  }))
}

function nobleName(state: StateData, code: string): string {
  return state.nobles.find((noble) => noble.code === code)?.name ?? code
}

const FIEF_TITLE_BY_SIZE: Array<[number, MessageKey]> = [
  [6, 'fief.title.duchy'],
  [5, 'fief.title.marquisate'],
  [4, 'fief.title.county'],
  [3, 'fief.title.barony'],
]

function fiefTitleKey(size: number): MessageKey {
  return (FIEF_TITLE_BY_SIZE.find(([min]) => size >= min) ?? FIEF_TITLE_BY_SIZE[3])[1]
}

/** True when every territory of the group is reachable from the first one over the edges. */
function groupConnected(group: string[], edges: Array<[string, string]>): boolean {
  const inGroup = new Set(group)
  const seen = new Set([group[0]])
  const queue = [group[0]]
  while (queue.length > 0) {
    const current = queue.shift() as string
    for (const [a, b] of edges) {
      const next = a === current ? b : b === current ? a : null
      if (next && inGroup.has(next) && !seen.has(next)) {
        seen.add(next)
        queue.push(next)
      }
    }
  }
  return seen.size === inGroup.size
}

/**
 * Orders granted by the titles of the player (pope, bishop, astrologer,
 * fiefs), each configured in a dialog that appends a commented line. Only the
 * orders the server reports as possible (winterAids) are offered.
 */
export function TitleOrdersSection({
  state,
  player,
  winterDraft,
  onWinterChange,
}: {
  state: StateData
  player: PlayerId
  winterDraft: string
  onWinterChange: (text: string) => void
}) {
  const { t } = useLanguage()
  const aids = state.winterAids
  // Every order below is a winter order: nothing is offered in the other seasons.
  if (state.season !== 'winter' || !aids) return null
  const add = (line: string) => onWinterChange(appendDraftLine(winterDraft, line))
  const lines = draftLines(winterDraft)
  const byCode = new Map(state.nobles.map((noble) => [noble.code, noble]))
  const toOption = (code: string): OrderFieldOption => {
    const noble = byCode.get(code)
    return noble ? nobleOption(noble) : codeOption(code)
  }
  const own = state.nobles.filter((noble) => noble.owner === player)
  // A power button names the title or the dignity that grants the power: a
  // religious title leads ("Pape Rainaud"), a dignity carries the noble in
  // parentheses ("Astrologue (Dame Hugon)").
  const firstNameOf = (code: string) => byCode.get(code)?.firstName ?? nobleName(state, code)
  const titleHolder = (code: string, title: 'pope' | 'bishop') =>
    t(`orders.holder.${title}`, { name: firstNameOf(code) })
  const dignityHolder = (code: string, dignity: 'astrologer' | 'witch') => {
    const honorific = byCode.get(code)?.name.split(' ')[0] ?? ''
    return `${t(`dignity.${dignity}`)} (${[honorific, firstNameOf(code)].filter(Boolean).join(' ')})`
  }

  const liftedInDraft = new Set(
    lines.filter((line) => line.startsWith('X L ')).map((line) => line.slice(4)),
  )
  const excommunicable = aids.excommunicable.filter(
    (target) => !target.blocker || liftedInDraft.has(target.blocker),
  )
  const canExcommunicate = excommunicable.length > 0 && !lines.some((line) => line.startsWith('X E '))
  const liftable = aids.liftable.filter((code) => !liftedInDraft.has(code))

  const forecast = state.calamityForecast ?? []
  const astrologers = own.filter(
    (noble) =>
      (noble.dignities ?? []).includes('astrologer') &&
      noble.status !== 'dungeon' &&
      !lines.some((line) => line.startsWith(`V C ${noble.code} `)),
  )
  const canVeto = astrologers.length > 0 && forecast.length > 0

  const vacantFiefs = (state.fiefs ?? []).filter((fief) => fief.owner === player && !fief.holder)
  const buyable = aids.buyableCardinals.filter(
    (code) => !lines.includes(`N C ${code}`),
  )
  const sites = aids.fiefSites
  const castles = [...new Set(sites.flatMap((site) => site.castles))].sort()

  const launchers: React.ReactNode[] = []

  if (canExcommunicate) {
    launchers.push(
      <OrderLauncher
        key="x-e"
        label={`${titleHolder(state.pope ?? '', 'pope')} · ${t('orders.excommunicate')}`}
        title={t('orders.excommunicate')}
        fields={[
          {
            key: 'noble',
            label: t('orders.field.excommunicated'),
            options: excommunicable.map((target) => toOption(target.code)),
          },
        ]}
        buildOrder={(values) =>
          values.noble
            ? {
                line: `X E ${values.noble}`,
                comment: t('orders.comment.excommunicate', { name: nobleName(state, values.noble) }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  if (liftable.length > 0) {
    launchers.push(
      <OrderLauncher
        key="x-l"
        label={`${titleHolder(state.pope ?? '', 'pope')} · ${t('orders.liftExcommunication')}`}
        title={t('orders.liftExcommunication')}
        fields={[
          { key: 'noble', label: t('orders.field.excommunicated'), options: liftable.map(toOption) },
        ]}
        buildOrder={(values) =>
          values.noble
            ? {
                line: `X L ${values.noble}`,
                comment: t('orders.comment.lift', { name: nobleName(state, values.noble) }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  for (const code of buyable) {
    launchers.push(
      <OrderLauncher
        key={`n-c-${code}`}
        label={`${titleHolder(code, 'bishop')} · ${t('orders.buyCardinal')}`}
        title={t('orders.buyCardinal')}
        hint={() => t('orders.hint.cost', { cost: aids.cardinalCost })}
        fields={[]}
        buildOrder={() => ({
          line: `N C ${code}`,
          comment: t('orders.comment.buyCardinal', { name: nobleName(state, code) }),
        })}
        onConfirm={add}
      />,
    )
  }

  if (canVeto) {
    const positions: OrderFieldOption[] = forecast.map((kind, index) => ({
      value: String(index + 1),
      label: `${index + 1} · ${t(`card.${kind}` as MessageKey)}`,
    }))
    for (const astrologer of astrologers) {
      launchers.push(
        <OrderLauncher
          key={`v-c-${astrologer.code}`}
          label={`${dignityHolder(astrologer.code, 'astrologer')} · ${t('orders.calamityVeto')}`}
          title={t('orders.calamityVeto')}
          description={t('orders.calamityVetoHelp')}
          fields={[{ key: 'first', label: t('orders.field.firstCalamity'), options: positions }]}
          buildOrder={(values) =>
            values.first
              ? {
                  line: `V C ${astrologer.code} ${values.first}`,
                  comment: t('orders.comment.calamityVeto', { positions: values.first }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
  }

  if (sites.length > 0 && own.length > 0) {
    const siteOf = (capital: string) => sites.find((site) => site.castles.includes(capital))
    const group = (values: Record<string, string>): string[] => {
      const site = siteOf(values.capital)
      if (!site) return []
      const picked = new Set(values.territories.split(' '))
      const chosen = site.territories.filter((code) => picked.has(code))
      return [values.capital, ...chosen.filter((code) => code !== values.capital)]
    }
    launchers.push(
      <OrderLauncher
        key="t-f"
        label={t('orders.foundFief')}
        title={t('orders.foundFief')}
        description={t('orders.foundFiefHelp')}
        hint={(values) => {
          const codes = group(values)
          if (codes.length < 3) return t('orders.hint.fiefTooSmall')
          const site = siteOf(values.capital)
          if (!site || !groupConnected(codes, site.edges)) return t('orders.hint.fiefNotConnected')
          return t('orders.hint.fiefCost', {
            title: t(fiefTitleKey(codes.length)),
            count: codes.length,
            cost: codes.length * aids.fiefCostPerTerritory,
          })
        }}
        fields={[
          { key: 'noble', label: t('orders.field.holder'), options: own.map(nobleOption) },
          { key: 'capital', label: t('orders.field.capital'), options: castles.map(codeOption) },
          {
            key: 'territories',
            label: t('orders.field.fiefTerritories'),
            multi: true,
            options: (values) =>
              (siteOf(values.capital)?.territories ?? [])
                .filter((code) => code !== values.capital)
                .map(codeOption),
          },
        ]}
        buildOrder={(values) => {
          const codes = group(values)
          const site = siteOf(values.capital)
          if (!values.noble || !site || codes.length < 3 || !groupConnected(codes, site.edges)) {
            return null
          }
          return {
            line: `T F ${values.noble} ${codes.join(' ')}`,
            comment: t('orders.comment.foundFief', {
              capital: codes[0],
              name: nobleName(state, values.noble),
            }),
          }
        }}
        onConfirm={add}
      />,
    )
  }

  if (vacantFiefs.length > 0 && own.length > 0) {
    launchers.push(
      <OrderLauncher
        key="t-a"
        label={t('orders.assignFief')}
        title={t('orders.assignFief')}
        fields={[
          { key: 'noble', label: t('orders.field.holder'), options: own.map(nobleOption) },
          { key: 'fief', label: t('orders.field.fief'), options: vacantFiefs.map((fief) => codeOption(fief.capital)) },
        ]}
        buildOrder={(values) =>
          values.noble && values.fief
            ? {
                line: `T A ${values.noble} ${values.fief}`,
                comment: t('orders.comment.assignFief', {
                  capital: values.fief,
                  name: nobleName(state, values.noble),
                }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  const ritual = aids.rituals.find((option) => !lines.some((line) => line.startsWith(`S R ${option.noble} `)))
  if (ritual && !lines.some((line) => line.startsWith('S R '))) {
    const kinds: CardKind[] = ['plague', 'bad_weather', 'famine']
    const kindCodes: Record<string, string> = { plague: 'PE', bad_weather: 'MT', famine: 'FA' }
    const seasons: Array<[string, Season]> = [
      ['1', 'spring'],
      ['2', 'summer'],
      ['3', 'autumn'],
    ]
    launchers.push(
      <OrderLauncher
        key="s-r"
        label={`${dignityHolder(ritual.noble, 'witch')} · ${t('orders.ritual')}`}
        title={t('orders.ritual')}
        description={t('orders.ritualHelp')}
        hint={() => t('orders.hint.ritualRegion', { region: ritual.region })}
        fields={[
          {
            key: 'mode',
            label: t('orders.field.ritualMode'),
            options: [
              { value: 'calamity', label: t('orders.ritual.modeCalamity') },
              { value: 'season', label: t('orders.ritual.modeSeason') },
            ],
          },
          {
            key: 'choice',
            label: t('orders.field.ritualChoice'),
            options: (values) =>
              values.mode === 'season'
                ? seasons.map(([value, season]) => ({ value, label: t(SEASON_LABEL_KEYS[season]) }))
                : kinds.map((kind) => ({ value: kindCodes[kind], label: t(`card.${kind}` as MessageKey) })),
          },
        ]}
        buildOrder={(values) => {
          if (!values.choice) return null
          const season = seasons.find(([value]) => value === values.choice)
          return {
            line: `S R ${ritual.noble} ${values.choice}`,
            comment: season
              ? t('orders.comment.ritualSeason', { season: t(SEASON_LABEL_KEYS[season[1]]) })
              : t('orders.comment.ritualCalamity', {
                  card: t(`card.${kinds.find((kind) => kindCodes[kind] === values.choice) ?? 'plague'}` as MessageKey),
                }),
          }
        }}
        onConfirm={add}
      />,
    )
  }

  if (launchers.length === 0) return null
  return (
    <section className="space-y-2 rounded-lg border border-[#9bbbd3] bg-[#f7fbff] p-3">
      <h4 className="font-serif text-base font-semibold text-[#2c5b7d]">
        {t('orders.titlesTitle')}
      </h4>
      <p className="text-xs leading-relaxed text-[#55738a]">{t('orders.titlesDescription')}</p>
      <div className="flex flex-wrap gap-1.5">{launchers}</div>
    </section>
  )
}

/** The calamity each bonus card cancels in the region it targets. */
const CANCELED_CALAMITY: Partial<Record<CardKind, CardKind>> = {
  fair_weather: 'bad_weather',
  abundant_harvest: 'famine',
}

/** Calamities announced on a region that the card would cancel. */
function canceledIn(kind: CardKind, state: StateData, seed: string) {
  const canceled = CANCELED_CALAMITY[kind]
  return canceled
    ? (state.announcements ?? []).filter(
        (announcement) => announcement.kind === canceled && announcement.region === seed,
      )
    : []
}

/**
 * Territories a Révolte can target: the ones the server reports, plus every
 * territory of a fief or bishopric taxed by a tax line of the sheet being
 * written (`P TX XXX`).
 */
function revoltOptions(
  state: StateData,
  regions: Region[],
  draft: string,
): OrderFieldOption[] {
  const targets = new Set(state.revoltTargets ?? [])
  for (const line of draftLines(draft)) {
    const match = /^P (?:TX|ST) ([A-Z]{3})\b/.exec(line)
    if (!match) continue
    const fief = (state.fiefs ?? []).find((entry) => entry.capital === match[1])
    const region = regions.find((entry) => entry.seed === match[1])
    for (const code of fief?.territories ?? region?.territories ?? []) targets.add(code)
  }
  return [...targets].sort().map(codeOption)
}

function specialTargetField(
  kind: CardKind,
  state: StateData,
  regions: Region[],
  t: Translate,
  draft: string,
): { label: string; options: OrderFieldOption[] } {
  switch (kind) {
    case 'revolt':
      return {
        label: t('orders.field.territory'),
        options: revoltOptions(state, regions, draft),
      }
    case 'seigneurial_tax':
      return {
        label: t('orders.field.fief'),
        options: (state.fiefs ?? []).map((fief) => codeOption(fief.capital)),
      }
    case 'trial':
      return { label: t('orders.field.trialTarget'), options: state.nobles.map(nobleOption) }
    default: {
      // Regions where the card cancels an announced calamity come first.
      const options = regionOptions(regions).map((option) =>
        canceledIn(kind, state, option.value).length > 0
          ? { ...option, label: `${option.label} · ${t('orders.hint.cancelsShort')}` }
          : option,
      )
      return {
        label: t('orders.field.region'),
        options: [
          ...options.filter((option) => option.label.includes(t('orders.hint.cancelsShort'))),
          ...options.filter((option) => !option.label.includes(t('orders.hint.cancelsShort'))),
        ],
      }
    }
  }
}

function cancelHint(kind: CardKind, state: StateData, t: Translate, seed: string): string | null {
  if (!CANCELED_CALAMITY[kind]) return null
  const canceled = canceledIn(kind, state, seed)
  if (canceled.length === 0) return t('orders.hint.cancelsNone')
  return t('orders.hint.cancels', {
    list: canceled
      .map((announcement) =>
        t('orders.hint.cancelItem', {
          card: t(`card.${announcement.kind}` as MessageKey),
          season: t(SEASON_LABEL_KEYS[announcement.season]),
          year: announcement.year,
        }),
      )
      .join(', '),
  })
}

/**
 * The special cards of the hand: play one (`P KIND TER`) in the action
 * sheet, or discard it (`D C KIND`) in the winter one.
 */
export function SpecialCardOrders({
  state,
  regions,
  draft,
  onChange,
  discard = false,
}: {
  state: StateData
  regions: Region[]
  draft: string
  onChange: (text: string) => void
  /** Winter: the cards are discarded rather than played. */
  discard?: boolean
}) {
  const { t } = useLanguage()
  const hand = (state.specialHand ?? []).filter((kind) => PLAYABLE_KINDS.includes(kind))
  if (hand.length === 0) return null
  const lines = draftLines(draft)
  const used = (kind: CardKind, prefix: string) =>
    lines.filter((line) => line.startsWith(`${prefix} ${shortCode(kind)}`)).length
  const copies = (kind: CardKind) => hand.filter((card) => card === kind).length

  return (
    <div className="flex flex-wrap gap-1.5">
      {[...new Set(hand)].map((kind) => {
        const card = formatCardLabel(kind, t)
        if (discard) {
          return (
            <OrderLauncher
              key={kind}
              label={t('orders.discardSpecialCard', { card })}
              title={t('orders.discardSpecialCard', { card })}
              disabled={used(kind, 'D C') >= copies(kind)}
              fields={[]}
              buildOrder={() => ({
                line: `D C ${shortCode(kind)}`,
                comment: t('orders.comment.discardSpecial', { card }),
              })}
              onConfirm={(line) => onChange(appendDraftLine(draft, line))}
            />
          )
        }
        const target = specialTargetField(kind, state, regions, t, draft)
        if (target.options.length === 0) return null
        return (
          <OrderLauncher
            key={kind}
            label={t('orders.playSpecialCard', { card })}
            title={t('orders.playSpecialCard', { card })}
            disabled={used(kind, 'P') >= copies(kind)}
            hint={(values) => cancelHint(kind, state, t, values.target)}
            fields={[{ key: 'target', label: target.label, options: target.options }]}
            buildOrder={(values) =>
              values.target
                ? {
                    line: `P ${shortCode(kind)} ${values.target}`,
                    comment: t('orders.comment.playSpecial', { card, target: values.target }),
                  }
                : null
            }
            onConfirm={(line) => onChange(appendDraftLine(draft, line))}
          />
        )
      })}
    </div>
  )
}

const SHORT_CODES: Record<CardKind, string> = {
  fair_weather: 'BT',
  abundant_harvest: 'RA',
  revolt: 'RE',
  plague: 'PE',
  bad_weather: 'MT',
  famine: 'FA',
  seigneurial_tax: 'TX',
  trial: 'PR',
}

function shortCode(kind: CardKind): string {
  return SHORT_CODES[kind]
}

/** Region seed of a territory, from the bishoprics (a region is a bishopric). */
function regionSeedOf(state: StateData, territory: string): string | undefined {
  return state.bishoprics?.find((bishopric) => bishopric.territories.includes(territory))?.region
}

/**
 * Revolt appeasement of the player's clerics (pope, cardinal, bishop, abbess),
 * played in the action sheet without a card: the free rite (`P AG HHH TER`) in
 * the region where the cleric stands, and the safe paid appeasement
 * (`P AP HHH TER`) of a bishop in their bishopric, a cardinal or the pope.
 */
export function AppeasementOrders({
  state,
  player,
  draft,
  onChange,
}: {
  state: StateData
  player: PlayerId
  draft: string
  onChange: (text: string) => void
}) {
  const { t } = useLanguage()
  const costBase = state.appeasementCostBase ?? 2
  const rebels = state.territories.filter((entry) => entry.army?.owner === 'NEUTRAL')
  if (state.season === 'winter' || rebels.length === 0) return null
  const lines = draftLines(draft)
  const add = (line: string) => onChange(appendDraftLine(draft, line))
  const clerics = state.nobles.filter(
    (noble) =>
      noble.owner === player &&
      noble.status !== 'dungeon' &&
      (noble.religiousTitle || (noble.dignities ?? []).includes('abbess')) &&
      !lines.some(
        (line) => /^P (?:AG|FQ|AP|PQ) /.test(line) && line.split(/\s+/)[2] === noble.code,
      ),
  )
  const rebelOption = (entry: StateData['territories'][number]): OrderFieldOption => ({
    value: entry.id,
    label: t('orders.appease.rebels', { territory: entry.id, count: entry.army?.size ?? 0 }),
  })
  const holder = (noble: Noble) =>
    noble.religiousTitle
      ? t(`orders.holder.${noble.religiousTitle}` as MessageKey, {
          name: noble.firstName ?? noble.name,
        })
      : `${t('dignity.abbess')} (${noble.firstName ?? noble.name})`

  const launchers: React.ReactNode[] = []
  for (const noble of clerics) {
    const here = regionSeedOf(state, noble.location)
    const ritual = rebels.filter((entry) => here && regionSeedOf(state, entry.id) === here)
    if (ritual.length > 0) {
      launchers.push(
        <OrderLauncher
          key={`ag-${noble.code}`}
          label={`${holder(noble)} · ${t('orders.appease.rite')}`}
          title={t('orders.appease.rite')}
          description={t('orders.appease.riteHelp')}
          fields={[
            { key: 'territory', label: t('orders.field.territory'), options: ritual.map(rebelOption) },
          ]}
          buildOrder={(values) =>
            values.territory
              ? {
                  line: `P AG ${noble.code} ${values.territory}`,
                  comment: t('orders.comment.appeaseRite', {
                    name: noble.name,
                    territory: values.territory,
                  }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
    const title = noble.religiousTitle
    const ownRegions = new Set(
      (state.bishoprics ?? [])
        .filter((bishopric) => bishopric.bishop === noble.code)
        .map((bishopric) => bishopric.region),
    )
    const paid = rebels.filter((entry) =>
      title === 'cardinal' || title === 'pope'
        ? true
        : title === 'bishop' && ownRegions.has(regionSeedOf(state, entry.id) ?? ''),
    )
    if (paid.length > 0) {
      launchers.push(
        <OrderLauncher
          key={`ap-${noble.code}`}
          label={`${holder(noble)} · ${t('orders.appease.paid')}`}
          title={t('orders.appease.paid')}
          description={t('orders.appease.paidHelp')}
          hint={(values) => {
            const size = state.territories.find((entry) => entry.id === values.territory)?.army?.size ?? 0
            return size > 0 ? t('orders.hint.cost', { cost: costBase ** size }) : null
          }}
          fields={[
            { key: 'territory', label: t('orders.field.territory'), options: paid.map(rebelOption) },
          ]}
          buildOrder={(values) =>
            values.territory
              ? {
                  line: `P AP ${noble.code} ${values.territory}`,
                  comment: t('orders.comment.appeasePaid', {
                    name: noble.name,
                    territory: values.territory,
                  }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
  }
  if (launchers.length === 0) return null
  return <div className="flex flex-wrap gap-1.5">{launchers}</div>
}
