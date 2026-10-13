import { OrderLauncher, type OrderFieldOption } from '@/components/OrderDialog'
import { formatCardLabel } from '@/lib/card-hand'
import { parseDraftTaxLine } from '@/lib/projected-tax'
import { royalIssuerOf } from '@/lib/royal'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { appendDraftLine, draftLines } from '@/lib/winter-draft'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import { NEUTRAL_PLAYER_ID, type CardKind, type Dignity, type FiefEdge, type Noble, type PlayerId, type Region, type Season, type StateData, type Territory } from '@/types'

const PLAYABLE_KINDS: CardKind[] = [
  'fair_weather',
  'abundant_harvest',
  'revolt',
  'seigneurial_tax',
  'tunnel',
  'ambush',
  'assassination',
  'justice',
]

function nobleOption(noble: Noble): OrderFieldOption {
  return { value: noble.code, label: `${noble.name} (${noble.code}) · ${noble.owner}` }
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
  return (
    state.nobles.find((noble) => noble.code === code)?.name ??
    state.deceased?.find((noble) => noble.code === code)?.name ??
    code
  )
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
function groupConnected(group: string[], edges: FiefEdge[]): boolean {
  const inGroup = new Set(group)
  const seen = new Set([group[0]])
  const queue = [group[0]]
  while (queue.length > 0) {
    const current = queue.shift() as string
    for (const { from, to } of edges) {
      const next = from === current ? to : to === current ? from : null
      if (next && inGroup.has(next) && !seen.has(next)) {
        seen.add(next)
        queue.push(next)
      }
    }
  }
  return seen.size === inGroup.size
}

const MARRIAGE_BLOCKING_DIGNITIES: Dignity[] = ['d_arc', 'abbess', 'herbalist', 'chevalier_d_eon']

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
  const titleHolder = (code: string, title: 'pope' | 'bishop' | 'cardinal') =>
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

  const canMarry = (noble: Noble) =>
    noble.status === 'free' &&
    !noble.spouse &&
    !(noble.dignities ?? []).some((dignity) => MARRIAGE_BLOCKING_DIGNITIES.includes(dignity))
  const fiances = own.filter(
    (noble) => canMarry(noble) && !lines.some((line) => line.startsWith(`M N ${noble.code} `)),
  )
  if (fiances.length > 0) {
    launchers.push(
      <OrderLauncher
        key="m-n"
        label={t('orders.marriage')}
        title={t('orders.marriage')}
        description={t('orders.marriageHelp')}
        fields={[
          { key: 'noble', label: t('orders.field.fiance'), options: fiances.map(nobleOption) },
          {
            key: 'spouse',
            label: t('orders.field.spouse'),
            options: (values) =>
              state.nobles
                .filter((other) => {
                  const mine = byCode.get(values.noble ?? '')
                  return (
                    mine !== undefined &&
                    canMarry(other) &&
                    other.owner !== player &&
                    other.sex !== mine.sex
                  )
                })
                .map(nobleOption),
          },
        ]}
        buildOrder={(values) =>
          values.noble && values.spouse
            ? {
                line: `M N ${values.noble} ${values.spouse}`,
                comment: t('orders.comment.marriage', {
                  name: nobleName(state, values.noble),
                  spouse: nobleName(state, values.spouse),
                }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  for (const marriage of aids.dissolvable ?? []) {
    if (lines.some((line) => line === `X D ${marriage.noble}` || line === `X D ${marriage.spouse}`)) continue
    const label =
      byCode.get(marriage.noble)?.owner === player && state.pope !== marriage.noble
        ? t('orders.dissolveRequest')
        : t('orders.dissolve')
    launchers.push(
      <OrderLauncher
        key={`x-d-${marriage.noble}`}
        label={`${nobleName(state, marriage.noble)} & ${nobleName(state, marriage.spouse)} · ${label}`}
        title={label}
        description={t('orders.dissolveHelp')}
        fields={[]}
        buildOrder={() => ({
          line: `X D ${marriage.noble}`,
          comment: t('orders.comment.dissolve', {
            name: nobleName(state, marriage.noble),
            spouse: nobleName(state, marriage.spouse),
          }),
        })}
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

  for (const code of aids.inquirers) {
    if (lines.some((line) => line.startsWith(`Q ${code} `))) continue
    const title = state.pope === code ? 'pope' : 'cardinal'
    launchers.push(
      <OrderLauncher
        key={`q-${code}`}
        label={`${titleHolder(code, title)} · ${t('orders.inquiry')}`}
        title={t('orders.inquiry')}
        description={t('orders.inquiryHelp')}
        hint={(values) =>
          values.noble ? t('orders.hint.cost', { cost: aids.inquiryCosts[values.noble] ?? 0 }) : ''
        }
        fields={[
          {
            key: 'noble',
            label: t('orders.field.investigated'),
            options: state.nobles
              .filter((noble) => noble.code !== code)
              .map((noble) => ({
                ...nobleOption(noble),
                label: `${nobleOption(noble).label} — ${aids.inquiryCosts[noble.code] ?? 0} R`,
              }))
              // The lineage of the dead can be inquired into, at the cost of
              // the title they bore.
              .concat(
                (state.deceased ?? []).map((noble) => ({
                  value: noble.code,
                  label: `✝ ${noble.name} (${noble.code}) · ${noble.owner} — ${aids.inquiryCosts[noble.code] ?? 0} R`,
                })),
              ),
          },
        ]}
        buildOrder={(values) =>
          values.noble
            ? {
                line: `Q ${code} ${values.noble}`,
                comment: t('orders.comment.inquiry', {
                  name: nobleName(state, values.noble),
                }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  // A J line names the backing cardinal; a second cardinal, of the same player
  // or of another, must file the same trial for it to take place.
  const judges = aids.trialJudges.filter((code) => !lines.some((line) => line.startsWith(`J ${code} `)))
  if (judges.length > 0) {
    launchers.push(
      <OrderLauncher
        key="trial"
        label={t('orders.trial')}
        title={t('orders.trial')}
        description={t('orders.trialHelp')}
        hint={(values) =>
          values.noble ? t('orders.hint.trialSecond', { name: nobleName(state, values.noble) }) : ''
        }
        fields={[
          {
            key: 'cardinal',
            label: t('orders.field.trialCardinal'),
            options: judges.map((code) => ({
              value: code,
              label: titleHolder(code, state.pope === code ? 'pope' : 'cardinal'),
            })),
          },
          {
            key: 'noble',
            label: t('orders.field.tried'),
            options: state.nobles
              .filter((noble) => !judges.includes(noble.code))
              .map((noble) => nobleOption(noble)),
          },
        ]}
        buildOrder={(values) =>
          values.cardinal && values.noble
            ? {
                line: `J ${values.cardinal} ${values.noble}`,
                comment: t('orders.comment.trial', { name: nobleName(state, values.noble) }),
              }
            : null
        }
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

  // T X: annex a free territory the player holds next to one of their fiefs.
  const annexFiefs = (aids.ownFiefs ?? []).filter((fief) => fief.annexable.length > 0)
  if (annexFiefs.length > 0) {
    const annexCost = aids.annexCostPerTerritory ?? 0
    const annexedInDraft = new Set(
      lines.filter((line) => line.startsWith('T X ')).map((line) => line.split(/\s+/)[4]),
    )
    const annexable = (capital: string | undefined) =>
      (annexFiefs.find((fief) => fief.capital === capital)?.annexable ?? []).filter(
        (code) => !annexedInDraft.has(code),
      )
    launchers.push(
      <OrderLauncher
        key="t-x"
        label={t('orders.annexTerritory')}
        title={t('orders.annexTerritory')}
        description={t('orders.annexTerritoryHelp')}
        hint={(values) =>
          values.territory
            ? t('orders.hint.annexCost', { territory: values.territory, cost: annexCost })
            : ''
        }
        fields={[
          {
            key: 'fief',
            label: t('orders.field.fief'),
            options: annexFiefs
              .filter((fief) => annexable(fief.capital).length > 0)
              .map((fief) => codeOption(fief.capital)),
          },
          {
            key: 'territory',
            label: t('orders.field.annexedTerritory'),
            options: (values) => annexable(values.fief).map(codeOption),
          },
        ]}
        buildOrder={(values) => {
          const fief = annexFiefs.find((candidate) => candidate.capital === values.fief)
          if (!fief || !values.territory) return null
          return {
            line: `T X ${fief.holder} ${fief.capital} ${values.territory}`,
            comment: t('orders.comment.annexTerritory', {
              territory: values.territory,
              capital: fief.capital,
            }),
          }
        }}
        onConfirm={add}
      />,
    )
  }

  // T R: raise a fief by one rank, once per winter and per fief.
  const rankFiefs = (aids.ownFiefs ?? []).filter(
    (fief) => fief.nextTitle && !lines.some((line) => line.startsWith(`T R ${fief.holder} ${fief.capital}`)),
  )
  if (rankFiefs.length > 0) {
    launchers.push(
      <OrderLauncher
        key="t-r"
        label={t('orders.changeRank')}
        title={t('orders.changeRank')}
        description={t('orders.changeRankHelp')}
        hint={(values) => {
          const fief = rankFiefs.find((candidate) => candidate.capital === values.fief)
          if (!fief?.nextTitle) return ''
          return t('orders.hint.rankCost', {
            from: t(`fief.title.${fief.title}` as MessageKey),
            to: t(`fief.title.${fief.nextTitle}` as MessageKey),
            count: fief.territories.length,
            cost: fief.rankCost,
          })
        }}
        fields={[
          {
            key: 'fief',
            label: t('orders.field.fief'),
            options: rankFiefs.map((fief) => ({
              value: fief.capital,
              label: `${fief.capital} · ${t(`fief.title.${fief.title}` as MessageKey)}`,
            })),
          },
        ]}
        buildOrder={(values) => {
          const fief = rankFiefs.find((candidate) => candidate.capital === values.fief)
          if (!fief?.nextTitle) return null
          return {
            line: `T R ${fief.holder} ${fief.capital}`,
            comment: t('orders.comment.changeRank', {
              capital: fief.capital,
              title: t(`fief.title.${fief.nextTitle}` as MessageKey),
            }),
          }
        }}
        onConfirm={add}
      />,
    )
  }

  // W X, W R, W F, W G: royal orders of the king, or of the regent in his stead.
  const royal = aids.royal
  if (royal) {
    const allFiefs = state.fiefs ?? []
    const fiefOptions = (fiefs: typeof allFiefs): OrderFieldOption[] =>
      fiefs.map((fief) => ({
        value: fief.capital,
        label: `${fief.capital} · ${t(`fief.title.${fief.title}` as MessageKey)} · ${fief.owner}`,
      }))
    const usedInDraft = (prefix: string) =>
      lines.filter((line) => line.startsWith(prefix)).length >= royal.ordersPerWinter
    // Territories of a fief other than the annexing one: the ones an accord opens.
    const annexableFrom = (capital: string | undefined) =>
      allFiefs
        .filter((fief) => fief.capital !== capital)
        .flatMap((fief) => fief.territories.filter((code) => code !== fief.capital))
    if (allFiefs.length > 1 && !usedInDraft('W X ')) {
      launchers.push(
        <OrderLauncher
          key="w-x"
          label={t('orders.royalAnnex')}
          title={t('orders.royalAnnex')}
          description={t('orders.royalAnnexHelp')}
          fields={[
            { key: 'fief', label: t('orders.field.fief'), options: fiefOptions(allFiefs) },
            {
              key: 'territory',
              label: t('orders.field.annexedTerritory'),
              options: (values) => annexableFrom(values.fief).map(codeOption),
            },
          ]}
          buildOrder={(values) =>
            values.fief && values.territory
              ? {
                  line: `W X ${royal.issuer} ${values.fief} ${values.territory}`,
                  comment: t('orders.comment.royalAnnex', {
                    capital: values.fief,
                    territory: values.territory,
                  }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
    if (allFiefs.length > 0 && !usedInDraft('W R ')) {
      launchers.push(
        <OrderLauncher
          key="w-r"
          label={t('orders.royalRank')}
          title={t('orders.royalRank')}
          description={t('orders.royalRankHelp')}
          fields={[{ key: 'fief', label: t('orders.field.fief'), options: fiefOptions(allFiefs) }]}
          buildOrder={(values) =>
            values.fief
              ? {
                  line: `W R ${royal.issuer} ${values.fief}`,
                  comment: t('orders.comment.royalRank', { capital: values.fief }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
    if (!usedInDraft('W F ')) {
      launchers.push(
        <OrderLauncher
          key="w-f"
          label={t('orders.royalFound')}
          title={t('orders.royalFound')}
          description={t('orders.royalFoundHelp')}
          fields={[
            {
              key: 'capital',
              label: t('orders.field.capital'),
              options: state.territories.map((territory) => codeOption(territory.id)),
            },
          ]}
          buildOrder={(values) =>
            values.capital
              ? {
                  line: `W F ${royal.issuer} ${values.capital}`,
                  comment: t('orders.comment.royalFound', { capital: values.capital }),
                }
              : null
          }
          onConfirm={add}
        />,
      )
    }
    if (vacantFiefs.length > 0 && !usedInDraft('W G ')) {
      launchers.push(
        <OrderLauncher
          key="w-g"
          label={t('orders.royalGrant')}
          title={t('orders.royalGrant')}
          description={t('orders.royalGrantHelp')}
          fields={[
            { key: 'fief', label: t('orders.field.fief'), options: fiefOptions(vacantFiefs) },
            {
              key: 'noble',
              label: t('orders.field.grantee'),
              options: state.nobles.map(nobleOption),
            },
          ]}
          buildOrder={(values) =>
            values.fief && values.noble
              ? {
                  line: `W G ${royal.issuer} ${values.fief} ${values.noble}`,
                  comment: t('orders.comment.royalGrant', {
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
    <section className="space-y-2 rounded-lg border border-winter-700/30 bg-winter-100 p-3">
      <h4 className="font-display text-base font-semibold text-winter-700">
        {t('orders.titlesTitle')}
      </h4>
      <p className="text-xs leading-relaxed text-winter-700">{t('orders.titlesDescription')}</p>
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
  for (const raw of draftLines(draft)) {
    const line = parseDraftTaxLine(raw)
    if (!line) continue
    const fief = line.tithe
      ? undefined
      : (state.fiefs ?? []).find((entry) => entry.capital === line.target)
    if (fief) {
      for (const code of fief.territories) targets.add(code)
      continue
    }
    // A tithe opens the bishopric it names, or every bishopric for the pope's
    // untargeted one.
    const papal =
      !line.target &&
      state.nobles.some((noble) => noble.code === line.issuer && noble.religiousTitle === 'pope')
    for (const region of regions) {
      if (papal || region.seed === line.target) {
        for (const code of region.territories) targets.add(code)
      }
    }
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
  player,
  mapTerritories = [],
  regions,
  draft,
  onChange,
  discard = false,
}: {
  state: StateData
  /** Map territories, for the commune names shown in the Tax card targets. */
  mapTerritories?: Territory[]
  /** Lets the Tax card offer the player's clerics as tithe issuers. */
  player?: PlayerId
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
        if (kind === 'seigneurial_tax') {
          const launcher = taxLauncher({ state, player, mapTerritories, t, lines, add: (line) => onChange(appendDraftLine(draft, line)) })
          return launcher ? (
            <span key={kind}>{launcher}</span>
          ) : null
        }
        if (kind === 'tunnel') {
          return (
            <span key={kind} className="contents">
              {tunnelLaunchers({ state, player, mapTerritories, t, lines, copies: copies(kind), add: (line) => onChange(appendDraftLine(draft, line)) })}
            </span>
          )
        }
        if (kind === 'ambush') {
          return (
            <span key={kind} className="contents">
              {ambushLauncher({ state, mapTerritories, t, lines, copies: copies(kind), add: (line) => onChange(appendDraftLine(draft, line)) })}
            </span>
          )
        }
        if (kind === 'assassination') {
          const launcher = assassinationLauncher({ state, player, t, lines, copies: copies(kind), add: (line) => onChange(appendDraftLine(draft, line)) })
          return launcher ? <span key={kind}>{launcher}</span> : null
        }
        if (kind === 'justice') {
          const launcher = justiceLauncher({ state, t, lines, copies: copies(kind), add: (line) => onChange(appendDraftLine(draft, line)) })
          return launcher ? <span key={kind}>{launcher}</span> : null
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

/** Issuer values: `tax:HHH` (lord), `tax:` (vacant fief, no noble), `tithe:HHH` (cleric). */
const VACANT_FIEF = 'tax:'

/**
 * The Tax card, always played through a noble of the player: the holder of a
 * fief taxes it (`P TX HHH XXX`, seigneurial tax), a cleric tithes a bishopric
 * (`P DI HHH XXX`: a bishop their own, a cardinal or the pope any). One noble
 * can be both lord and bishop, so the issuer entry also fixes the kind of tax.
 */
function taxLauncher({
  state,
  player,
  mapTerritories,
  t,
  lines,
  add,
}: {
  state: StateData
  player?: PlayerId
  mapTerritories: Territory[]
  t: Translate
  lines: string[]
  add: (line: string) => void
}) {
  const copies = (state.specialHand ?? []).filter((card) => card === 'seigneurial_tax').length
  const used = lines.filter((line) => /^P (?:TX|ST|DI|TT) /.test(line)).length
  const card = formatCardLabel('seigneurial_tax', t)
  const communeName = (code: string, fallback?: string) =>
    mapTerritories.find((territory) => territory.id === code)?.name ?? fallback ?? code
  const ofName = (name: string) =>
    t(/^[AEIOUYÀÂÉÈÊÎÔÛ]/i.test(name) ? 'orders.tax.ofVowel' : 'orders.tax.of', { name })
  const ownFiefs = (state.fiefs ?? []).filter((fief) => fief.owner === player)
  const fiefOption = (fief: (typeof ownFiefs)[number]): OrderFieldOption => ({
    value: fief.capital,
    label: t('orders.tax.fief', {
      code: fief.capital,
      title: t(`fief.title.${fief.title}` as MessageKey),
      name: ofName(communeName(fief.capital)),
    }),
  })
  const nobleLabel = (noble: Noble) => `${noble.firstName ?? noble.name} (${noble.code})`
  const bishoprics = state.bishoprics ?? []
  const clerics = state.nobles.filter(
    (noble) =>
      noble.owner === player &&
      noble.status !== 'dungeon' &&
      (noble.religiousTitle === 'bishop' ||
        noble.religiousTitle === 'cardinal' ||
        noble.religiousTitle === 'pope'),
  )
  const bishopricOptions = (cleric: Noble | undefined): OrderFieldOption[] =>
    bishoprics
      .filter(
        (bishopric) =>
          cleric?.religiousTitle === 'cardinal' ||
          cleric?.religiousTitle === 'pope' ||
          bishopric.bishop === cleric?.code,
      )
      .map((bishopric) => ({
        value: bishopric.region,
        label: t('orders.tax.bishopric', {
          code: bishopric.region,
          name: ofName(communeName(bishopric.region, bishopric.name)),
        }),
      }))
  const issuers: OrderFieldOption[] = []
  const lordCodes = [...new Set(ownFiefs.map((fief) => fief.holder).filter(Boolean))] as string[]
  for (const code of lordCodes) {
    const lord = state.nobles.find((noble) => noble.code === code)
    if (!lord) continue
    issuers.push({
      value: `tax:${code}`,
      label: t('orders.issuer.lord', { noble: nobleLabel(lord) }),
    })
  }
  if (ownFiefs.some((fief) => !fief.holder)) {
    issuers.push({ value: VACANT_FIEF, label: t('orders.issuer.vacantFief') })
  }
  const hasLord = issuers.length > 0
  // The king, or the regent in his place, may tax any fief he does not hold
  // himself (politique.md "Pouvoirs royaux").
  const royalIssuer = royalIssuerOf(state)
  const royalFiefs = (state.fiefs ?? []).filter(
    (fief) => !(fief.owner === player && (!fief.holder || fief.holder === royalIssuer?.code)),
  )
  if (royalIssuer && royalIssuer.owner === player && royalFiefs.length > 0) {
    issuers.push({
      value: `royal:${royalIssuer.code}`,
      label: t('orders.issuer.royal', { noble: nobleLabel(royalIssuer) }),
    })
  }
  for (const cleric of clerics) {
    if (bishopricOptions(cleric).length === 0) continue
    issuers.push({
      value: `tithe:${cleric.code}`,
      label: t(
        cleric.religiousTitle === 'bishop'
          ? 'orders.issuer.ownBishopric'
          : cleric.religiousTitle === 'pope'
            ? 'orders.issuer.allBishoprics'
            : 'orders.issuer.anyBishopric',
        { noble: `${nobleLabel(cleric)} (${t(`orders.holder.${cleric.religiousTitle}` as MessageKey, { name: '' }).trim()})` },
      ),
    })
  }
  const hasTithe = issuers.some((issuer) => issuer.value.startsWith('tithe:'))
  const hasRoyal = issuers.some((issuer) => issuer.value.startsWith('royal:'))
  const help: MessageKey = hasRoyal
    ? 'orders.taxHelp.royal'
    : hasLord
    ? hasTithe
      ? 'orders.taxHelp.both'
      : 'orders.taxHelp.lord'
    : 'orders.taxHelp.tithe'
  if (issuers.length === 0) return null
  const targetOptions = (values: Record<string, string>) => {
    const [kind, code] = (values.issuer ?? '').split(':')
    if (kind === 'tithe') return bishopricOptions(clerics.find((cleric) => cleric.code === code))
    if (kind === 'royal') return royalFiefs.map(fiefOption)
    return ownFiefs.filter((fief) => (fief.holder ?? '') === code).map(fiefOption)
  }
  // The pope's tithe names no bishopric: it covers all of them.
  const isPope = (values: Record<string, string>) => {
    const [kind, code] = (values.issuer ?? '').split(':')
    return kind === 'tithe' && clerics.find((cleric) => cleric.code === code)?.religiousTitle === 'pope'
  }
  const lineFor = (values: Record<string, string>) => {
    const [kind, code] = values.issuer.split(':')
    if (kind === 'tithe' && isPope(values)) return `P DI ${code}`
    return kind === 'tithe'
      ? `P DI ${code} ${values.target}`
      : code
        ? `P TX ${code} ${values.target}`
        : `P TX ${values.target}`
  }
  return (
    <OrderLauncher
      label={t('orders.playSpecialCard', { card })}
      title={t('orders.playSpecialCard', { card })}
      description={t(help)}
      disabled={used >= copies}
      fields={[
        { key: 'issuer', label: t('orders.field.issuer'), options: issuers },
        {
          key: 'target',
          label: t('orders.field.taxTarget'),
          options: targetOptions,
          hidden: isPope,
        },
      ]}
      buildOrder={(values) =>
        values.target || isPope(values)
          ? {
              line: lineFor(values),
              comment: t('orders.comment.playSpecial', {
                card,
                target: isPope(values) ? t('orders.tax.allBishoprics') : values.target,
              }),
            }
          : null
      }
      onConfirm={add}
    />
  )
}

/**
 * The origin and target options of a card played on an attack: an origin holds
 * an army (the player's own, which must then plan that attack, or another
 * player's, whose attack is only advised), a target is a territory next to the
 * chosen origin.
 */
function attackFieldOptions(state: StateData, mapTerritories: Territory[]) {
  const origins = state.territories
    .filter((territory) => territory.army && territory.army.owner !== NEUTRAL_PLAYER_ID)
    .map((territory) => codeOption(territory.id))
  const allTargets = state.territories.map((territory) => codeOption(territory.id)).sort((a, b) => a.value.localeCompare(b.value))
  const targets = (values: Record<string, string>) => {
    const origin = mapTerritories.find((territory) => territory.id === values.origin)
    return origin ? allTargets.filter((option) => origin.adjacencies.includes(option.value)) : allTargets
  }
  return { origins, targets }
}

/**
 * The Tunnel card has two forms: `P SO NNN` frees one of the player's detained
 * nobles, `P SO AAA - BBB` designates the player's attack from AAA to BBB,
 * which ignores the fortifications of BBB.
 */
function tunnelLaunchers({
  state,
  player,
  mapTerritories,
  t,
  lines,
  copies,
  add,
}: {
  state: StateData
  player?: PlayerId
  mapTerritories: Territory[]
  t: Translate
  lines: string[]
  copies: number
  add: (line: string) => void
}) {
  const card = formatCardLabel('tunnel', t)
  const prefix = `P ${shortCode('tunnel')}`
  const spent = lines.filter((line) => line.startsWith(prefix)).length >= copies
  const detained = state.nobles.filter((noble) => noble.owner === player && noble.status !== 'free')
  const { origins, targets } = attackFieldOptions(state, mapTerritories)
  return (
    <>
      {detained.length > 0 && (
        <OrderLauncher
          label={t('orders.playTunnelEscape', { card })}
          title={t('orders.playTunnelEscape', { card })}
          disabled={spent}
          fields={[{ key: 'noble', label: t('orders.field.tunnelNoble'), options: detained.map(nobleOption) }]}
          buildOrder={(values) =>
            values.noble
              ? {
                  line: `${prefix} ${values.noble}`,
                  comment: t('orders.comment.tunnelEscape', { card, target: values.noble }),
                }
              : null
          }
          onConfirm={add}
        />
      )}
      {origins.length > 0 && (
        <OrderLauncher
          label={t('orders.playTunnelAttack', { card })}
          title={t('orders.playTunnelAttack', { card })}
          disabled={spent}
          fields={[
            { key: 'origin', label: t('orders.field.tunnelOrigin'), options: origins },
            { key: 'target', label: t('orders.field.tunnelDestination'), options: targets },
          ]}
          buildOrder={(values) =>
            values.origin && values.target
              ? {
                  line: `${prefix} ${values.origin} - ${values.target}`,
                  comment: t('orders.comment.tunnelAttack', { card, origin: values.origin, target: values.target }),
                }
              : null
          }
          onConfirm={add}
        />
      )}
    </>
  )
}

/**
 * The Assassination card, `P AS HHH NNN`: the free noble HHH of the player
 * commissions the death of the noble NNN of another player.
 */
export function assassinationLauncher({
  state,
  player,
  t,
  lines,
  copies,
  add,
  label,
}: {
  state: StateData
  player?: PlayerId
  t: Translate
  lines: string[]
  copies: number
  add: (line: string) => void
  /** Text of the button; defaults to the "play" label. */
  label?: string
}) {
  const card = formatCardLabel('assassination', t)
  const prefix = `P ${shortCode('assassination')}`
  const commissioners = state.nobles.filter((noble) => noble.owner === player && noble.status === 'free')
  const victims = state.nobles.filter((noble) => noble.owner !== player)
  if (commissioners.length === 0 || victims.length === 0) return null
  return (
    <OrderLauncher
      label={label ?? t('orders.playAssassination', { card })}
      title={t('orders.playAssassination', { card })}
      disabled={lines.filter((line) => line.startsWith(prefix)).length >= copies}
      fields={[
        { key: 'commissioner', label: t('orders.field.commissioner'), options: commissioners.map(nobleOption) },
        { key: 'victim', label: t('orders.field.assassinationTarget'), options: victims.map(nobleOption) },
      ]}
      buildOrder={(values) =>
        values.commissioner && values.victim
          ? {
              line: `${prefix} ${values.commissioner} ${values.victim}`,
              comment: t('orders.comment.assassination', { card, commissioner: values.commissioner, target: values.victim }),
            }
          : null
      }
      onConfirm={add}
    />
  )
}

/**
 * The Justice card, `P JU HHH`: any noble, of any player, is tried; they are
 * executed if they commissioned an assassination.
 */
export function justiceLauncher({
  state,
  t,
  lines,
  copies,
  add,
  label,
}: {
  state: StateData
  t: Translate
  lines: string[]
  copies: number
  add: (line: string) => void
  /** Text of the button; defaults to the "play" label. */
  label?: string
}) {
  const card = formatCardLabel('justice', t)
  const prefix = `P ${shortCode('justice')}`
  // The King is never tried: the order would be rejected.
  const targets = state.nobles.filter((noble) => noble.code !== state.king)
  if (targets.length === 0) return null
  return (
    <OrderLauncher
      label={label ?? t('orders.playJustice', { card })}
      title={t('orders.playJustice', { card })}
      disabled={lines.filter((line) => line.startsWith(prefix)).length >= copies}
      fields={[{ key: 'target', label: t('orders.field.justiceTarget'), options: targets.map(nobleOption) }]}
      buildOrder={(values) =>
        values.target
          ? { line: `${prefix} ${values.target}`, comment: t('orders.comment.justice', { card, target: values.target }) }
          : null
      }
      onConfirm={add}
    />
  )
}

/**
 * The Ambush card, `P EM AAA BBB`, designates the attack from AAA to BBB (the
 * player's own, or another player's as advice): if it wins and the defending
 * army retreats with a noble, the first noble of its owner's line of
 * succession is captured.
 */
function ambushLauncher({
  state,
  mapTerritories,
  t,
  lines,
  copies,
  add,
}: {
  state: StateData
  mapTerritories: Territory[]
  t: Translate
  lines: string[]
  copies: number
  add: (line: string) => void
}) {
  const card = formatCardLabel('ambush', t)
  const prefix = `P ${shortCode('ambush')}`
  const { origins, targets } = attackFieldOptions(state, mapTerritories)
  if (origins.length === 0) return null
  return (
    <OrderLauncher
      label={t('orders.playAmbush', { card })}
      title={t('orders.playAmbush', { card })}
      disabled={lines.filter((line) => line.startsWith(prefix)).length >= copies}
      fields={[
        { key: 'origin', label: t('orders.field.tunnelOrigin'), options: origins },
        { key: 'target', label: t('orders.field.tunnelDestination'), options: targets },
      ]}
      buildOrder={(values) =>
        values.origin && values.target
          ? {
              line: `${prefix} ${values.origin} ${values.target}`,
              comment: t('orders.comment.ambush', { card, origin: values.origin, target: values.target }),
            }
          : null
      }
      onConfirm={add}
    />
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
  tunnel: 'SO',
  ambush: 'EM',
  assassination: 'AS',
  justice: 'JU',
}

export function shortCode(kind: CardKind): string {
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
    t(`orders.holder.${noble.religiousTitle ?? 'abbess'}` as MessageKey, {
      name: noble.firstName ?? noble.name,
    })

  const launchers: React.ReactNode[] = []
  for (const noble of clerics) {
    const here = regionSeedOf(state, noble.location)
    const ritual = rebels.filter((entry) => here && regionSeedOf(state, entry.id) === here)
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
    const modes: OrderFieldOption[] = []
    if (ritual.length > 0) modes.push({ value: 'rite', label: t('orders.appease.rite') })
    if (paid.length > 0) modes.push({ value: 'paid', label: t('orders.appease.paid') })
    if (modes.length === 0) continue
    launchers.push(
      <OrderLauncher
        key={`appease-${noble.code}`}
        label={`${holder(noble)} · ${t('orders.appease')}`}
        title={t('orders.appease')}
        description={t('orders.appeaseHelp')}
        hint={(values) => {
          if (values.mode !== 'paid') return t('orders.appease.riteHelp')
          const size = state.territories.find((entry) => entry.id === values.territory)?.army?.size ?? 0
          return size > 0 ? t('orders.hint.cost', { cost: costBase ** size }) : null
        }}
        fields={[
          { key: 'mode', label: t('orders.field.appeaseMode'), options: modes },
          {
            key: 'territory',
            label: t('orders.field.territory'),
            options: (values) => (values.mode === 'paid' ? paid : ritual).map(rebelOption),
          },
        ]}
        buildOrder={(values) =>
          values.territory
            ? {
                line: `P ${values.mode === 'paid' ? 'AP' : 'AG'} ${noble.code} ${values.territory}`,
                comment: t(
                  values.mode === 'paid' ? 'orders.comment.appeasePaid' : 'orders.comment.appeaseRite',
                  { name: noble.name, territory: values.territory },
                ),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }
  if (launchers.length === 0) return null
  return <div className="flex flex-wrap gap-1.5">{launchers}</div>
}
