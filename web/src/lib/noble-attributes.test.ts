import { describe, expect, it } from 'vitest'

import { nobleAttributesByCode } from '@/lib/noble-attributes'
import type { Noble, StateData } from '@/types'

function noble(code: string, extra: Partial<Noble> = {}): Noble {
  return {
    id: code,
    code,
    name: code,
    owner: 'P1',
    location: 'ROS',
    status: 'free',
    ...extra,
  }
}

function stateWith(extra: Partial<StateData>): StateData {
  return { nobles: [], fiefs: [], ...extra } as unknown as StateData
}

describe('nobleAttributesByCode', () => {
  it('crowns a noble with its highest secular title, the throne first', () => {
    const attributes = nobleAttributesByCode(
      stateWith({
        nobles: [noble('DUC'), noble('ROI')],
        fiefs: [
          {
            capital: 'AAA',
            title: 'barony',
            territories: [],
            owner: 'P1',
            holder: 'DUC',
          },
          { capital: 'BBB', title: 'duchy', territories: [], owner: 'P1', holder: 'DUC' },
          {
            capital: 'CCC',
            title: 'county',
            territories: [],
            owner: 'P1',
            holder: 'ROI',
          },
        ],
        king: 'ROI',
      }),
    )
    expect(attributes.get('DUC')?.crown).toBe('duchy')
    expect(attributes.get('ROI')?.crown).toBe('king')
  })

  it('gives the spouse of a titled noble, and the queen, a courtesy crown', () => {
    const attributes = nobleAttributesByCode(
      stateWith({
        nobles: [
          noble('COM', { spouse: 'CSE' }),
          noble('CSE', { spouse: 'COM' }),
          noble('REI'),
        ],
        fiefs: [
          {
            capital: 'AAA',
            title: 'county',
            territories: [],
            owner: 'P1',
            holder: 'COM',
          },
        ],
        queen: 'REI',
        queenDowager: true,
      }),
    )
    expect(attributes.get('CSE')).toMatchObject({ courtesyCrown: 'county' })
    expect(attributes.get('CSE')?.crown).toBeUndefined()
    expect(attributes.get('COM')?.courtesyCrown).toBeUndefined()
    expect(attributes.get('REI')?.courtesyCrown).toBe('king')
  })

  it('adds the religious attributes, the pope robe over the cardinal one', () => {
    const attributes = nobleAttributesByCode(
      stateWith({
        nobles: [noble('PAP'), noble('CAR'), noble('EVE'), noble('EXC'), noble('RAS')],
        pope: 'PAP',
        cardinals: ['PAP', 'CAR'],
        bishoprics: [{ region: 'R1', name: 'Ros', territories: [], bishop: 'EVE' }],
        excommunicated: [{ noble: 'EXC', reason: 'papal', turn: 1 }],
      }),
    )
    expect(attributes.get('PAP')?.robe).toBe('pope')
    expect(attributes.get('CAR')?.robe).toBe('cardinal')
    expect(attributes.get('EVE')).toMatchObject({ crosier: true, robe: undefined })
    expect(attributes.get('EXC')?.excommunicated).toBe(true)
    // A noble with no title wears nothing.
    expect(attributes.has('RAS')).toBe(false)
  })
})
