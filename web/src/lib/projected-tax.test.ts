import { describe, expect, it } from 'vitest'
import { draftedTaxEffects, parseDraftTaxLine } from '@/lib/projected-tax'
import type { StateData } from '@/types'

const state = {
  king: 'KKK',
  nobles: [
    { code: 'KKK', owner: 'P1' },
    { code: 'LLL', owner: 'P2' },
  ],
  fiefs: [
    { capital: 'AAA', owner: 'P2', holder: 'LLL', projectedIncome: 4 },
    { capital: 'BBB', owner: 'P1', holder: 'KKK', projectedIncome: 3 },
    { capital: 'CCC', owner: 'P3', projectedIncome: 5 },
  ],
} as unknown as StateData

describe('draftedTaxEffects', () => {
  it('counts a royal tax supplement on a fief of another player', () => {
    expect(draftedTaxEffects(state, 'P1', 'P TX KKK AAA\nP TX KKK CCC # x')).toEqual({
      seigneurial: {},
      royalSupplement: 9,
      tithe: 0,
      titheDrafted: false,
      royalDrafted: true,
    })
  })

  it('doubles the fief of a seigneurial tax and ignores tithes', () => {
    expect(draftedTaxEffects(state, 'P1', 'P TX KKK BBB\nP DI KKK AAA')).toEqual({
      seigneurial: { BBB: 3 },
      royalSupplement: 0,
      tithe: 0,
      titheDrafted: false,
      royalDrafted: false,
    })
  })

  it('ignores a royal tax from a noble who is not the king', () => {
    expect(draftedTaxEffects(state, 'P2', 'P TX LLL CCC').royalSupplement).toBe(0)
  })

  it('lets the regent, and only her, tax in place of a held king', () => {
    const court = {
      ...state,
      queen: 'QQQ',
      nobles: [
        { code: 'KKK', owner: 'P1', status: 'free' },
        { code: 'QQQ', owner: 'P2', status: 'free' },
      ],
    } as unknown as StateData
    // The king reigns: his queen has no power.
    expect(draftedTaxEffects(court, 'P2', 'P TX QQQ CCC').royalSupplement).toBe(0)
    // The king is in a dungeon: his wife governs, and he no longer does.
    const held = {
      ...court,
      nobles: [
        { code: 'KKK', owner: 'P1', status: 'dungeon' },
        { code: 'QQQ', owner: 'P2', status: 'free' },
      ],
    } as unknown as StateData
    expect(draftedTaxEffects(held, 'P2', 'P TX QQQ CCC').royalSupplement).toBe(5)
    expect(draftedTaxEffects(held, 'P1', 'P TX KKK CCC').royalSupplement).toBe(0)
  })
})

describe('parseDraftTaxLine', () => {
  it('reads the three shapes of a Tax card line, with or without comment', () => {
    expect(parseDraftTaxLine('P TX AAA BBB')).toEqual({ tithe: false, issuer: 'AAA', target: 'BBB' })
    expect(parseDraftTaxLine('P TX BBB # Impôts (TX) sur BBB')).toEqual({ tithe: false, target: 'BBB' })
    expect(parseDraftTaxLine('P DI AAA # sur tous les évêchés')).toEqual({ tithe: true, issuer: 'AAA' })
    expect(parseDraftTaxLine('P DI AAA RRB')).toEqual({ tithe: true, issuer: 'AAA', target: 'RRB' })
  })

  it('ignores every other line', () => {
    expect(parseDraftTaxLine('P BT ROS')).toBeNull()
    expect(parseDraftTaxLine('D C BT')).toBeNull()
    expect(parseDraftTaxLine('')).toBeNull()
  })
})

describe('drafted tithes', () => {
  const tithed = {
    nobles: [
      { code: 'PPP', owner: 'P1', religiousTitle: 'pope' },
      { code: 'CCC', owner: 'P1', religiousTitle: 'cardinal' },
    ],
    fiefs: [],
    bishoprics: [
      { region: 'RRA', territories: ['T1', 'T2'] },
      { region: 'RRB', territories: ['T3'] },
    ],
    territories: [
      { id: 'T1', owner: 'P2', millProduction: 3 },
      { id: 'T2', owner: 'P1', millProduction: 5 },
      { id: 'T3', owner: 'P3', millProduction: 2 },
    ],
  } as unknown as StateData

  it('counts the other players mills of every bishopric for the pope', () => {
    expect(draftedTaxEffects(tithed, 'P1', 'P DI PPP').tithe).toBe(5)
  })

  it('flags a drafted tithe even when no mill is diverted', () => {
    const effects = draftedTaxEffects(tithed, 'P1', 'P DI CCC RRZ')
    expect(effects.titheDrafted).toBe(false)
    expect(draftedTaxEffects(tithed, 'P1', 'P DI PPP').titheDrafted).toBe(true)
  })

  it('reads the lines the order dialog writes, with their comment', () => {
    expect(draftedTaxEffects(tithed, 'P1', 'P DI PPP # Impôts (TX) sur tous les évêchés').titheDrafted).toBe(true)
    expect(draftedTaxEffects(tithed, 'P1', 'P DI CCC RRB # Impôts (TX) sur RRB').tithe).toBe(2)
  })

  it('does not take the pope’s seigneurial tax on a fief for a tithe', () => {
    const lord = { ...tithed, fiefs: [{ capital: 'RRA', owner: 'P1', holder: 'PPP', projectedIncome: 3 }] } as unknown as StateData
    const effects = draftedTaxEffects(lord, 'P1', 'P TX PPP RRA')
    expect(effects.titheDrafted).toBe(false)
    expect(effects.seigneurial).toEqual({ RRA: 3 })
  })

  it('counts only the targeted bishopric for a cardinal', () => {
    expect(draftedTaxEffects(tithed, 'P1', 'P DI CCC RRB').tithe).toBe(2)
  })
})
