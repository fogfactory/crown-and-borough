import { ShieldIcon } from '@/components/ArmsIcons'
import {
  BLOCK_FRAME,
  BLOCK_HEADER,
  RegionBlocks,
  regionSeedOf,
} from '@/components/RegionBlocks'
import { playerArms } from '@/lib/heraldry'
import { FIEF_RANK } from '@/lib/lineage'
import {
  groupWinterReport,
  houseRank,
  isHalved,
  type WinterHouseKind,
} from '@/lib/winter-report'
import { TERRAIN_LABEL_KEYS } from '@/components/MapLegend'
import { MANEUVER_REST, NEUTRAL_PLAYER_ID } from '@/types'
import { Fragment, useId, useState, type ReactNode, type SyntheticEvent } from 'react'

import {
  IconBell,
  IconBuildingChurch,
  IconBuildingWarehouse,
  IconBuildingCottage,
  IconBuildingWindTurbine,
  IconChevronDown,
  IconCloudStorm,
  IconCoins,
  IconCrown,
  IconHeart,
  IconMapPin,
  IconScript,
  IconSkull,
  IconSnowflake,
  IconSwords,
} from '@tabler/icons-react'

import {
  NobleRef,
  RegionRef,
  TerritoryRef,
  useMapHighlight,
  useMapLinks,
} from '@/components/MapLinks'
import type { EventMarks } from '@/lib/map-frame'
import { legacyMovements } from '@/lib/legacy-movements'
import { formatOrderLabel, orderTargets } from '@/lib/order-label'
import { joinNodes, richTranslate } from '@/lib/rich-text'
import { formatCardLabel } from '@/lib/card-hand'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { useLanguage } from '@/i18n/LanguageContext'
import { useMediaQuery } from '@/lib/use-media-query'
import type { MessageKey, Translate } from '@/i18n/messages'
import type {
  FiefTitle,
  InfraType,
  MapData,
  Outcome,
  Player,
  PlayerId,
  ReportArmy,
  CardReport,
  CombatReport,
  ConsumptionReport,
  FiefReport,
  MarriageReport,
  MillReport,
  PlayerReport,
  ArmyMovement,
  OrderReport,
  ReceptionReport,
  WinterStockReport,
  ProductionReport,
  IncomeReport,
  SeasonEffectReport,
  TurnReport,
  Dignity,
  WinterInvestmentReport,
  WinterOrder,
} from '@/types'
import { DEFAULT_PLAYER_COLOR, NEUTRAL_ARMY_COLOR } from '@/lib/map-palette'

const FIEF_TITLE_KEYS: Record<FiefTitle, MessageKey> = {
  barony: 'fief.title.barony',
  county: 'fief.title.county',
  marquisate: 'fief.title.marquisate',
  duchy: 'fief.title.duchy',
}

interface ReportPanelProps {
  report: TurnReport | null
  map: MapData | null
  players: Player[]
  /** Player whose house is listed first and opened in the economy card. */
  viewerId?: PlayerId
}

const WINTER_INFRA_SYMBOLS: Partial<Record<InfraType, string>> = {
  mill: 'M',
  castle: 'C',
  supply_depot: 'D',
  village: 'V',
}

const OUTCOME_KEYS: Record<Outcome, MessageKey> = {
  success: 'reports.outcome.success',
  failure: 'reports.outcome.failure',
  invalid: 'reports.outcome.invalid',
}

const REASON_KEYS: Record<string, MessageKey> = {
  insufficient_resources: 'reports.reason.insufficient_resources',
  transferred: 'reports.reason.transferred',
  transferred_partially: 'reports.reason.transferred_partially',
  famished_sender: 'reports.reason.famished_sender',
  famished_attacker: 'reports.reason.famished_attacker',
  host_disappeared: 'reports.reason.host_disappeared',
  transfer_over_capacity: 'reports.reason.transfer_over_capacity',
  transfer_path_blocked: 'reports.reason.transfer_path_blocked',
  transfer_target_occupied: 'reports.reason.transfer_target_occupied',
  invalid_transfer_destination: 'reports.reason.invalid_transfer_destination',
  transfer_source_not_controlled: 'reports.reason.transfer_source_not_controlled',
  transfer_same_territory: 'reports.reason.transfer_same_territory',
  transfer_source_not_settlement: 'reports.reason.transfer_source_not_settlement',
  transfer_target_not_settlement: 'reports.reason.transfer_target_not_settlement',
  invalid_transfer_amount: 'reports.reason.invalid_transfer_amount',
  territory_not_controlled: 'reports.reason.territory_not_controlled',
  noble_requires_owned_army: 'reports.reason.noble_requires_owned_army',
  noble_requires_settlement: 'reports.reason.noble_requires_settlement',
  noble_limit_reached: 'reports.reason.noble_limit_reached',
  noble_draw_already_used: 'reports.reason.noble_draw_already_used',
  noble_deck_empty: 'reports.reason.noble_deck_empty',
  hand_limit_reached: 'reports.reason.hand_limit_reached',
  card_not_in_hand: 'reports.reason.card_not_in_hand',
  justice_king: 'reports.reason.justice_king',
  justice_unknown_noble: 'reports.reason.justice_unknown_noble',
  noble_already_bastard: 'reports.reason.noble_already_bastard',
  claim_requires_marriage: 'reports.reason.claim_requires_marriage',
  claim_on_own_noble: 'reports.reason.claim_on_own_noble',
  claim_by_bastard: 'reports.reason.claim_by_bastard',
  no_claim_to_renounce: 'reports.reason.no_claim_to_renounce',
  noble_not_contested: 'reports.reason.noble_not_contested',
  claim_already_staked: 'reports.reason.claim_already_staked',
  claim_against_married_eon: 'reports.reason.claim_against_married_eon',
  claim_lost_married_eon: 'reports.reason.claim_lost_married_eon',
  troop_requires_adjacent_noble: 'reports.reason.troop_requires_adjacent_noble',
  noble_not_prisoner: 'reports.reason.noble_not_prisoner',
  noble_not_held: 'reports.reason.noble_not_held',
  noble_not_controlled: 'reports.reason.noble_not_controlled',
  no_army_at_destination: 'reports.reason.no_army_at_destination',
  transfer_to_self: 'reports.reason.transfer_to_self',
  transfer_status_to_owner: 'reports.reason.transfer_status_to_owner',
  marriage_not_reciprocated: 'reports.reason.marriage_not_reciprocated',
  election_not_open: 'reports.reason.election_not_open',
  cardinal_requires_bishop: 'reports.reason.cardinal_requires_bishop',
  not_pope: 'reports.reason.not_pope',
  not_married: 'reports.reason.not_married',
  dissolution_not_requested: 'reports.reason.dissolution_not_requested',
  dissolution_not_granted: 'reports.reason.dissolution_not_granted',
  cannot_excommunicate_self: 'reports.reason.cannot_excommunicate_self',
  excommunication_limit: 'reports.reason.excommunication_limit',
  already_excommunicated: 'reports.reason.already_excommunicated',
  excommunication_slot_taken: 'reports.reason.excommunication_slot_taken',
  not_excommunicated: 'reports.reason.not_excommunicated',
  excommunication_not_liftable: 'reports.reason.excommunication_not_liftable',
  noble_not_witch: 'reports.reason.noble_not_witch',
  ritual_already_used: 'reports.reason.ritual_already_used',
  ritual_invalid_target: 'reports.reason.ritual_invalid_target',
  ritual_no_region: 'reports.reason.ritual_no_region',
  dignity_exclusive: 'reports.reason.dignity_exclusive',
  dignity_hidden_own_only: 'reports.reason.dignity_hidden_own_only',
  dignity_female_only: 'reports.reason.dignity_female_only',
  calamity_not_forecast: 'reports.reason.calamity_not_forecast',
  calamity_veto_already_used: 'reports.reason.calamity_veto_already_used',
  noble_not_astrologer: 'reports.reason.noble_not_astrologer',
  already_cardinal: 'reports.reason.already_cardinal',
  cardinal_cap_reached: 'reports.reason.cardinal_cap_reached',
  not_cardinal: 'reports.reason.not_cardinal',
  inquiry_limit: 'reports.reason.inquiry_limit',
  trial_limit: 'reports.reason.trial_limit',
  candidate_not_eligible: 'reports.reason.candidate_not_eligible',
  candidacy_already_filed: 'reports.reason.candidacy_already_filed',
  unknown_candidate: 'reports.reason.unknown_candidate',
  vote_already_cast: 'reports.reason.vote_already_cast',
  no_voice: 'reports.reason.no_voice',
  marriage_pending: 'reports.reason.marriage_pending',
  marriage_refused: 'reports.reason.marriage_refused',
  marriage_same_owner: 'reports.reason.marriage_same_owner',
  marriage_same_sex: 'reports.reason.marriage_same_sex',
  marriage_forbidden: 'reports.reason.marriage_forbidden',
  noble_not_owned: 'reports.reason.noble_not_owned',
  noble_not_free: 'reports.reason.noble_not_free',
  noble_already_married: 'reports.reason.noble_already_married',
  no_capital: 'reports.reason.no_capital',
  no_army_at_capital: 'reports.reason.no_army_at_capital',
  structure_present: 'reports.reason.structure_present',
  village_already_fortified: 'reports.reason.village_already_fortified',
  mill_requires_productive_neighbor: 'reports.reason.mill_requires_productive_neighbor',
  capital_requires_controlled_castle: 'reports.reason.capital_requires_controlled_castle',
  attack_wins: 'reports.reason.attack_wins',
  defense_holds: 'reports.reason.defense_holds',
  standoff: 'reports.reason.standoff',
  non_adjacent_destination: 'reports.reason.non_adjacent_destination',
  allied_destination: 'reports.reason.allied_destination',
  combat_lost: 'reports.reason.combat_lost',
  dislodged: 'reports.reason.dislodged',
  support_cut: 'reports.reason.support_cut',
  support_void: 'reports.reason.support_void',
  join_move: 'reports.reason.join_move',
  join_attack_arrival: 'reports.reason.join_attack_arrival',
  enemy_destination: 'reports.reason.enemy_destination',
  disperse_partial: 'reports.reason.disperse_partial',
  disperse_complete: 'reports.reason.disperse_complete',
  disperse_chain_discarded: 'reports.reason.disperse_chain_discarded',
  disperse_no_residual: 'reports.reason.disperse_no_residual',
  disperse_noble_left_behind: 'reports.reason.disperse_noble_left_behind',
  no_retreat_destination: 'reports.reason.no_retreat_destination',
  retreat_collision: 'reports.reason.retreat_collision',
  position_mismatch: 'reports.reason.position_mismatch',
  missing_pending_disperse: 'reports.reason.missing_pending_disperse',
  missing_disperse_residual: 'reports.reason.missing_disperse_residual',
  join_not_terminal: 'reports.reason.join_not_terminal',
  invalid_hold_shape: 'reports.reason.invalid_hold_shape',
  invalid_pillage_shape: 'reports.reason.invalid_pillage_shape',
  no_infrastructure: 'reports.reason.no_infrastructure',
  unknown_order_type: 'reports.reason.unknown_order_type',
  invalid_target_shape: 'reports.reason.invalid_target_shape',
  invalid_support_shape: 'reports.reason.invalid_support_shape',
  unknown_support_target: 'reports.reason.unknown_support_target',
  invalid_defensive_support: 'reports.reason.invalid_defensive_support',
  invalid_offensive_support: 'reports.reason.invalid_offensive_support',
  non_adjacent_disperse_destination: 'reports.reason.non_adjacent_disperse_destination',
  invalid_disperse_assignment_destination:
    'reports.reason.invalid_disperse_assignment_destination',
  duplicate_disperse_wildcard: 'reports.reason.duplicate_disperse_wildcard',
  invalid_disperse_noble_assignment: 'reports.reason.invalid_disperse_noble_assignment',
  join_convergence: 'reports.reason.join_convergence',
  join_enemy_convergence: 'reports.reason.join_enemy_convergence',
  attacked_destination: 'reports.reason.attacked_destination',
  attacked_origin: 'reports.reason.attacked_origin',
  join_host: 'reports.reason.join_host',
  join_pair: 'reports.reason.join_pair',
  disperse_friendly_fusion: 'reports.reason.disperse_friendly_fusion',
  maneuver_complete: 'reports.reason.maneuver_complete',
  maneuver_partial: 'reports.reason.maneuver_partial',
  maneuver_failed: 'reports.reason.maneuver_failed',
  maneuver_chain_discarded: 'reports.reason.maneuver_chain_discarded',
  maneuver_friendly_fusion: 'reports.reason.maneuver_friendly_fusion',
  invalid_maneuver_noble_assignment: 'reports.reason.invalid_maneuver_noble_assignment',
  support_applied: 'reports.reason.support_applied',
  unresolved_order: 'reports.reason.unresolved_order',
  unknown_infrastructure: 'reports.reason.unknown_infrastructure',
  pillaged: 'reports.reason.pillaged',
  pillaged_downgraded: 'reports.reason.pillaged_downgraded',
  pillaged_unfortified: 'reports.reason.pillaged_unfortified',
  pillaged_ruined: 'reports.reason.pillaged_ruined',
  ruined_infrastructure: 'reports.reason.ruined_infrastructure',
  fief_capital_ruined: 'reports.reason.fief_capital_ruined',
  capital_castle_ruined: 'reports.reason.capital_castle_ruined',
  ruined_fief_capital_rebuild_only: 'reports.reason.ruined_fief_capital_rebuild_only',
  army_destroyed: 'reports.reason.army_destroyed',
  held: 'reports.reason.held',
  disperse_summary: 'reports.reason.disperse_summary',
  invalid_winter_order: 'reports.reason.invalid_winter_order',
  unknown_territory: 'reports.reason.unknown_territory',
  territory_occupied_by_other_player: 'reports.reason.territory_occupied_by_other_player',
  invalid_infrastructure: 'reports.reason.invalid_infrastructure',
  unknown_noble: 'reports.reason.unknown_noble',
  mill_max_level_reached: 'reports.reason.mill_max_level_reached',
  bad_weather: 'reports.reason.bad_weather',
  disperse_residual_dislodged: 'reports.reason.disperse_residual_dislodged',
  invalid_transfer_shape: 'reports.reason.invalid_transfer_shape',
  no_available_first_name: 'reports.reason.no_available_first_name',
  fief_holder_not_owned: 'reports.reason.fief_holder_not_owned',
  succession_rank_blocked: 'reports.reason.succession_rank_blocked',
  fief_duplicate_territory: 'reports.reason.fief_duplicate_territory',
  fief_too_small: 'reports.reason.fief_too_small',
  fief_capital_requires_castle: 'reports.reason.fief_capital_requires_castle',
  fief_territory_already_in_fief: 'reports.reason.fief_territory_already_in_fief',
  fief_territory_occupied_by_other_player:
    'reports.reason.fief_territory_occupied_by_other_player',
  fief_not_contiguous: 'reports.reason.fief_not_contiguous',
  fief_not_found: 'reports.reason.fief_not_found',
  fief_not_owned: 'reports.reason.fief_not_owned',
  fief_not_vacant: 'reports.reason.fief_not_vacant',
  annex_not_titular: 'reports.reason.annex_not_titular',
  annex_already_member: 'reports.reason.annex_already_member',
  annex_not_adjacent: 'reports.reason.annex_not_adjacent',
  annex_not_occupied: 'reports.reason.annex_not_occupied',
  annex_capital: 'reports.reason.annex_capital',
  annex_royal_accord_required: 'reports.reason.annex_royal_accord_required',
  annex_donor_too_small: 'reports.reason.annex_donor_too_small',
  rank_not_titular: 'reports.reason.rank_not_titular',
  rank_max: 'reports.reason.rank_max',
  rank_already_changed: 'reports.reason.rank_already_changed',
  royal_fief_not_crown: 'reports.reason.royal_fief_not_crown',
  royal_not_issuer: 'reports.reason.royal_not_issuer',
  royal_order_limit: 'reports.reason.royal_order_limit',
  royal_accord_unused: 'reports.reason.royal_accord_unused',
  royal_accord_pending: 'reports.reason.royal_accord_pending',
  capital_castle_lost: 'reports.reason.capital_castle_lost',
  fief_auto_assigned_default_holder: 'reports.reason.fief_auto_assigned_default_holder',
  prosperity_fief: 'reports.reason.prosperity_fief',
  prosperity_controlled: 'reports.reason.prosperity_controlled',
  prosperity_free: 'reports.reason.prosperity_free',
  prosperity_depot_upgraded: 'reports.reason.prosperity_depot_upgraded',
  prosperity_mill_upgraded: 'reports.reason.prosperity_mill_upgraded',
  prosperity_renovated: 'reports.reason.prosperity_renovated',
  prosperity_leveled_up: 'reports.reason.prosperity_leveled_up',
}

const RECEPTION_REASON_KEYS: Record<string, MessageKey> = {
  'error.assignment.no_army': 'reports.reason.reception.noArmy',
  'error.assignment.army_not_owned': 'reports.reason.reception.armyNotOwned',
  'error.assignment.noble_dungeon': 'reports.reason.reception.nobleDungeon',
  'error.assignment.emission_capacity': 'reports.reason.reception.emissionCapacity',
  'error.assignment.noble_unknown': 'reports.reason.reception.invalid',
  'error.assignment.pending_disperse': 'reports.reason.reception.invalid',
  'error.assignment.chain_id_in_use': 'reports.reason.reception.invalid',
  'error.assignment.chain_validation': 'reports.reason.reception.invalid',
}

// Raw trigram, for typed order lines (`G ABC DEF 3`) only.
function territoryCode(
  map: MapData | null,
  id: string | undefined,
  t: Translate,
): string {
  if (!id) return '—'
  return (
    map?.territories.find((candidate) => candidate.id === id)?.id ??
    t('reports.unknownTerritory')
  )
}

// Display label: full commune name, trigram as a complement, linked to the
// map (charte-graphique.md § Identifiants et liens vers la carte).
function territoryLabel(
  map: MapData | null,
  id: string | undefined,
  t: Translate,
  Ref: typeof TerritoryRef | typeof RegionRef = TerritoryRef,
): ReactNode {
  if (!id) return '—'
  const territory = map?.territories.find((candidate) => candidate.id === id)
  if (!territory) return t('reports.unknownTerritory')
  return <Ref id={territory.id} name={territory.name} />
}

/** Territories of the region seeded at `seed`, or the seed alone off the map. */
function regionTerritories(map: MapData | null, seed: string): string[] {
  return map?.regions?.find((region) => region.seed === seed)?.territories ?? [seed]
}

// A region is named by its seed territory; hovering the link lights it all.
function regionLabel(
  map: MapData | null,
  seed: string | undefined,
  t: Translate,
): ReactNode {
  return territoryLabel(map, seed, t, RegionRef)
}

// Display label of a noble: full name first, trigram as a complement, linked
// to the territory where it stands when the viewer knows it.
function nobleLabel(code: string | undefined, name: string | undefined): ReactNode {
  if (!code) return name || '—'
  return <NobleRef code={code} name={name} />
}

function playerLabel(
  players: Player[],
  playerId: PlayerId | undefined,
  t: Translate,
): string {
  if (!playerId) return t('reports.unknownPlayer')
  return players.find((player) => player.id === playerId)?.name ?? playerId
}

function playerColor(players: Player[], playerId?: PlayerId): string {
  return players.find((player) => player.id === playerId)?.color ?? DEFAULT_PLAYER_COLOR
}

/**
 * Income line destination label: a fief line prints its title and capital
 * (`{title} of {capital}`) instead of the bare destination trigram, since
 * several lines can otherwise share the same destination code (a fief whose
 * capital is also the player's own capital, titres.md #196).
 */
function incomeDestinationLabel(
  line: IncomeReport,
  map: MapData | null,
  t: Translate,
): ReactNode {
  if (line.fief && line.title) {
    const capital = map?.territories.find((candidate) => candidate.id === line.fief)
    return richTranslate(t, 'map.fiefLabel', {
      title: t(FIEF_TITLE_KEYS[line.title]),
      capital: capital ? (
        <TerritoryRef id={capital.id} name={capital.name} lights={line.fiefTerritories} />
      ) : (
        t('reports.unknownTerritory')
      ),
    })
  }
  return territoryLabel(map, line.destination, t)
}

function playerMarker(players: Player[], playerId: PlayerId | undefined, t: Translate) {
  return (
    <span
      role="img"
      className="inline-block size-2.5 shrink-0 rounded-full border border-ink-900/30 shadow-inner"
      style={{ backgroundColor: playerColor(players, playerId) }}
      aria-label={t('app.colorOf', { player: playerLabel(players, playerId, t) })}
    />
  )
}

function outcomeClass(outcome: Outcome): string {
  return outcome === 'success' ? 'text-success-700' : 'text-brass-700'
}

function outcomeLabel(outcome: Outcome, t: Translate): string {
  return t(OUTCOME_KEYS[outcome])
}

function armyDescription(
  report: TurnReport,
  map: MapData | null,
  armyID: string,
  t: Translate,
): ReactNode {
  const army: ReportArmy | undefined = report.players
    .flatMap((player) => player.armies)
    .find((candidate) => candidate.id === armyID)
  if (!army) return t('reports.unknownArmy')
  return richTranslate(t, 'reports.armyDescription', {
    owner: army.owner,
    territory: territoryLabel(map, army.territory, t),
  })
}

function formatReportOrderLabel(
  reportOrder: TurnReport['orders'][number],
  map: MapData | null,
  t: Translate,
): string {
  const targets =
    reportOrder.targets ?? (reportOrder.target ? [reportOrder.target] : undefined)
  return formatOrderLabel({
    type: reportOrder.type ?? 'hold',
    position: territoryCode(map, reportOrder.source, t),
    targets: targets?.map((target) => territoryCode(map, target, t)),
    counts: reportOrder.counts,
    nobleAssignments: reportOrder.nobleAssignments,
    liaison: reportOrder.liaison ?? 'single',
    amount: reportOrder.amount,
  })
}

function winterOrderLabel(order: WinterOrder, map: MapData | null, t: Translate): string {
  const territory = territoryCode(map, order.territory, t)
  switch (order.type) {
    case 'recruit_noble':
      return order.cardCode ? `R N ${order.cardCode} ${territory}` : `R N ${territory}`
    case 'draw_noble':
      return 'T N'
    case 'discard_noble_card':
      return `D C ${order.cardCode ?? '—'}`
    case 'dissolve_marriage':
      return `X D ${order.nobleCode ?? '—'}`
    case 'inquiry':
      return `Q ${order.nobleCode ?? '—'} ${order.targetCode ?? '—'}`
    case 'trial':
      return `J ${order.nobleCode ?? '—'} ${order.targetCode ?? '—'}`
    case 'claim':
      return `C N ${order.nobleCode ?? '—'} ${order.spouseCode ?? '—'}`
    case 'play_dignity':
      return `D N ${order.nobleCode ?? '—'} ${order.cardCode ?? '—'}${order.territory ? ` ${order.territory}` : ''}`
    case 'calamity_veto':
      return `V C ${order.nobleCode ?? '—'} ${(order.indices ?? []).join(' ')}`.trim()
    case 'ritual':
      return `S R ${order.nobleCode ?? '—'}`
    case 'recruit_troop':
      return `R T ${territory}`
    case 'build':
      return `C ${WINTER_INFRA_SYMBOLS[order.infrastructureType ?? 'mill'] ?? '?'} ${territory}`
    case 'elect_capital':
      return `E C ${territory}`
    case 'transfer_noble':
      return `H N ${order.nobleCode ?? '—'} ${territory}${order.status ? ` ${order.status === 'dungeon' ? 'P' : 'O'}` : ''}`
    case 'hostage':
      return `O N ${order.nobleCode ?? '—'}`
    case 'dungeon':
      return `P N ${order.nobleCode ?? '—'}`
    case 'transfer':
      return `G ${territoryCode(map, order.source, t)} ${territoryCode(map, order.target, t)} ${order.amount ?? '—'}`
    case 'found_fief': {
      const group = order.territories?.length
        ? order.territories
        : order.territory
          ? [order.territory]
          : []
      return `T F ${order.nobleCode ?? '—'} ${group.map((id) => territoryCode(map, id, t)).join(' ')}`
    }
    case 'assign_fief':
      return `T A ${order.nobleCode ?? '—'} ${territory}`
    case 'annex_territory':
      return `T X ${order.nobleCode ?? '—'} ${territory} ${territoryCode(map, order.target, t)}`
    case 'change_rank':
      return `T R ${order.nobleCode ?? '—'} ${territory}`
    case 'renounce_claim':
      return `T C ${order.nobleCode ?? '—'}`
    case 'renounce_title':
      return `T D ${order.nobleCode ?? '—'} ${territory}`
    case 'royal_annex':
      return `W X ${order.nobleCode ?? '—'} ${territory} ${territoryCode(map, order.target, t)}`
    case 'royal_rank':
      return `W R ${order.nobleCode ?? '—'} ${territory}`
    case 'royal_found':
      return `W F ${order.nobleCode ?? '—'} ${territory}`
    case 'royal_grant':
      return `W G ${order.nobleCode ?? '—'} ${territory} ${order.targetCode ?? '—'}`
    case 'marriage':
      return `M N ${order.nobleCode ?? '—'} ${order.spouseCode ?? '—'}`
  }
}

// Code of the card that confers each dignity, as typed in `D N NNN CCC`.
const DIGNITY_CARD_CODES: Record<Dignity, string> = {
  bastard: 'BAS',
  d_arc: 'ARC',
  castellan: 'CTL',
  abbess: 'ABB',
  herbalist: 'HRB',
  astrologer: 'AST',
  chevalier_d_eon: 'EON',
  correspondent: 'COR',
  spy: 'ESP',
  poisoner: 'EMP',
  witch: 'SOR',
  cardinal: 'CAR',
}

function investmentLabel(
  investment: WinterInvestmentReport,
  map: MapData | null,
  t: Translate,
): string {
  if (investment.order) return winterOrderLabel(investment.order, map, t)
  const territory = territoryCode(map, investment.territory, t)
  switch (investment.kind) {
    case 'recruit':
      return investment.nobleCode
        ? `R N ${investment.nobleCode} ${territory}`
        : `R T ${territory}`
    case 'build':
      return `C ${WINTER_INFRA_SYMBOLS[investment.type ?? 'mill'] ?? '?'} ${territory}`
    case 'upgrade':
      return `C ${WINTER_INFRA_SYMBOLS[investment.type ?? 'mill'] ?? '?'} ${territory}`
    case 'fortify':
      // A fortify event always comes from a `C C` order (the syntax
      // fortifying a village instead of building a castle, #193): show it
      // with the castle symbol the player actually typed, not the village's.
      return `C ${WINTER_INFRA_SYMBOLS.castle} ${territory}`
    case 'capital_elected':
      return `E C ${territory}`
    case 'noble_transfer':
      return `H N ${investment.nobleCode ?? '—'} ${territory}`
    case 'transfer':
      return `G ${territoryCode(map, investment.source, t)} ${territoryCode(map, investment.target, t)} ${investment.amount ?? '—'}`
    case 'fief_founded': {
      const group = investment.territories?.length
        ? investment.territories
        : [investment.territory ?? '']
      return `T F ${investment.nobleCode ?? '—'} ${group.map((id) => territoryCode(map, id, t)).join(' ')}`
    }
    case 'fief_assigned':
    case 'fief_auto_assigned':
      return `T A ${investment.nobleCode ?? '—'} ${territory}`
    case 'fief_annexed':
      return `T X ${investment.nobleCode ?? '—'} ${territoryCode(map, investment.territories?.[0], t)} ${territory}`
    case 'fief_rank_changed':
      return `T R ${investment.nobleCode ?? '—'} ${territory}`
    case 'marriage_refused':
      return `M N ${investment.nobleCode ?? '—'}`
    case 'noble_draw':
      return 'T N'
    case 'noble_discard':
      // The public report does not name the discarded card.
      return 'D C'
    case 'inquiry':
      return `Q ${investment.nobleCode ?? '—'}`
    case 'trial_filed':
      return `J ${investment.nobleCode ?? '—'}`
    case 'claim':
      return `C N ${investment.nobleCode ?? '—'} ${investment.claimTarget ?? '—'}`
    case 'ritual':
      return `S R ${investment.nobleCode ?? '—'}`
    case 'dignity':
      return `D N ${investment.nobleCode ?? '—'} ${DIGNITY_CARD_CODES[investment.dignity ?? 'bastard']}`
    case 'prosperity_founded':
      return `${WINTER_INFRA_SYMBOLS.village} ${territoryCode(map, investment.target, t)} ← ${territoryCode(map, investment.source, t)}`
    default:
      return t('reports.winterOrder')
  }
}

function reportReason(
  reason: string | undefined,
  t: Translate,
  reasonKey?: string,
  reasonArgs?: unknown[],
  map: MapData | null = null,
): ReactNode {
  if (!reason && !reasonKey) return undefined
  // Reception reasons name a territory by its code: write it in full.
  const territory = () =>
    reasonArgs?.[0] ? territoryLabel(map, String(reasonArgs[0]), t) : '—'
  if (reasonKey === 'error.reception.concurrent') {
    return richTranslate(t, 'reports.reason.reception.concurrent', {
      territory: territory(),
      count: Number(reasonArgs?.[1] ?? 0),
      turn: Number(reasonArgs?.[2] ?? 0),
    })
  }
  if (reasonKey && RECEPTION_REASON_KEYS[reasonKey]) {
    const key = RECEPTION_REASON_KEYS[reasonKey]
    if (
      key === 'reports.reason.reception.noArmy' ||
      key === 'reports.reason.reception.armyNotOwned'
    ) {
      return richTranslate(t, key, { territory: territory() })
    }
    return t(key)
  }
  if (reason) {
    const base = reason.split(':', 1)[0]
    const key = REASON_KEYS[base]
    if (key) return t(key)
    return reason
  }
  return undefined
}

function winterDetails(
  investment: WinterInvestmentReport,
  map: MapData | null,
  t: Translate,
): ReactNode {
  if (investment.kind === 'ritual') {
    return richTranslate(
      t,
      `reports.ritual.${investment.reason ?? 'declared'}` as MessageKey,
      {
        region: territoryLabel(map, investment.territory, t),
      },
    )
  }
  if (investment.reason) return reportReason(investment.reason, t) ?? investment.reason
  if (investment.kind === 'claim' && investment.nobleName) {
    return richTranslate(t, 'reports.claimDetails', {
      heir: nobleLabel(investment.nobleCode, investment.nobleName),
      target: nobleLabel(investment.claimTarget, investment.claimTargetName),
    })
  }
  if (investment.nobleName) {
    return nobleLabel(investment.nobleCode, investment.nobleName)
  }
  if (investment.level) return t('reports.level', { level: investment.level })
  return territoryLabel(map, investment.territory, t)
}

function marriageLabel(
  marriage: MarriageReport,
  players: Player[],
  t: Translate,
): ReactNode {
  return richTranslate(
    t,
    marriage.outcome === 'success'
      ? 'reports.marriageConcluded'
      : marriage.outcome === 'dissolved'
        ? 'reports.marriageDissolved'
        : 'reports.marriageFailed',
    {
      noble: nobleLabel(marriage.nobleCode, marriage.nobleName),
      owner: playerLabel(players, marriage.owner, t),
      spouse: nobleLabel(marriage.spouseCode, marriage.spouseName),
      spouseOwner: playerLabel(players, marriage.spouseOwner, t),
    },
  )
}

function fiefEventLabel(
  fief: FiefReport,
  players: Player[],
  map: MapData | null,
  t: Translate,
): ReactNode {
  const capital = territoryLabel(map, fief.capital, t)
  const owner = playerLabel(players, fief.owner, t)
  const noble = fief.nobleName ?? '—'
  const exiled = fief.exiledName ?? '—'
  switch (fief.kind) {
    case 'fief_conquered':
      if (fief.reason === 'usurpation') {
        return richTranslate(t, 'reports.fiefUsurped', {
          capital,
          previousOwner: playerLabel(players, fief.previousOwner, t),
          owner,
          noble,
          exiled,
        })
      }
      if (fief.reason === 'claim_usurpation') {
        return richTranslate(t, 'reports.fiefClaimUsurped', {
          capital,
          owner,
          noble,
          exiled,
        })
      }
      if (fief.reason === 'restoration') {
        return richTranslate(t, 'reports.fiefRestored', { capital, owner, noble })
      }
      if (fief.reason === 'usurper_died') {
        return richTranslate(t, 'reports.fiefUsurperDied', { capital, owner, noble })
      }
      return richTranslate(t, 'reports.fiefConquered', {
        capital,
        previousOwner: playerLabel(players, fief.previousOwner, t),
        owner,
      })
    case 'fief_vacated':
      return richTranslate(t, 'reports.fiefVacated', { capital, owner })
    case 'fief_dissolved':
      return richTranslate(t, 'reports.fiefDissolved', {
        capital,
        owner,
        reason: reportReason(fief.reason, t) ?? fief.reason ?? '—',
      })
    case 'fief_title_settled':
      return richTranslate(
        t,
        fief.reason === 'exile_replaced'
          ? 'reports.fiefExileReplaced'
          : 'reports.fiefLegitimised',
        { capital, noble },
      )
    case 'title_renounced':
      if (fief.reason === 'claim') {
        return richTranslate(t, 'reports.claimRenounced', {
          noble,
          target: fief.targetName ?? '—',
        })
      }
      return richTranslate(
        t,
        fief.reason === 'usurped'
          ? 'reports.titleRenouncedUsurped'
          : 'reports.titleRenouncedExiled',
        { capital, noble },
      )
    case 'fief_member_occupied':
      if (fief.reason === 'capital') {
        return richTranslate(t, 'reports.fiefCapitalOccupied', {
          capital,
          occupant: playerLabel(players, fief.occupant, t),
        })
      }
      return richTranslate(t, 'reports.fiefMemberOccupied', {
        territory: territoryLabel(map, fief.territory, t),
        capital,
        occupant: playerLabel(players, fief.occupant, t),
      })
    default:
      return capital
  }
}

function cardEventLabel(card: CardReport, map: MapData | null, t: Translate): ReactNode {
  const label = formatCardLabel(card.kind, t)
  const region = regionLabel(map, card.region, t)
  const player = card.player ?? t('reports.unknownPlayer')
  switch (card.eventType) {
    case 'deck_draw':
      return t('reports.cardDrawn', { card: label, player })
    case 'deck_discard':
      return t('reports.cardDiscarded', { card: label, player })
    case 'deck_order_played':
      if (card.noble) {
        return richTranslate(
          t,
          card.commissioner
            ? 'reports.cardPlayedAssassination'
            : 'reports.cardPlayedOnNoble',
          {
            card: label,
            player,
            noble: nobleLabel(card.noble, card.nobleName),
            commissioner: nobleLabel(card.commissioner, card.commissionerName),
          },
        )
      }
      return card.region
        ? richTranslate(t, 'reports.cardPlayed', { card: label, player, region })
        : t('reports.cardPlayedBare', { card: label, player })
    case 'deck_restore':
      return t('reports.cardRestored', { card: label, player })
    case 'calamity_scheduled':
      return richTranslate(
        t,
        card.ritual ? 'reports.cardScheduledOmens' : 'reports.cardScheduled',
        {
          card: label,
          region,
          season: card.season ? t(SEASON_LABEL_KEYS[card.season]) : '—',
        },
      )
    default:
      return label
  }
}

function seasonEffectLabel(
  effect: SeasonEffectReport,
  map: MapData | null,
  t: Translate,
): ReactNode {
  const region = regionLabel(map, effect.region, t)
  const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
  switch (effect.kind) {
    case 'calamity_canceled':
      return richTranslate(t, 'reports.calamityCanceled', { card, bonus: card, region })
    case 'bonus_effect':
      return richTranslate(t, 'reports.bonusEffect', { card, region })
    default:
      return card || effect.kind
  }
}

interface SeasonEffectLine {
  key: string
  owner?: PlayerId
  label: ReactNode
  /** Territories the effect touches, shown on the map by « Voir ». */
  territories?: string[]
  /** Neutral-army lines render in a muted gray instead of the section red. */
  muted?: boolean
}

interface SeasonEffectGroup {
  key: string
  header?: ReactNode
  /** Territories of the header line, shown with the lines' own by « Voir ». */
  headerTerritories?: string[]
  lines: SeasonEffectLine[]
}

function seasonEffectLine(
  effect: SeasonEffectReport,
  map: MapData | null,
  t: Translate,
  index: number,
): SeasonEffectLine {
  const region = regionLabel(map, effect.region, t)
  const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
  const key = `${effect.kind}-${effect.cardKind ?? ''}-${index}`
  const owner = effect.owner ?? t('reports.unknownPlayer')
  switch (effect.kind) {
    case 'calamity_applied':
      if (effect.army) {
        return {
          key,
          owner: effect.owner,
          label: richTranslate(t, 'reports.calamityPlagueArmy', {
            owner,
            territory: territoryLabel(map, effect.territory, t),
            before: effect.sizeBefore ?? 0,
            after: effect.sizeAfter ?? 0,
          }),
        }
      }
      return { key, label: richTranslate(t, 'reports.calamityApplied', { card, region }) }
    case 'bad_weather_blocked': {
      const base = {
        owner,
        territory: territoryLabel(map, effect.territory, t),
      }
      return {
        key,
        owner: effect.owner,
        label: effect.target
          ? richTranslate(t, 'reports.calamityBadWeatherBlocked', {
              ...base,
              target: territoryLabel(map, effect.target, t),
            })
          : richTranslate(t, 'reports.calamityBadWeatherBlockedNoTarget', base),
      }
    }
    case 'famine_loss':
      if (!effect.territory) {
        return {
          key,
          label: richTranslate(t, 'reports.calamityFamineRegion', {
            region,
            production: effect.productionLost ?? 0,
            rations: effect.rationsLost ?? 0,
          }),
        }
      }
      return {
        key,
        label: richTranslate(t, 'reports.calamityFamineSettlement', {
          territory: territoryLabel(map, effect.territory, t),
          production: effect.productionLost ?? 0,
        }),
      }
    case 'bad_weather_loss':
      return {
        key,
        label: effect.territory
          ? richTranslate(t, 'reports.calamityBadWeatherMill', {
              territory: territoryLabel(map, effect.territory, t),
              production: effect.productionLost ?? 0,
            })
          : richTranslate(t, 'reports.calamityBadWeatherRegion', {
              region,
              production: effect.productionLost ?? 0,
            }),
      }
    case 'plague_noble_death':
      return {
        key,
        label: richTranslate(t, 'reports.plagueDeath', {
          noble: nobleLabel(effect.noble, effect.nobleName),
          territory: territoryLabel(map, effect.territory, t),
        }),
      }
    case 'trial':
      return {
        key,
        label:
          effect.reason === 'trial_executed'
            ? richTranslate(t, 'reports.trialExecuted', {
                noble: nobleLabel(effect.noble, effect.nobleName),
                territory: territoryLabel(map, effect.territory, t),
                dignity: t(`dignity.${effect.dignity ?? 'bastard'}`),
              })
            : richTranslate(t, 'reports.trialUnfounded', {
                noble: nobleLabel(effect.noble, effect.nobleName),
              }),
      }
    case 'ambush':
      return {
        key,
        label:
          effect.reason === 'ambush_capture'
            ? richTranslate(t, 'reports.ambushCapture', {
                noble: nobleLabel(effect.noble, effect.nobleName),
                source: territoryLabel(map, effect.target, t),
                territory: territoryLabel(map, effect.territory, t),
              })
            : richTranslate(t, 'reports.ambushVoid', {
                source: territoryLabel(map, effect.target, t),
                territory: territoryLabel(map, effect.territory, t),
              }),
      }
    case 'assassination':
      return {
        key,
        label: richTranslate(
          t,
          effect.commissioner
            ? 'reports.assassinationCommissioner'
            : 'reports.assassinationPublic',
          {
            noble: nobleLabel(effect.noble, effect.nobleName),
            territory: territoryLabel(map, effect.territory, t),
            commissioner: effect.commissioner ?? '',
          },
        ),
      }
    case 'justice':
      return {
        key,
        label: richTranslate(
          t,
          effect.reason === 'justice_executed'
            ? 'reports.justiceExecuted'
            : 'reports.justiceNoEffect',
          {
            player: effect.owner ?? t('reports.unknownPlayer'),
            noble: nobleLabel(effect.noble, effect.nobleName),
            territory: territoryLabel(map, effect.territory, t),
          },
        ),
      }
    case 'assassination_revealed':
      return {
        key,
        label: richTranslate(t, 'reports.assassinationRevealed', {
          noble: nobleLabel(effect.noble, effect.nobleName),
          commissioner: nobleLabel(effect.commissioner, effect.commissionerName),
        }),
      }
    case 'tunnel':
      return {
        key,
        label:
          effect.reason === 'tunnel_escape'
            ? richTranslate(t, 'reports.tunnelEscape', {
                noble: nobleLabel(effect.noble, effect.nobleName),
                source: territoryLabel(map, effect.target, t),
                territory: territoryLabel(map, effect.territory, t),
              })
            : richTranslate(
                t,
                effect.reason === 'tunnel_bypass'
                  ? 'reports.tunnelBypass'
                  : 'reports.tunnelVoid',
                {
                  source: territoryLabel(map, effect.target, t),
                  territory: territoryLabel(map, effect.territory, t),
                },
              ),
      }
    case 'plague_noble_survived':
      return {
        key,
        label: richTranslate(t, 'reports.plagueSurvived', {
          noble: nobleLabel(effect.noble, effect.nobleName),
          territory: territoryLabel(map, effect.territory, t),
        }),
      }
    case 'excommunication':
    case 'excommunication_lifted':
      return {
        key,
        label: richTranslate(
          t,
          effect.kind === 'excommunication'
            ? 'reports.excommunication'
            : 'reports.excommunicationLifted',
          {
            noble: nobleLabel(effect.noble, effect.nobleName),
          },
        ),
      }
    case 'dignity_revealed':
      return {
        key,
        label: richTranslate(t, 'reports.dignityRevealed', {
          noble: nobleLabel(effect.noble, effect.nobleName),
          dignity: t(`dignity.${effect.dignity ?? 'bastard'}`),
        }),
      }
    case 'eon_unmasked':
      return {
        key,
        label: richTranslate(t, 'reports.eonUnmasked', {
          noble: nobleLabel(effect.noble, effect.nobleName),
        }),
      }
    case 'famine':
      return {
        key,
        muted: true,
        label: richTranslate(t, 'reports.neutralFamine', {
          territory: territoryLabel(map, effect.territory, t),
          before: effect.sizeBefore ?? 0,
          after: effect.sizeAfter ?? 0,
        }),
      }
    case 'card_canceled': {
      const card = effect.cardKind ? formatCardLabel(effect.cardKind, t) : ''
      const target = effect.territory
        ? richTranslate(t, 'reports.cardCanceled', {
            player: effect.owner ?? t('reports.unknownPlayer'),
            card,
            territory: territoryLabel(map, effect.territory, t),
          })
        : card
      return { key, owner: effect.owner, label: target }
    }
    case 'neutral_army_created': {
      const count = effect.troops ?? 0
      return {
        key,
        muted: true,
        label: richTranslate(
          t,
          count === 1 ? 'reports.neutralArmyCreatedTroop' : 'reports.neutralArmyCreated',
          { count, territory: territoryLabel(map, effect.territory, t) },
        ),
      }
    }
    case 'army_revolt':
      return {
        key,
        owner: effect.victimOwner,
        label: richTranslate(
          t,
          effect.reason === 'neutralized'
            ? 'reports.armyRevolt.neutralized'
            : 'reports.armyRevolt.suppressed',
          {
            owner: effect.victimOwner ?? t('reports.unknownPlayer'),
            territory: territoryLabel(map, effect.territory, t),
            size: effect.sizeBefore ?? 0,
            total: effect.troops ?? 0,
          },
        ),
      }
    case 'revolt_appeased':
      return {
        key,
        owner: effect.owner,
        label: richTranslate(t, revoltAppeasedKey(effect.reason), {
          noble: nobleLabel(effect.noble, effect.nobleName),
          territory: territoryLabel(map, effect.territory, t),
          cost: effect.cost ?? 0,
        }),
      }
    default:
      return { key, label: seasonEffectLabel(effect, map, t) }
  }
}

function revoltAppeasedKey(reason: string | undefined) {
  switch (reason) {
    case 'paid':
      return 'reports.revoltAppeased.paid'
    case 'succeeded':
      return 'reports.revoltAppeased.succeeded'
    case 'failed_death':
      return 'reports.revoltAppeased.failed_death'
    default:
      return 'reports.revoltAppeased.failed'
  }
}

function groupSeasonEffects(
  effects: SeasonEffectReport[],
  map: MapData | null,
  t: Translate,
): { flat: SeasonEffectLine[]; groups: SeasonEffectGroup[] } {
  const flat: SeasonEffectLine[] = []
  const groups = new Map<string, SeasonEffectGroup>()
  const groupFor = (effect: SeasonEffectReport): SeasonEffectGroup => {
    const key = `${effect.cardKind ?? ''}-${effect.region ?? ''}`
    let group = groups.get(key)
    if (!group) {
      group = { key, lines: [] }
      groups.set(key, group)
    }
    return group
  }
  effects.forEach((effect, index) => {
    const line = {
      ...seasonEffectLine(effect, map, t, index),
      // A region-level effect lights its whole region; a line about one
      // territory keeps to it (and the seat that names its region).
      territories: [
        effect.territory,
        effect.target,
        ...(effect.region
          ? effect.territory
            ? [effect.region]
            : regionTerritories(map, effect.region)
          : []),
      ].filter((id): id is string => Boolean(id)),
    }
    switch (effect.kind) {
      case 'calamity_applied':
        if (effect.army) {
          groupFor(effect).lines.push(line)
        } else {
          const group = groupFor(effect)
          group.header = line.label
          group.headerTerritories = line.territories
        }
        break
      case 'bad_weather_blocked':
      case 'famine_loss':
      case 'bad_weather_loss':
      case 'plague_noble_death':
      case 'plague_noble_survived':
      case 'trial':
      case 'tunnel':
      case 'ambush':
      case 'assassination':
      case 'justice':
      case 'assassination_revealed':
      case 'eon_unmasked':
      case 'dignity_revealed':
      case 'excommunication':
      case 'excommunication_lifted':
      case 'card_canceled':
        groupFor(effect).lines.push(line)
        break
      case 'famine':
      case 'neutral_army_created':
      case 'army_revolt':
        flat.push(line)
        break
      default:
        flat.push(line)
    }
  })
  return { flat, groups: Array.from(groups.values()) }
}

export function ReportPanel({ report, map, players, viewerId }: ReportPanelProps) {
  const { t } = useLanguage()
  if (!report) return null
  const income = report.income ?? []
  const mills = report.mills ?? []
  const production = report.production ?? []
  const consumption = report.consumption ?? []
  const winterInvestments = report.winter?.investments ?? []
  const winterStocks = report.winter?.stocks ?? []
  const rumors = report.rumors ?? report.winter?.rumors ?? []
  const cards = report.cards ?? report.winter?.cards ?? []
  const seasonEffects = report.seasonEffects ?? []
  const seasonEffectView = groupSeasonEffects(seasonEffects, map, t)
  const fiefs = report.fiefs ?? []
  const marriages = report.marriages ?? []
  const elections = report.elections ?? []

  // Repetitive results fold into one summary card each; lines without
  // effect (nothing credited, a stock unchanged) are left out.
  const incomeLines = income.filter(
    (line) => line.lost || line.credited > 0 || (line.suppressed ?? 0) > 0,
  )
  const millLines = mills.filter(
    (line) => line.production > 0 || (line.suppressed ?? 0) > 0,
  )
  const famines = consumption.filter((line) => line.famine)
  const stockLines = winterStocks.filter(isHalved)
  // Reports stored before the engine listed movements are rebuilt from
  // their combats and orders (temporary).
  const movements = report.movements ?? legacyMovements(report)
  const houseLogs = groupHouseLogs(report, players, viewerId)

  const hasEvents =
    seasonEffects.length > 0 ||
    famines.length > 0 ||
    fiefs.length > 0 ||
    marriages.length > 0 ||
    elections.length > 0
  const showEconomy =
    incomeLines.length > 0 || millLines.length > 0 || consumption.length > 0
  const hasEconomy = showEconomy
  const hasWinter = winterInvestments.length > 0 || stockLines.length > 0
  const empty =
    movements.length === 0 &&
    houseLogs.length === 0 &&
    !hasWinter &&
    !hasEvents &&
    !hasEconomy &&
    rumors.length === 0 &&
    cards.length === 0

  return (
    <section className="min-w-0 space-y-4">
      <div>
        <p className="text-2xs font-semibold uppercase tracking-[0.2em] text-brass-700">
          {t('reports.title', { turn: report.header.turn })}
        </p>
        <h3 className="font-display text-xl font-semibold">
          {t('reports.resolutionComplete')}
        </h3>
      </div>

      {empty && (
        <p className="text-sm italic text-ink-500">{t('reports.nothingToReport')}</p>
      )}

      {(rumors.length > 0 || cards.length > 0) && (
        <div className="grid gap-3">
          {rumors.length > 0 && (
            <aside
              data-report-rumors
              className="space-y-1.5 rounded-md border border-line bg-paper-100 p-3"
            >
              <h4 className="text-sm font-semibold italic text-ink-900">
                {t('reports.rumorsMarket')}
              </h4>
              <ul className="space-y-1 text-sm italic text-ink-700">
                {rumors.map((rumor, index) => (
                  <li key={`${rumor.key}-${index}`}>{t(rumor.key as MessageKey)}</li>
                ))}
              </ul>
            </aside>
          )}
          {cards.length > 0 && (
            <aside
              data-report-cards
              className="space-y-1.5 rounded-md border border-line bg-paper-50 p-3"
            >
              <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-ink-700">
                {t('reports.cards')}
              </h4>
              <ol className="space-y-1 text-sm text-ink-700">
                {cards.map((card, index) => (
                  <li key={`${card.eventType}-${card.kind}-${index}`}>
                    {cardEventLabel(card, map, t)}
                  </li>
                ))}
              </ol>
            </aside>
          )}
        </div>
      )}

      {hasEvents && (
        <EventCategory
          id="events"
          title={t('reports.category.events')}
          icon={<IconBell />}
        >
          {seasonEffectView.flat.map((line) => (
            <EventCard
              key={line.key}
              icon={<IconCloudStorm />}
              tone={line.muted ? 'muted' : 'alert'}
              title={line.label}
              owner={line.owner ? playerMarker(players, line.owner, t) : undefined}
              marks={
                line.territories?.length ? { territories: line.territories } : undefined
              }
            />
          ))}
          {seasonEffectView.groups.map((group) => (
            <EventCard
              key={group.key}
              icon={<IconCloudStorm />}
              tone="alert"
              title={group.header ?? group.lines[0]?.label}
              owner={
                !group.header && group.lines[0]?.owner
                  ? playerMarker(players, group.lines[0].owner, t)
                  : undefined
              }
              detail={
                group.lines.length > (group.header ? 0 : 1) ? (
                  <ul className="space-y-0.5">
                    {group.lines.slice(group.header ? 0 : 1).map((line) => (
                      <li
                        key={line.key}
                        className={`flex items-center gap-2 ${line.muted ? 'text-ink-500' : ''}`}
                      >
                        {line.owner ? playerMarker(players, line.owner, t) : null}
                        <span>{line.label}</span>
                      </li>
                    ))}
                  </ul>
                ) : undefined
              }
              marks={(() => {
                const territories = [
                  ...new Set([
                    ...(group.headerTerritories ?? []),
                    ...group.lines.flatMap((line) => line.territories ?? []),
                  ]),
                ]
                return territories.length > 0 ? { territories } : undefined
              })()}
            />
          ))}
          {famines.map((line) => (
            <EventCard
              key={line.army}
              icon={<IconSkull />}
              tone="alert"
              owner={playerMarker(players, line.owner, t)}
              title={
                <>
                  {line.owner} · {territoryLabel(map, line.territory, t)}
                </>
              }
              detail={
                <>
                  <span className="block">
                    {line.source
                      ? `${t('reports.consumptionSource', { source: line.source })} · `
                      : ''}
                    {t('reports.consumptionLine', {
                      local: line.receivedLocal,
                      transfer: line.receivedTransfer,
                      total: line.totalReceived,
                      missing: line.missing,
                    })}
                  </span>
                  {line.warned ? (
                    <span className="block font-semibold">
                      {t('reports.famineWarning')}
                    </span>
                  ) : (
                    <span className="block font-semibold">
                      {line.savedByPillage ? t('reports.savedByPillage') : ''}
                      {(line.troopsLost ?? 0) > 0
                        ? t(
                            line.troopsLost === 1
                              ? 'reports.lostTroop'
                              : 'reports.lostTroops',
                            {
                              count: line.troopsLost ?? 0,
                            },
                          )
                        : ''}
                      {line.destroyed ? t('reports.famineDestroyed') : ''}
                      {(line.resourceCredit ?? 0) > 0 && (
                        <>
                          {' '}
                          {richTranslate(t, 'reports.consumptionPillageCredit', {
                            count: line.resourceCredit ?? 0,
                            territory: territoryLabel(map, line.creditTerritory, t),
                          })}
                        </>
                      )}
                    </span>
                  )}
                </>
              }
              marks={{
                territories: line.source
                  ? [line.source, line.territory]
                  : [line.territory],
                // The supply flows from its source to the army it feeds.
                arrows:
                  line.source && line.source !== line.territory
                    ? [[line.source, line.territory]]
                    : undefined,
              }}
            />
          ))}
          {fiefs.map((fief, index) => (
            <EventCard
              key={`${fief.kind}-${fief.capital}-${index}`}
              icon={<IconCrown />}
              tone={fief.kind === 'fief_member_occupied' ? 'alert' : undefined}
              title={fiefEventLabel(fief, players, map, t)}
              marks={{
                territories: [fief.capital, fief.territory].filter((id): id is string =>
                  Boolean(id),
                ),
              }}
            />
          ))}
          {marriages.map((marriage, index) => (
            <EventCard
              key={`${marriage.outcome}-${marriage.noble}-${marriage.spouse}-${index}`}
              icon={<IconHeart />}
              title={marriageLabel(marriage, players, t)}
            />
          ))}
          {elections.map((election, index) => {
            const winner = election.candidates.find(
              (candidate) => candidate.noble === election.winner,
            )
            const seat =
              election.kind === 'bishop' ? territoryLabel(map, election.seat, t) : ''
            return (
              <EventCard
                key={`${election.kind}-${election.region ?? ''}-${index}`}
                icon={<IconBuildingChurch />}
                title={
                  <>
                    {election.kind === 'bishop'
                      ? richTranslate(t, 'reports.electionBishop', { seat })
                      : election.kind === 'king'
                        ? t('reports.electionKing')
                        : t('reports.electionPope')}
                    {' — '}
                    {richTranslate(t, `reports.electionResult.${election.result}`, {
                      noble: winner
                        ? nobleLabel(winner.nobleCode, winner.nobleName)
                        : '—',
                    })}
                  </>
                }
                detail={
                  election.candidates.length > 0
                    ? `${election.candidates
                        .map(
                          (candidate) =>
                            `${candidate.nobleName} (${candidate.owner}) : ${candidate.votes}${
                              candidate.anointed
                                ? ` ${t('reports.electionAnointed')}`
                                : ''
                            }`,
                        )
                        .join(' · ')}${
                        election.required || election.minVotes
                          ? ` — ${t('reports.electionRequired', {
                              count: election.required ?? election.minVotes ?? 0,
                            })}`
                          : ''
                      }`
                    : undefined
                }
                marks={election.seat ? { territories: [election.seat] } : undefined}
              />
            )
          })}
        </EventCategory>
      )}

      {movements.length > 0 && (
        <EventCategory
          id="movements"
          title={t('reports.movements')}
          icon={<IconSwords />}
        >
          {(['victory', 'move', 'failure'] as const).map((kind) => {
            const group = movements.filter((movement) => movement.kind === kind)
            if (group.length === 0) return null
            return (
              <section key={kind} data-movement-group={kind} className="space-y-1.5">
                <h5 className={SUBHEADING}>{t(`reports.movements.${kind}`)}</h5>
                {group.map((movement, index) => (
                  <MovementCard
                    key={`${movement.army ?? movement.combat ?? movement.legs[0]?.to}-${index}`}
                    movement={movement}
                    report={report}
                    map={map}
                    players={players}
                  />
                ))}
              </section>
            )
          })}
        </EventCategory>
      )}

      {hasWinter && (
        <EventCategory
          id="winter"
          title={t(
            report.header.season === 'winter' ? 'reports.winter' : 'reports.nobleCards',
          )}
          icon={<IconSnowflake />}
        >
          <WinterHouses
            investments={winterInvestments}
            stocks={winterStocks}
            map={map}
            players={players}
            viewerId={viewerId}
          />
        </EventCategory>
      )}

      {hasEconomy && (
        <EventCategory
          id="economy"
          title={t('reports.category.economy')}
          icon={<IconCoins />}
        >
          {showEconomy && (
            <EconomyCard
              stocks={report.players}
              income={incomeLines}
              mills={millLines}
              production={production}
              supply={consumption}
              map={map}
              players={players}
              viewerId={viewerId}
            />
          )}
        </EventCategory>
      )}

      {houseLogs.length > 0 && (
        <EventCategory
          id="houses"
          title={t('reports.ordersExecuted')}
          icon={<IconScript />}
        >
          {houseLogs.map((house) => (
            <HouseLog
              key={house.player}
              house={house}
              open={house.player === viewerId || houseLogs.length === 1}
              map={map}
              players={players}
            />
          ))}
        </EventCategory>
      )}
    </section>
  )
}

const SUBHEADING = 'text-2xs font-semibold uppercase tracking-wide text-ink-500'

/** Orders and receptions of one player, for the closing block of the report. */
interface HouseLogData {
  player: PlayerId
  receptions: ReceptionReport[]
  orders: OrderReport[]
}

/**
 * The receptions of every house, and the orders of the viewer's only: another
 * house's orders are not the viewer's to read (a spectator or a hotseat
 * screen, with no viewer, reads them all).
 */
function groupHouseLogs(
  report: TurnReport,
  players: Player[],
  viewerId?: PlayerId,
): HouseLogData[] {
  const houses = new Map<PlayerId, HouseLogData>()
  const house = (player: PlayerId) => {
    let entry = houses.get(player)
    if (!entry) {
      entry = { player, receptions: [], orders: [] }
      houses.set(player, entry)
    }
    return entry
  }
  for (const reception of report.receptions ?? [])
    house(reception.player).receptions.push(reception)
  for (const order of report.orders ?? []) {
    if (order.visibility === 'hidden' || !order.owner) continue
    if (viewerId && order.owner !== viewerId) continue
    house(order.owner).orders.push(order)
  }
  const rank = (player: PlayerId) => houseRank(player, players, viewerId)
  return [...houses.values()].sort((a, b) => rank(a.player) - rank(b.player))
}

/** Territory names, each linked, joined by commas. */
function territoryList(ids: string[], map: MapData | null, t: Translate): ReactNode {
  return joinNodes(
    [...new Set(ids)].map((id) => (
      <Fragment key={id}>{territoryLabel(map, id, t)}</Fragment>
    )),
    ', ',
  )
}

/** What an order does, in a sentence. */
function orderMeaning(order: OrderReport, map: MapData | null, t: Translate): ReactNode {
  const targets = orderTargets(order)
  const source = territoryLabel(map, order.source, t)
  const at = (index: number) => territoryLabel(map, targets[index], t)
  const list = () =>
    joinNodes(
      targets.map((target, index) => {
        const count = order.counts?.[index]
        const suffix =
          count === MANEUVER_REST
            ? ` (${t('reports.orderMeaning.maneuverRest')})`
            : count !== undefined && count !== 1
              ? ` (${count})`
              : ''
        return (
          <Fragment key={target}>
            {territoryLabel(map, target, t)}
            {suffix}
          </Fragment>
        )
      }),
      ', ',
    )
  switch (order.type) {
    case 'hold':
      return richTranslate(t, 'reports.orderMeaning.hold', { source })
    case 'attack':
      return richTranslate(t, 'reports.orderMeaning.attack', { source, target: at(0) })
    case 'support':
      return targets.length > 1
        ? richTranslate(t, 'reports.orderMeaning.supportAttack', {
            source,
            supported: at(0),
            target: at(1),
          })
        : richTranslate(t, 'reports.orderMeaning.supportDefense', {
            source,
            supported: at(0),
          })
    case 'join':
      return richTranslate(t, 'reports.orderMeaning.join', { source, target: at(0) })
    case 'pillage':
      return richTranslate(t, 'reports.orderMeaning.pillage', { source })
    case 'disperse':
      return richTranslate(t, 'reports.orderMeaning.disperse', {
        source,
        targets: list(),
      })
    case 'maneuver':
      return richTranslate(t, 'reports.orderMeaning.maneuver', {
        source,
        targets: list(),
      })
    case 'transfer':
      return richTranslate(t, 'reports.orderMeaning.transfer', {
        source,
        target: at(0),
        amount: order.amount ?? '—',
      })
    default:
      return formatReportOrderLabel(order, map, t)
  }
}

/** A house's closing block: its chain receptions, then the orders it gave. */
function HouseLog({
  house,
  open,
  map,
  players,
}: {
  house: HouseLogData
  open: boolean
  map: MapData | null
  players: Player[]
}) {
  const { t } = useLanguage()
  const { arms, name } = houseIdentity('player', house.player, players, t)
  const received = house.receptions.filter((reception) => reception.received).length
  return (
    <HouseBlock
      shieldId={`log-${house.player}`}
      open={open}
      arms={arms}
      name={name}
      balance={
        house.receptions.length > 0
          ? t('reports.summary.receptions', {
              received,
              total: house.receptions.length,
            })
          : ''
      }
      inner="space-y-2 px-2 py-1.5"
    >
      {house.receptions.length > 0 && (
        <section data-house-receptions className="space-y-1">
          <h5 className={SUBHEADING}>{t('reports.receptionsTitle')}</h5>
          <ul className="space-y-1 text-sm">
            {house.receptions.map((reception, index) => (
              <li
                key={`${reception.noble}-${index}`}
                className="flex items-center justify-between gap-3"
              >
                <span className="min-w-0">
                  {nobleLabel(reception.noble, reception.nobleName)}
                </span>
                <span
                  className={
                    reception.received
                      ? 'shrink-0 text-success-700'
                      : 'shrink-0 text-brass-700'
                  }
                >
                  {reception.received
                    ? t('reports.receptionReceived')
                    : reception.reason
                      ? richTranslate(t, 'reports.receptionLost', {
                          reason:
                            reportReason(
                              reception.reason,
                              t,
                              reception.reasonKey,
                              reception.reasonArgs,
                              map,
                            ) ?? reception.reason,
                        })
                      : t('reports.receptionLostPlain')}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}
      {house.orders.length > 0 && (
        <section data-house-orders className="space-y-1">
          <h5 className={SUBHEADING}>{t('reports.ordersExecuted')}</h5>
          {house.orders.map((order, index) => {
            const targets = orderTargets(order)
            const territories = [order.source, ...targets].filter((id): id is string =>
              Boolean(id),
            )
            return (
              <EventCard
                key={`${order.chain ?? 'order'}-${order.order ?? index}-${index}`}
                icon={<IconScript />}
                dense
                title={orderMeaning(order, map, t)}
                detail={
                  <>
                    {richTranslate(t, 'reports.noble', {
                      noble: nobleLabel(order.noble, order.nobleName),
                    })}
                    {' · '}
                    <span className="font-mono">
                      {formatReportOrderLabel(order, map, t)}
                    </span>
                  </>
                }
                aside={
                  <span className={outcomeClass(order.outcome)}>
                    {outcomeLabel(order.outcome, t)}
                    {order.reason ? (
                      <span className="block max-w-36 text-2xs leading-tight">
                        {reportReason(order.reason, t)}
                      </span>
                    ) : null}
                  </span>
                }
                marks={
                  territories.length > 0
                    ? {
                        territories: [...new Set(territories)],
                        arrows: orderArrows(order.type, order.source, targets),
                      }
                    : undefined
                }
              />
            )
          })}
        </section>
      )}
    </HouseBlock>
  )
}

/** Where each support that bears on a fight stands, drawn toward it. */
function supporterLegs(
  report: TurnReport,
  combat: CombatReport,
): Array<[string, string]> {
  const armies = new Set([...(combat.supporters ?? []), ...(combat.cutSupporters ?? [])])
  const legs: Array<[string, string]> = []
  for (const army of armies) {
    const from = armyOrigin(report, army)
    if (from && from !== combat.territory) legs.push([from, combat.territory])
  }
  return legs
}

/**
 * One army's move: what it did in a sentence, where it came from, and, for a
 * real fight, the forces behind the result in a fold-out.
 */
function MovementCard({
  movement,
  report,
  map,
  players,
}: {
  movement: ArmyMovement
  report: TurnReport
  map: MapData | null
  players: Player[]
}) {
  const { t } = useLanguage()
  const exact = movement.combat
    ? report.combats?.find(
        (combat) =>
          combat.territory === movement.combat && combat.visibility !== 'general',
      )
    : undefined
  const destinations = [...new Set(movement.legs.map((leg) => leg.to))]
  const origins = [
    ...new Set(
      movement.legs.map((leg) => leg.from).filter((id): id is string => Boolean(id)),
    ),
  ]
  const to = territoryList(destinations, map, t)
  const owner = movement.owner ?? t('reports.unknownPlayer')
  const title = movement.army
    ? richTranslate(t, `reports.movement.${movement.kind}`, { owner, to })
    : richTranslate(
        t,
        movement.kind === 'victory'
          ? 'reports.movement.victoryUnknown'
          : 'reports.movement.failureUnknown',
        { to },
      )
  const result: ReactNode = exact
    ? exact.winner
      ? richTranslate(t, 'reports.victory', {
          army: armyDescription(report, map, exact.winner, t),
        })
      : exact.standoff
        ? t('reports.standoff')
        : (reportReason(exact.reason, t) ?? exact.reason)
    : undefined
  const why =
    movement.kind === 'failure'
      ? (reportReason(movement.reason, t) ?? movement.reason)
      : undefined
  const detail = [
    origins.length > 0
      ? richTranslate(t, 'reports.movement.from', {
          from: territoryList(origins, map, t),
        })
      : null,
    why,
  ].filter(Boolean)
  // The supports behind a fight are drawn with it, each from where it stands.
  const supports = exact ? supporterLegs(report, exact) : []
  const arrows = movement.legs
    .filter(
      (leg): leg is { from: string; to: string } =>
        Boolean(leg.from) && leg.from !== leg.to,
    )
    .map((leg): [string, string] => [leg.from, leg.to])
  arrows.push(
    ...supports.filter(([from, to]) => !arrows.some(([a, b]) => a === from && b === to)),
  )
  return (
    <EventCard
      icon={<IconSwords />}
      tone={movement.kind === 'failure' ? 'muted' : 'plain'}
      owner={movement.owner ? playerMarker(players, movement.owner, t) : undefined}
      title={title}
      detail={detail.length > 0 ? joinNodes(detail, ' · ') : undefined}
      marks={{
        territories: [
          ...new Set([...origins, ...destinations, ...supports.map(([from]) => from)]),
        ],
        arrows: arrows.length > 0 ? arrows : undefined,
      }}
      more={exact ? combatDetails(report, map, exact, result, t) : undefined}
    />
  )
}

/**
 * The moves an order draws on the map. A support points at the destination it
 * backs, and the supported army's own move is drawn too; it never points at
 * the supported army itself.
 */
function orderArrows(
  type: TurnReport['orders'][number]['type'],
  source: string | undefined,
  targets: string[],
): Array<[string, string]> | undefined {
  if (!source || !type || type === 'hold' || type === 'pillage') return undefined
  if (type === 'support') {
    const [supported, destination] = targets
    if (!supported) return undefined
    return destination
      ? [
          [supported, destination],
          [source, destination],
        ]
      : [[source, supported]]
  }
  return targets.map((target): [string, string] => [source, target])
}

/** Territory an army set out from this turn, read from its order. */
function armyOrigin(report: TurnReport, army: string | undefined): string | undefined {
  // A castle-only defense has no army; hidden orders have none either.
  if (!army) return undefined
  return report.orders?.find((order) => order.army === army && order.source)?.source
}

/**
 * Which side a supporter backs, read from its order: `S supported dest`
 * backs the attack into dest, `S territory` backs the defense of territory.
 */
function supportSide(
  report: TurnReport,
  army: string,
  combat: CombatReport,
): 'attack' | 'defense' | undefined {
  const targets = report.orders.find((order) => order.army === army)?.targets ?? []
  if (targets.length > 1 && targets[1] === combat.territory) return 'attack'
  if (targets.length === 1 && targets[0] === combat.territory) return 'defense'
  return undefined
}

/** The fold-out of a combat card: each side's forces, their supports, the result. */
function combatDetails(
  report: TurnReport,
  map: MapData | null,
  combat: CombatReport,
  result: ReactNode,
  t: Translate,
): ReactNode {
  const contenders = combat.contenders ?? []
  const origin = (army: string | undefined) => {
    const from = armyOrigin(report, army)
    return from && from !== combat.territory ? (
      <> · {territoryLabel(map, from, t)}</>
    ) : null
  }
  // The owner of an army, from the combat itself or else the report; a castle
  // defending alone has no army and no owner.
  const owner = (army: string | undefined, known?: string): string | undefined =>
    known ||
    (army
      ? (report.players
          .flatMap((player) => player.armies)
          .find((entry) => entry.id === army)?.owner ??
        report.orders.find((order) => order.army === army)?.owner)
      : undefined)
  const cut = new Set(combat.cutSupporters ?? [])
  // A cut support still applies, so the engine lists it in both lists.
  const supporters = [...new Set([...(combat.supporters ?? []), ...cut])]
  const bySide = (side: 'attack' | 'defense' | undefined) =>
    supporters.filter((army) => supportSide(report, army, combat) === side)
  const supportList = (title: string, armies: string[]) =>
    armies.length > 0 && (
      <section>
        <h5 className="font-semibold">{title}</h5>
        <ul className="list-disc pl-4">
          {armies.map((army) => (
            <li key={army}>
              {owner(army) ?? t('reports.unknownPlayer')}
              {origin(army)}
              {cut.has(army) && t('reports.combat.cut')}
            </li>
          ))}
        </ul>
      </section>
    )
  const attackers = contenders.filter((contender) => !contender.defender)
  const defenders = contenders.filter((contender) => contender.defender)
  const force = (contender: (typeof contenders)[number]) =>
    [
      t(contender.supported ? 'reports.combat.forceSupported' : 'reports.combat.force', {
        force: contender.force ?? '—',
        base: (contender.force ?? 0) - (contender.supported ?? 0),
        support: contender.supported ?? 0,
      }) +
        (contender.nobleBonus
          ? t('reports.nobleBonus', { bonus: contender.nobleBonus })
          : '') +
        (contender.tunnel ? t('reports.tunnelAttack') : ''),
      contender.defender && combat.castleBonus
        ? t('reports.combat.castleBonus', { bonus: combat.castleBonus })
        : '',
      contender.tunnelForce !== undefined
        ? t('reports.tunnelDefense', { force: contender.tunnelForce }).trim()
        : '',
    ]
      .filter(Boolean)
      .join(' · ')
  return (
    <div className="space-y-2">
      {attackers.length > 0 && (
        <section>
          <h5 className="font-semibold">{t('reports.combat.attackers')}</h5>
          <ul className="list-disc pl-4">
            {attackers.map((contender, index) => (
              <li key={contender.army ?? index}>
                {owner(contender.army, contender.owner) ?? t('reports.unknownPlayer')}
                {origin(contender.army)} · {force(contender)}
              </li>
            ))}
          </ul>
        </section>
      )}
      {defenders.length > 0 && (
        <section>
          <h5 className="font-semibold">{t('reports.combat.defenders')}</h5>
          <ul className="list-disc pl-4">
            {defenders.map((contender, index) => (
              <li key={contender.army ?? index}>
                {owner(contender.army, contender.owner)
                  ? `${owner(contender.army, contender.owner)} · `
                  : ''}
                {force(contender)}
              </li>
            ))}
          </ul>
        </section>
      )}
      {supportList(t('reports.combat.supportAttack'), bySide('attack'))}
      {supportList(t('reports.combat.supportDefense'), bySide('defense'))}
      {supportList(t('reports.combat.supportOther'), bySide(undefined))}
      <section>
        <h5 className="font-semibold">{t('reports.combat.result')}</h5>
        <p>
          {result}
          {combat.dislodged && (
            <>
              {' · '}
              {richTranslate(t, 'reports.combat.dislodged', {
                army: armyDescription(report, map, combat.dislodged, t),
              })}
            </>
          )}
        </p>
      </section>
    </div>
  )
}

/** A category of the report; the caller leaves out an empty one. */
function EventCategory({
  id,
  title,
  icon,
  children,
}: {
  id: string
  title: ReactNode
  icon: ReactNode
  children: ReactNode
}) {
  return (
    <section data-report-category={id} className="space-y-1.5">
      <h4 className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-[0.16em] text-ink-500 [&_svg]:size-3.5">
        {icon}
        {title}
      </h4>
      <div className="space-y-1.5">{children}</div>
    </section>
  )
}

const CARD_TONES = {
  plain: 'border-line bg-paper-50',
  alert: 'border-vermilion-700/30 bg-vermilion-100 text-vermilion-700',
  warning: 'border-warning-700/40 bg-warning-100',
  muted: 'border-line-soft bg-paper-100 text-ink-500',
} as const

/**
 * One event of the report as a card (charte-graphique.md § Rapport de tour):
 * ink pictogram, title, detail; « Voir » frames the map on its territories
 * and draws it there.
 */
function EventCard({
  icon,
  title,
  detail,
  aside,
  owner,
  tone = 'plain',
  marks,
  more,
  dense = false,
}: {
  icon: ReactNode
  title: ReactNode
  detail?: ReactNode
  aside?: ReactNode
  /** The player's colour, before the title. */
  owner?: ReactNode
  tone?: keyof typeof CARD_TONES
  marks?: EventMarks
  /** Folded under the card: a click on the card, or its chevron, unfolds it. */
  more?: ReactNode
  /** A tighter card, for the long lists inside a block. */
  dense?: boolean
}) {
  const { t } = useLanguage()
  const links = useMapLinks()
  const titleId = useId()
  const moreId = useId()
  const [open, setOpen] = useState(false)
  // A touch screen has no hover: it emulates one on a tap and never takes it
  // back, which would leave the map lit.
  const canHover = useMediaQuery('(hover: hover)')
  const light = useMapHighlight()
  const lit = marks?.territories.length ? marks : null
  // A name inside the card lights its own territory: the card leaves it be.
  const onName = (event: SyntheticEvent) =>
    event.target instanceof Element && event.target.closest('[data-map-link]') !== null
  const lightAll = (event: SyntheticEvent) => {
    if (lit && !onName(event)) light.on(lit)
  }
  const lightOff = () => {
    if (lit) light.off()
  }
  return (
    <article
      data-event-card
      // Hovering or focusing the card lights all its territories; a name
      // inside it lights its own, and the card's set comes back when the
      // pointer leaves that name.
      onMouseOver={canHover ? lightAll : undefined}
      onFocus={lightAll}
      onMouseLeave={lightOff}
      onBlur={lightOff}
      // A click on the card's own text unfolds it; links, buttons, the
      // unfolded detail and a text selection keep their own behaviour.
      onClick={
        more
          ? (event) => {
              if (
                event.target instanceof Element &&
                event.target.closest('button, a, summary, [data-event-more]') === null &&
                !window.getSelection()?.toString()
              ) {
                setOpen((value) => !value)
              }
            }
          : undefined
      }
      className={`flex items-start rounded-md border text-sm ${dense ? 'gap-1.5 px-2 py-1' : 'gap-2 px-3 py-2'} ${CARD_TONES[tone]} ${more ? 'cursor-pointer' : ''}`}
    >
      <span
        aria-hidden="true"
        className="mt-0.5 flex size-4 shrink-0 items-center justify-center [&_svg]:size-4"
      >
        {icon}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-start gap-2">
          {owner && <span className="mt-1.5">{owner}</span>}
          <div id={titleId} className="min-w-0 font-medium">
            {title}
          </div>
        </div>
        {detail && <div className="mt-0.5 text-xs opacity-80">{detail}</div>}
        {more && open && (
          <div
            id={moreId}
            data-event-more
            className="mt-2 cursor-default border-t border-current/20 pt-2 text-xs"
          >
            {more}
          </div>
        )}
      </div>
      {aside && <div className="shrink-0 text-right text-xs">{aside}</div>}
      {more && (
        <button
          type="button"
          data-event-toggle
          aria-expanded={open}
          aria-controls={open ? moreId : undefined}
          aria-label={t(open ? 'reports.combat.collapse' : 'reports.combat.expand')}
          className="shrink-0 rounded p-0.5 max-lg:min-h-11"
          onClick={() => setOpen((value) => !value)}
        >
          <IconChevronDown
            aria-hidden="true"
            className={`size-4 transition-transform ${open ? 'rotate-180' : ''}`}
          />
        </button>
      )}
      {marks && links && lit && (
        <button
          type="button"
          data-show-event
          aria-describedby={titleId}
          className="shrink-0 rounded px-1 text-xs font-semibold text-brass-700 underline decoration-dotted underline-offset-2 hover:decoration-solid max-lg:min-h-11"
          onClick={() => links.showEvent(marks)}
        >
          {t('reports.show')}
        </button>
      )}
    </article>
  )
}

/**
 * « 12 fed · 2 starving · 1 with losses », with a leading separator, empty
 * when there is no army: every army counts once, a loss outranking famine
 * (a deficit, warned or not), famine outranking fed.
 */
function armyTally(lines: ConsumptionReport[], t: Translate): string {
  const lost = lines.filter((line) => (line.troopsLost ?? 0) > 0 || line.destroyed)
  const starving = lines.filter((line) => !lost.includes(line) && line.famine)
  const fed = lines.length - lost.length - starving.length
  const parts = [
    fed > 0 && t(fed === 1 ? 'reports.tallyFedOne' : 'reports.tallyFed', { count: fed }),
    starving.length > 0 && t('reports.tallyFamine', { count: starving.length }),
    lost.length > 0 && t('reports.tallyLosses', { count: lost.length }),
  ].filter(Boolean)
  return parts.length > 0 ? ` · ${parts.join(' · ')}` : ''
}

const NEUTRAL_ARMS = { field: NEUTRAL_ARMY_COLOR, charge: null }

/**
 * The economy of the turn, one block per house (the viewer's open, the
 * others folded on their balance). Income comes in one line per fief and one
 * per region for the territories held outside any fief, mills counted on the
 * line they feed; supply goes out per army, grouped by region, the terrain's
 * rations taken off the demand on the army's own line.
 */
function EconomyCard({
  stocks,
  income,
  mills,
  production,
  supply,
  map,
  players,
  viewerId,
}: {
  /** The stock of each player before and after the turn. */
  stocks: PlayerReport[]
  income: IncomeReport[]
  mills: MillReport[]
  production: ProductionReport[]
  supply: ConsumptionReport[]
  map: MapData | null
  players: Player[]
  viewerId?: PlayerId
}) {
  const { t } = useLanguage()
  const sum = (values: number[]) => values.reduce((total, value) => total + value, 0)
  const houseOf = (owner?: PlayerId) => owner ?? NEUTRAL_PLAYER_ID
  const territoryById = new Map((map?.territories ?? []).map((item) => [item.id, item]))
  // The rations the terrain of each territory gives, a discount on the demand.
  const rationsAt = new Map(
    production.map((line) => [
      line.territory,
      line.terrainRations + (line.bonusRations ?? 0),
    ]),
  )
  const rank = (key: string) => houseRank(key, players, viewerId)
  const houses = Array.from(
    new Set([...income, ...mills, ...supply].map((line) => houseOf(line.owner))),
  ).sort((a, b) => rank(a) - rank(b))
  const lineTotal = (line: IncomeReport) => (line.lost ? 0 : line.credited)
  // A fair-weather mill credits its production twice: the bonus is reported apart.
  const millTotal = (mill: MillReport) => mill.production + (mill.bonus ?? 0)
  // What the supply network really distributed from the stocks, famine included.
  const consumedOf = (lines: ConsumptionReport[]) =>
    sum(lines.map((line) => line.receivedTransfer))
  const demandOf = (lines: ConsumptionReport[]) => sum(lines.map((line) => line.demand))
  const cost = (count: number) =>
    count > 0 ? t('reports.economyCost', { count }) : t('reports.economyNoCost')
  const gain = (count: number) => t('reports.economyGain', { count })
  const signed = (net: number) =>
    net >= 0 ? gain(net) : t('reports.economyCost', { count: -net })
  const netClass = (net: number) => (net >= 0 ? 'text-success-700' : 'text-vermilion-700')
  const armyCount = (count: number) =>
    t(count === 1 ? 'reports.supplyArmy' : 'reports.supplyArmies', { count })

  function renderHouse(key: string) {
    const own = (line: { owner?: PlayerId }) => houseOf(line.owner) === key
    // Fiefs first, the most prestigious title leading; Array.sort is stable.
    const titleRank = (line: IncomeReport) => (line.title ? FIEF_RANK[line.title] : -1)
    const houseIncome = income.filter(own).sort((a, b) => titleRank(b) - titleRank(a))
    const houseMills = mills.filter(own)
    const houseSupply = supply.filter(own)
    const { arms, name } = houseIdentity(
      key === NEUTRAL_PLAYER_ID ? 'neutral' : 'player',
      key,
      players,
      t,
    )

    // A mill is counted on the line of its fief, or of its region outside one.
    // A report from before mills carried their fief and region falls back on
    // the destination its output went to.
    const attached = new Map<IncomeReport, MillReport[]>()
    const loose: MillReport[] = []
    houseMills.forEach((mill) => {
      const legacy = mill.fief === undefined && mill.region === undefined
      const feeds = (line: IncomeReport) => {
        if (legacy) return line.destination === mill.destination
        return mill.fief
          ? line.fief === mill.fief
          : !line.fief && line.region === mill.region
      }
      const target = houseIncome.find((line) => !line.source && feeds(line))
      if (!target) return void loose.push(mill)
      attached.set(target, [...(attached.get(target) ?? []), mill])
    })
    const withMills = (line: IncomeReport) =>
      lineTotal(line) + sum((attached.get(line) ?? []).map(millTotal))
    const earned = sum(houseIncome.map(lineTotal)) + sum(houseMills.map(millTotal))
    const net = earned - consumedOf(houseSupply)
    const stock = stocks.find((candidate) => candidate.id === key)

    const nameOf = (id: string) => territoryById.get(id)?.name ?? id
    const byName = (a: string, b: string) => nameOf(a).localeCompare(nameOf(b))
    // One gain of a block: what it is on the left, what it brings in green on the right.
    const gainRow = (
      key: string,
      icon: ReactNode,
      label: ReactNode,
      gain: ReactNode,
      stopped = false,
    ) => (
      <li key={key} className="flex items-center justify-between gap-3 text-xs">
        <span className="flex min-w-0 items-center gap-1.5 text-ink-700">
          <span aria-hidden="true" className="flex shrink-0 [&_svg]:size-3.5">
            {icon}
          </span>
          {label}
        </span>
        <span
          className={`shrink-0 font-semibold ${stopped ? 'text-brass-700' : 'text-success-700'}`}
        >
          {gain}
        </span>
      </li>
    )
    const millRow = (mill: MillReport, index: number) => {
      const stopped = mill.production === 0 && (mill.suppressed ?? 0) > 0
      return gainRow(
        `mill-${mill.territory}-${index}`,
        <IconBuildingWindTurbine />,
        <>
          {territoryLabel(map, mill.territory, t)}{' '}
          <span className="text-ink-500">
            {t('reports.economyMillLevel', { level: mill.level })}
          </span>
        </>,
        stopped ? t('reports.millStopped') : gain(millTotal(mill)),
        stopped,
      )
    }

    const incomeTitle = (line: IncomeReport): ReactNode => {
      if (line.source || line.fief) return incomeDestinationLabel(line, map, t)
      return line.region
        ? richTranslate(t, 'reports.economyOutsideFief', {
            region: regionLabel(map, line.region, t),
          })
        : t('reports.economyOutsideFiefPlain')
    }
    // A line is a balance: the territories, then the villages and the mills,
    // each with its gain; a line with nothing but territories stays compact.
    const renderIncomeBody = (line: IncomeReport) => {
      const villages = [...(line.villageIncome ?? [])].sort((a, b) =>
        byName(a.territory, b.territory),
      )
      const mills = [...(attached.get(line) ?? [])].sort((a, b) =>
        byName(a.territory, b.territory),
      )
      const suppressed = (line.suppressed ?? 0) > 0 && (
        <p className="text-2xs font-semibold text-vermilion-700">
          {t('reports.incomeSuppressed', { count: line.suppressed ?? 0 })}
        </p>
      )
      const destination = line.destination && !line.fief && !line.source && (
        <>
          {' '}
          {richTranslate(t, 'reports.economyTo', {
            destination: territoryLabel(map, line.destination, t),
          })}
        </>
      )
      const territories = t(
        line.territories === 1 ? 'reports.incomeTerritory' : 'reports.incomeTerritories',
        { count: line.territories },
      )
      // A report from before villages were detailed only counts them.
      const legacyVillages = !line.source &&
        line.villages > 0 &&
        villages.length === 0 && (
          <> · {t('reports.incomeVillages', { count: line.villages })}</>
        )
      if (line.source || line.lost || (villages.length === 0 && mills.length === 0)) {
        // Nothing of the territories reaches a stock when the line is lost, but
        // its mills still produce on their own territory.
        return (
          <div className="space-y-1 px-2 py-1.5">
            <p className="text-2xs text-ink-500">
              {line.source
                ? richTranslate(t, `reports.incomeSource.${line.source}`, {
                    fief: territoryLabel(map, line.taxed, t),
                  })
                : territories}
              {destination}
              {legacyVillages}
            </p>
            {mills.length > 0 && <ul className="space-y-0.5">{mills.map(millRow)}</ul>}
            {suppressed}
          </div>
        )
      }
      // A report from before the server sent it: all of the line but its villages.
      const territoryGain =
        line.territoryAmount ??
        line.credited - sum(villages.map((village) => village.amount))
      return (
        <div className="space-y-1 px-2 py-1.5">
          <ul className="space-y-0.5">
            {gainRow(
              'territories',
              <IconMapPin />,
              <>
                {territories}
                {destination}
                {legacyVillages}
              </>,
              gain(territoryGain),
            )}
            {villages.map((village) =>
              gainRow(
                `village-${village.territory}`,
                <IconBuildingCottage />,
                <>
                  {territoryLabel(map, village.territory, t)}{' '}
                  <span className="text-ink-500">
                    {t('reports.economyVillageLevel', { level: village.level })}
                  </span>
                </>,
                gain(village.amount),
              ),
            )}
            {mills.map(millRow)}
          </ul>
          {suppressed}
        </div>
      )
    }
    const renderIncome = (line: IncomeReport, index: number) => {
      const frame = line.fief
        ? 'border-l-ink-500'
        : line.source
          ? 'border-l-vermilion-700'
          : 'border-l-brass-500'
      return (
        <li
          key={`${line.fief ?? line.region ?? line.source ?? 'x'}-${line.destination ?? 'lost'}-${index}`}
          className={`${BLOCK_FRAME} ${frame}`}
        >
          <div className={`${BLOCK_HEADER} text-xs text-ink-900`}>
            <span className="min-w-0">{incomeTitle(line)}</span>
            <span
              className={`shrink-0 ${line.lost ? 'text-brass-700' : 'text-success-700'}`}
            >
              {line.lost
                ? t('reports.incomeLost')
                : t('reports.incomeCredited', { count: withMills(line) })}
            </span>
          </div>
          {renderIncomeBody(line)}
        </li>
      )
    }

    const renderArmy = (line: ConsumptionReport) => {
      const rations = rationsAt.get(line.territory) ?? 0
      const terrain = territoryById.get(line.territory)?.terrain
      const parts: ReactNode[] = [
        t(line.size === 1 ? 'app.troop' : 'app.troops', { count: line.size }),
      ]
      if (line.receivedTransfer > 0) {
        parts.push(
          richTranslate(t, 'reports.economyFromSource', {
            count: line.receivedTransfer,
            source: territoryLabel(map, line.source, t),
          }),
        )
      }
      return (
        <div className="space-y-0.5 text-xs">
          <div className="flex items-center justify-between gap-3">
            <span className="min-w-0">
              {territoryLabel(map, line.territory, t)}
              {terrain && (
                <span className="text-ink-500">
                  {' '}
                  · {t(TERRAIN_LABEL_KEYS[terrain])} (−{rations} R)
                </span>
              )}
            </span>
            <span className="shrink-0 text-right">
              <span className="font-semibold text-vermilion-700">
                {cost(line.receivedTransfer)}
              </span>{' '}
              <span className="text-2xs text-ink-500">
                {t('reports.economyDemand', { count: line.demand })}
              </span>
            </span>
          </div>
          <p className="text-2xs text-ink-500">
            {joinNodes(parts, ' · ')}
            {line.famine && (
              <span className="font-semibold text-vermilion-700">
                {parts.length > 0 && ' · '}
                {line.warned
                  ? t('reports.economyMissing', { count: line.missing })
                  : t('reports.economyFamine', {
                      count: line.missing,
                      lost: line.troopsLost ?? 0,
                    })}
              </span>
            )}
          </p>
        </div>
      )
    }

    const heading = 'text-2xs font-semibold uppercase tracking-wide text-ink-500'
    return (
      <li key={key} data-report-house={key}>
        <HouseBlock
          shieldId={`economy-${key}`}
          open={key === viewerId || houses.length === 1}
          arms={arms}
          name={name}
          balance={
            <>
              <span className={netClass(net)}>{signed(net)}</span>
              {armyTally(houseSupply, t)}
            </>
          }
          inner="space-y-3 px-2.5 py-2"
        >
          {(houseIncome.length > 0 || loose.length > 0) && (
            <section className="space-y-1.5">
              <h4 className={`${heading} flex items-center justify-between gap-3`}>
                {t('reports.economyIn')}
                <span className="text-success-700">{gain(earned)}</span>
              </h4>
              <ul className="space-y-1.5">
                {houseIncome.map(renderIncome)}
                {loose.length > 0 && (
                  <li className="rounded-md border border-line-soft bg-paper-50 px-2 py-1.5">
                    <ul className="space-y-0.5">{loose.map(millRow)}</ul>
                  </li>
                )}
              </ul>
            </section>
          )}
          {houseSupply.length > 0 && (
            <section className="space-y-1.5">
              <h4 className={`${heading} flex items-center justify-between gap-3`}>
                {t('reports.economyOut', { armies: armyCount(houseSupply.length) })}
                <span className="text-vermilion-700">
                  {cost(consumedOf(houseSupply))}{' '}
                  <span className="font-normal normal-case tracking-normal text-ink-500">
                    {t('reports.economyDemand', { count: demandOf(houseSupply) })}
                  </span>
                </span>
              </h4>
              <RegionBlocks
                items={houseSupply}
                map={map}
                regionOf={(line) => regionSeedOf(map, line.territory)}
                territoryOf={(line) => line.territory}
                keyOf={(line) => line.army}
                renderItem={renderArmy}
                subtotal={(group) => (
                  <>
                    {armyCount(group.length)} ·{' '}
                    <span className="text-vermilion-700">{cost(consumedOf(group))}</span>
                  </>
                )}
              />
            </section>
          )}
          <div className="space-y-0.5 border-t border-line pt-1.5">
            <div className="flex items-center justify-between gap-3 text-sm font-semibold">
              <span className="text-ink-900">{t('reports.economyNet')}</span>
              <span className={netClass(net)}>{signed(net)}</span>
            </div>
            {stock && (
              <div className="flex items-center justify-between gap-3 text-xs text-ink-700">
                <span>{t('reports.economyStock')}</span>
                <span className="font-semibold">
                  {t('reports.economyStockValue', {
                    before: stock.resourcesBefore,
                    after: stock.resourcesAfter,
                  })}
                </span>
              </div>
            )}
          </div>
        </HouseBlock>
      </li>
    )
  }

  return (
    <SummaryCard
      icon={<IconCoins />}
      summary={t('reports.summary.economy', {
        net: signed(
          sum(income.map(lineTotal)) + sum(mills.map(millTotal)) - consumedOf(supply),
        ),
        armies: armyTally(supply, t),
      })}
    >
      {houses.map(renderHouse)}
    </SummaryCard>
  )
}

/**
 * One sentence saying what a winter order did, in the viewer's language;
 * nothing for an order the report only knows as rejected or whose own detail
 * already says it (claim, ritual).
 */
function winterDescription(
  investment: WinterInvestmentReport,
  map: MapData | null,
  t: Translate,
): ReactNode {
  const territory = territoryLabel(map, investment.territory, t)
  const noble = nobleLabel(investment.nobleCode, investment.nobleName)
  const infra = investment.type
    ? t(`reports.winterInfra.${investment.type}` as MessageKey)
    : ''
  const sentence = (key: string, values: Record<string, ReactNode> = {}) =>
    richTranslate(t, `reports.winterDesc.${key}` as MessageKey, values)
  switch (investment.kind) {
    case 'recruit':
      return investment.nobleCode
        ? sentence('recruitNoble', { noble, territory })
        : sentence('recruit', { territory })
    case 'build':
      return investment.type ? sentence('build', { infra, territory }) : null
    case 'upgrade':
      if (!investment.type) return null
      return investment.level
        ? sentence('upgradeLevel', { infra, territory, level: investment.level })
        : sentence('upgrade', { infra, territory })
    case 'fortify':
    case 'capital_elected':
      return sentence(investment.kind, { territory })
    case 'noble_transfer':
    case 'fief_assigned':
    case 'fief_auto_assigned':
    case 'fief_annexed':
    case 'fief_rank_changed':
      return sentence(investment.kind, { noble, territory })
    case 'fief_founded':
      return sentence('fief_founded', { noble, territory })
    case 'transfer':
      return sentence('transfer', {
        amount: investment.amount ?? '—',
        source: territoryLabel(map, investment.source, t),
        target: territoryLabel(map, investment.target, t),
      })
    case 'marriage_refused':
    case 'inquiry':
    case 'trial_filed':
      return sentence(investment.kind, { noble })
    case 'noble_draw':
    case 'noble_discard':
      return sentence(investment.kind)
    case 'dignity':
      return sentence('dignity', {
        noble,
        dignity: t(`dignity.${investment.dignity ?? 'bastard'}` as MessageKey),
      })
    case 'prosperity_founded':
      return sentence('prosperity_founded', {
        source: territoryLabel(map, investment.source, t),
        target: territoryLabel(map, investment.target, t),
      })
    default:
      return null
  }
}

/**
 * One house of a report card: a folded block opened on its shield and name,
 * with a one-line balance on the right. `inner` sets the body's spacing.
 */
function HouseBlock({
  shieldId,
  winterHouse,
  open,
  arms,
  name,
  balance,
  inner,
  children,
}: {
  shieldId: string
  /** The house key a winter block carries as `data-winter-house`. */
  winterHouse?: string
  open: boolean
  arms: ReturnType<typeof playerArms>
  name: ReactNode
  balance: ReactNode
  inner: string
  children: ReactNode
}) {
  return (
    <details
      data-winter-house={winterHouse}
      open={open}
      className="rounded-md border border-line-soft bg-paper-100/40"
    >
      <summary className="flex cursor-pointer items-center justify-between gap-3 px-2.5 py-1.5">
        <span className="flex min-w-0 items-center gap-2 font-display text-sm font-semibold text-ink-900">
          <ShieldIcon id={shieldId} arms={arms} className="h-5 w-4 shrink-0" />
          {name}
        </span>
        <span className="shrink-0 text-xs font-semibold text-ink-700">{balance}</span>
      </summary>
      <div className={`border-t border-line-soft ${inner}`}>{children}</div>
    </details>
  )
}

/**
 * The shield and name of a house block: a player's own, the neutral stocks',
 * or the realm's for the winter events no house ordered.
 */
function houseIdentity(
  kind: WinterHouseKind,
  key: string,
  players: Player[],
  t: Translate,
) {
  const player = players.find((candidate) => candidate.id === key)
  const name =
    player?.name ??
    (kind === 'realm'
      ? t('reports.winterRealm')
      : kind === 'neutral'
        ? t('reports.groupNeutral')
        : key)
  return { arms: player ? playerArms(player) : NEUTRAL_ARMS, name }
}

/** One winter order: what it did in a sentence, its code beneath, its outcome and cost. */
function WinterOrderCard({
  investment,
  map,
}: {
  investment: WinterInvestmentReport
  map: MapData | null
}) {
  const { t } = useLanguage()
  const isAutoAssignedWarning = investment.kind === 'fief_auto_assigned'
  const description = winterDescription(investment, map, t)
  const territories = [
    investment.territory,
    investment.target,
    investment.source,
    ...(investment.territories ?? []),
  ].filter((id): id is string => Boolean(id))
  return (
    <EventCard
      dense
      icon={<IconSnowflake />}
      tone={isAutoAssignedWarning ? 'warning' : undefined}
      title={
        <>
          {description ?? (
            <strong className="font-mono text-xs">
              {investmentLabel(investment, map, t)}
            </strong>
          )}
          {isAutoAssignedWarning && (
            <span className="block text-2xs font-bold uppercase tracking-wide text-warning-700">
              {t('reports.fiefAutoAssignedWarning')}
            </span>
          )}
        </>
      }
      detail={
        <>
          {description && (
            <span className="font-mono text-2xs text-ink-500">
              {investmentLabel(investment, map, t)}
            </span>
          )}
          {(!description ||
            (investment.reason && investment.kind !== 'prosperity_founded')) && (
            <span className="block">{winterDetails(investment, map, t)}</span>
          )}
        </>
      }
      aside={
        <span className={outcomeClass(investment.outcome)}>
          {outcomeLabel(investment.outcome, t)}
          {investment.outcome === 'success' && investment.cost > 0 && (
            <span className="font-normal text-ink-500">
              {' · '}
              {t('reports.winterOrderCost', { cost: investment.cost })}
            </span>
          )}
        </span>
      }
      marks={
        territories.length > 0 ? { territories: [...new Set(territories)] } : undefined
      }
    />
  )
}

/**
 * The winter orders and the stocks kept through winter, one block per house:
 * the viewer's house first and open, the others folded on a one-line balance.
 * The grouping itself is lib/winter-report.
 */
function WinterHouses({
  investments,
  stocks,
  map,
  players,
  viewerId,
}: {
  investments: WinterInvestmentReport[]
  stocks: WinterStockReport[]
  map: MapData | null
  players: Player[]
  viewerId?: PlayerId
}) {
  const { t } = useLanguage()
  const houses = groupWinterReport(investments, stocks, players, viewerId)
  const heading =
    'flex items-center gap-1.5 text-2xs font-semibold uppercase tracking-wide text-ink-500 [&_svg]:size-3.5'

  return (
    <>
      {houses.map((house) => {
        const { arms, name } = houseIdentity(house.kind, house.key, players, t)
        const balance = [
          house.orderCount > 0
            ? t(
                house.orderCount === 1
                  ? 'reports.winterHouseOrder'
                  : 'reports.winterHouseOrders',
                { count: house.orderCount },
              )
            : null,
          house.failed > 0
            ? t('reports.winterHouseFailed', { count: house.failed })
            : null,
          house.cost > 0 ? t('reports.winterHouseCost', { cost: house.cost }) : null,
          house.halved.length > 0
            ? t('reports.winterHouseStocks', { count: house.halved.length })
            : null,
        ]
          .filter((part): part is string => part !== null)
          .join(' · ')
        return (
          <HouseBlock
            key={house.key}
            winterHouse={house.key}
            shieldId={`winter-${house.kind === 'realm' ? 'realm' : house.key}`}
            open={house.key === viewerId || house.kind === 'realm' || houses.length === 1}
            arms={arms}
            name={name}
            balance={balance}
            inner="space-y-2 px-2 py-1.5"
          >
            {house.groups.map(({ group, orders }) => (
              <section key={group} data-winter-group={group} className="space-y-1">
                <h5 className={heading}>{t(`reports.winterGroup.${group}`)}</h5>
                {orders.map((investment, index) => (
                  <WinterOrderCard
                    key={`${investment.kind}-${index}`}
                    investment={investment}
                    map={map}
                  />
                ))}
              </section>
            ))}
            {house.halved.length > 0 && (
              <section data-winter-group="conservation" className="space-y-1">
                <h5 className={heading}>
                  <IconBuildingWarehouse />
                  {t('reports.winterGroup.conservation')}
                </h5>
                <ul className="space-y-0.5">
                  {house.halved.map((stock) => (
                    <li key={stock.territory} className="text-xs">
                      {richTranslate(t, 'reports.conservationLine', {
                        territory: territoryLabel(map, stock.territory, t),
                        before: stock.stockBefore,
                        after: stock.stockAfter,
                      })}
                    </li>
                  ))}
                </ul>
                <p className="border-t border-line-soft pt-1 text-xs font-semibold text-ink-700">
                  {t(
                    house.received === 0
                      ? 'reports.winterBalance'
                      : 'reports.winterBalanceReceived',
                    {
                      before: house.before,
                      cost: house.cost,
                      received: house.received,
                      left: house.left,
                      after: house.after,
                    },
                  )}
                </p>
              </section>
            )}
          </HouseBlock>
        )
      })}
    </>
  )
}

/** Results of one kind folded into a single card, unfolding on the list. */
function SummaryCard({
  icon,
  summary,
  children,
}: {
  icon: ReactNode
  summary: ReactNode
  children: ReactNode
}) {
  return (
    <details
      data-summary-card
      className="rounded-md border border-line bg-paper-50 text-sm"
    >
      <summary className="flex cursor-pointer items-center gap-2 px-3 py-2 font-medium [&_svg]:size-4">
        <span aria-hidden="true" className="flex shrink-0">
          {icon}
        </span>
        {summary}
      </summary>
      <ul className="space-y-1.5 border-t border-line-soft px-3 py-2">{children}</ul>
    </details>
  )
}
