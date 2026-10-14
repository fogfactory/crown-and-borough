import { useMemo, type ReactNode } from 'react'
import { IconBuildingChurch, IconCrown, IconHierarchy2 } from '@tabler/icons-react'

import { NobleHolder } from '@/components/NobleHolder'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import type { MessageKey } from '@/i18n/messages'
import { nobleAttributesByCode } from '@/lib/noble-attributes'
import { royalIssuerOf } from '@/lib/royal'
import type { FiefTitle, StateData, Territory } from '@/types'

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

interface RealmPanelProps {
  state: StateData
  /** Map territories, to write the fiefs' capitals in full. */
  territories?: Territory[]
  /** Opens the lineage and alliances window. */
  onOpenLineage?: () => void
}

/**
 * Realm bookmark (charte-graphique.md § Marque-page Royaume): the crown
 * (politique.md) and the Church (pope, cardinals, one bishop per bishopric),
 * which concern the whole kingdom.
 */
export function RealmPanel({ state, territories = [], onOpenLineage }: RealmPanelProps) {
  const { t } = useLanguage()
  const attributes = useMemo(() => nobleAttributesByCode(state), [state])
  const holder = (code: string | undefined) => (
    <NobleHolder
      code={code}
      nobles={state.nobles}
      players={state.players}
      attributes={attributes}
    />
  )

  const crownFiefs = (state.fiefs ?? []).filter((fief) => fief.crown)
  const hasCrown = Boolean(state.king || state.queen || crownFiefs.length > 0)
  const issuer = royalIssuerOf(state)
  const regent = issuer && issuer.code !== state.king ? issuer : undefined

  const bishoprics = state.bishoprics ?? []
  const cardinals = state.cardinals ?? []
  const hasChurch = Boolean(state.pope || bishoprics.length > 0 || cardinals.length > 0)

  const territoryName = (id: string) =>
    territories.find((territory) => territory.id === id)?.name ?? id

  return (
    <div className="space-y-5">
      {!hasCrown && !hasChurch && (
        <p className="text-sm italic text-ink-500">{t('realm.empty')}</p>
      )}
      {hasCrown && (
        <RealmSection
          icon={<IconCrown aria-hidden="true" className="size-4" />}
          title={t('crown.title')}
          dataAttribute="data-realm-crown"
        >
          <RealmRow label={t('crown.king')}>{holder(state.king)}</RealmRow>
          {state.queen && (
            <RealmRow
              label={t(state.queenDowager ? 'crown.queenDowager' : 'crown.queen')}
            >
              {holder(state.queen)}
            </RealmRow>
          )}
          {regent && (
            <RealmRow label={t('crown.regent')} data-crown-regent>
              {holder(regent.code)}
            </RealmRow>
          )}
          {crownFiefs.length > 0 && (
            <RealmRow label={t('crown.fiefs')}>
              <ul className="space-y-0.5">
                {crownFiefs.map((fief) => (
                  <li key={fief.capital} data-crown-fief={fief.capital}>
                    {t('map.fiefLabel', {
                      title: t(FIEF_TITLE_KEYS[fief.title]),
                      capital: territoryName(fief.capital),
                    })}{' '}
                    <span className="font-mono text-ink-500">({fief.capital})</span>
                    {' · '}
                    {t('realm.fiefTerritories', { count: fief.territories.length })}
                  </li>
                ))}
              </ul>
            </RealmRow>
          )}
        </RealmSection>
      )}
      {hasChurch && (
        <RealmSection
          icon={<IconBuildingChurch aria-hidden="true" className="size-4" />}
          title={t('realm.church')}
          dataAttribute="data-realm-church"
        >
          <RealmRow label={t('map.hierarchyPope')}>{holder(state.pope)}</RealmRow>
          <RealmRow label={t('map.hierarchyCardinals')}>
            {cardinals.length === 0 ? (
              holder(undefined)
            ) : (
              <ul className="space-y-0.5">
                {cardinals.map((code) => (
                  <li key={code}>{holder(code)}</li>
                ))}
              </ul>
            )}
          </RealmRow>
          {bishoprics.length > 0 && (
            <RealmRow label={t('map.hierarchyBishops')}>
              <ul className="space-y-0.5">
                {bishoprics.map((bishopric) => (
                  <li key={bishopric.region} data-bishopric={bishopric.region}>
                    <span className="font-semibold">{bishopric.name}</span> :{' '}
                    {holder(bishopric.bishop)}
                  </li>
                ))}
              </ul>
            </RealmRow>
          )}
        </RealmSection>
      )}
      {onOpenLineage && (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={onOpenLineage}
        >
          <IconHierarchy2 aria-hidden="true" className="size-4" />
          {t('realm.openLineage')}
        </Button>
      )}
    </div>
  )
}

function RealmSection({
  icon,
  title,
  dataAttribute,
  children,
}: {
  icon: ReactNode
  title: string
  dataAttribute: string
  children: ReactNode
}) {
  return (
    <section {...{ [dataAttribute]: true }} className="space-y-2">
      <h3 className="flex items-center gap-2 border-b border-line pb-1 font-display text-base font-semibold text-ink-900">
        {icon}
        {title}
      </h3>
      <dl className="space-y-2 text-sm text-ink-700">{children}</dl>
    </section>
  )
}

function RealmRow({
  label,
  children,
  ...rest
}: {
  label: string
  children: ReactNode
  'data-crown-regent'?: boolean
}) {
  return (
    <div {...rest}>
      <dt className="text-xs font-semibold uppercase tracking-[0.12em] text-ink-500">
        {label}
      </dt>
      <dd className="mt-0.5">{children}</dd>
    </div>
  )
}
