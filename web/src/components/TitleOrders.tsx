import { OrderLauncher, type OrderFieldOption } from '@/components/OrderDialog'
import { formatCardLabel } from '@/lib/card-hand'
import { appendDraftLine, draftLines } from '@/lib/winter-draft'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey, Translate } from '@/i18n/messages'
import type { CardKind, Noble, PlayerId, Region, StateData } from '@/types'

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

/**
 * Orders granted by the titles of the player (pope, bishop, astrologer,
 * fiefs), each configured in a dialog that appends a commented line.
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
  const add = (line: string) => onWinterChange(appendDraftLine(winterDraft, line))
  const lines = draftLines(winterDraft)
  const excommunications = state.excommunicated ?? []
  const excommunicatedCodes = new Set(excommunications.map((entry) => entry.noble))
  const own = state.nobles.filter((noble) => noble.owner === player)

  const pope = state.nobles.find((noble) => noble.code === state.pope)
  const isPope =
    pope !== undefined &&
    pope.owner === player &&
    pope.status !== 'dungeon' &&
    !excommunicatedCodes.has(pope.code)
  const excommunicable = state.nobles.filter(
    (noble) => noble.code !== pope?.code && !excommunicatedCodes.has(noble.code),
  )
  const liftable = state.nobles.filter((noble) =>
    excommunications.some((entry) => entry.noble === noble.code && entry.reason === 'papal'),
  )

  const cardinals = new Set(state.cardinals ?? [])
  const buyable = own.filter(
    (noble) =>
      (state.bishoprics ?? []).some((bishopric) => bishopric.bishop === noble.code) &&
      !cardinals.has(noble.code) &&
      !excommunicatedCodes.has(noble.code),
  )

  const forecast = state.calamityForecast ?? []
  const astrologers = own.filter((noble) => (noble.dignities ?? []).includes('astrologer'))

  const fiefs = state.fiefs ?? []
  const vacantFiefs = fiefs.filter((fief) => fief.owner === player && !fief.holder)

  const launchers: React.ReactNode[] = []

  if (isPope) {
    launchers.push(
      <OrderLauncher
        key="x-e"
        label={t('orders.excommunicate')}
        title={t('orders.excommunicate')}
        disabled={excommunicable.length === 0 || lines.some((line) => line.startsWith('X E '))}
        fields={[
          { key: 'noble', label: t('orders.field.excommunicated'), options: excommunicable.map(nobleOption) },
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
      <OrderLauncher
        key="x-l"
        label={t('orders.liftExcommunication')}
        title={t('orders.liftExcommunication')}
        disabled={liftable.length === 0}
        fields={[
          { key: 'noble', label: t('orders.field.excommunicated'), options: liftable.map(nobleOption) },
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

  if (buyable.length > 0) {
    launchers.push(
      <OrderLauncher
        key="n-c"
        label={t('orders.buyCardinal')}
        title={t('orders.buyCardinal')}
        fields={[{ key: 'noble', label: t('orders.field.bishop'), options: buyable.map(nobleOption) }]}
        buildOrder={(values) =>
          values.noble
            ? {
                line: `N C ${values.noble}`,
                comment: t('orders.comment.buyCardinal', { name: nobleName(state, values.noble) }),
              }
            : null
        }
        onConfirm={add}
      />,
    )
  }

  if (astrologers.length > 0 && forecast.length > 0) {
    const positions: OrderFieldOption[] = forecast.map((kind, index) => ({
      value: String(index + 1),
      label: `${index + 1} · ${t(`card.${kind}` as MessageKey)}`,
    }))
    launchers.push(
      <OrderLauncher
        key="v-c"
        label={t('orders.calamityVeto')}
        title={t('orders.calamityVeto')}
        fields={[
          { key: 'noble', label: t('orders.field.astrologer'), options: astrologers.map(nobleOption) },
          { key: 'first', label: t('orders.field.firstCalamity'), options: positions },
          {
            key: 'second',
            label: t('orders.field.secondCalamity'),
            options: [{ value: '', label: '—' }, ...positions],
          },
        ]}
        buildOrder={(values) => {
          if (!values.noble || values.first === values.second) return null
          const chosen = [values.first, values.second].filter((value) => value !== '')
          return {
            line: `V C ${values.noble} ${chosen.join(' ')}`,
            comment: t('orders.comment.calamityVeto', { positions: chosen.join(', ') }),
          }
        }}
        onConfirm={add}
      />,
    )
  }

  if (own.length > 0) {
    launchers.push(
      <OrderLauncher
        key="t-f"
        label={t('orders.foundFief')}
        title={t('orders.foundFief')}
        description={t('orders.foundFiefHelp')}
        fields={[
          { key: 'noble', label: t('orders.field.holder'), options: own.map(nobleOption) },
          {
            key: 'territories',
            label: t('orders.field.fiefTerritories'),
            placeholder: 'AAA BBB CCC',
            maxLength: 80,
          },
        ]}
        buildOrder={(values) => {
          const codes = values.territories.trim().toUpperCase().split(/\s+/).filter(Boolean)
          if (!values.noble || codes.length < 3 || codes.some((code) => !/^[A-Z]{3}$/.test(code))) {
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

function specialTargetField(
  kind: CardKind,
  state: StateData,
  regions: Region[],
  t: Translate,
): { label: string; options: OrderFieldOption[] } {
  switch (kind) {
    case 'revolt':
      return {
        label: t('orders.field.territory'),
        options: state.territories.map((territory) => codeOption(territory.id)),
      }
    case 'seigneurial_tax':
      return {
        label: t('orders.field.fief'),
        options: (state.fiefs ?? []).map((fief) => codeOption(fief.capital)),
      }
    case 'trial':
      return { label: t('orders.field.trialTarget'), options: state.nobles.map(nobleOption) }
    default:
      return { label: t('orders.field.region'), options: regionOptions(regions) }
  }
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
        const target = specialTargetField(kind, state, regions, t)
        return (
          <OrderLauncher
            key={kind}
            label={t('orders.playSpecialCard', { card })}
            title={t('orders.playSpecialCard', { card })}
            disabled={used(kind, 'P') >= copies(kind) || target.options.length === 0}
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
