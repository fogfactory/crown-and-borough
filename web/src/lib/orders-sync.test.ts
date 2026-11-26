import { describe, expect, it } from 'vitest'

import {
  compareOrders,
  normalizeOrders,
  normalizeOrdersText,
  submissionBody,
} from '@/lib/orders-sync'
import type { MySubmissionResponse } from '@/types'

const pending: MySubmissionResponse = {
  turn: 1,
  season: 'spring',
  status: 'pending',
  chains: [],
}
const draft: MySubmissionResponse = {
  ...pending,
  status: 'draft',
  chains: [{ noble: 'JEA', text: 'JEA\nROS H' }],
  winter: { lines: 'R N TST ROS' },
  special: [{ text: 'P RA ROS' }],
}
const sealed: MySubmissionResponse = { ...draft, status: 'submitted' }

function server(submission: MySubmissionResponse) {
  return normalizeOrders(submissionBody(submission))
}

describe('normalizeOrdersText', () => {
  it('ignores line endings, trailing spaces and blank edges', () => {
    expect(normalizeOrdersText('\r\nROS H  \r\nBRU A ROS\n\n')).toBe('ROS H\nBRU A ROS')
  })
})

describe('normalizeOrders', () => {
  it('drops the noble header and the empty chains', () => {
    expect(
      normalizeOrders({
        chains: [
          { noble: 'JEA', text: 'JEA\nROS H' },
          { noble: 'HUG', text: 'HUG\n' },
        ],
        winter: [{ lines: 'T N' }],
        special: [{ text: 'a' }, { text: 'b' }],
      }),
    ).toEqual({ chains: { JEA: 'ROS H' }, winter: 'T N', special: 'a\nb' })
  })
})

describe('compareOrders against the server version', () => {
  const local = normalizeOrders({
    chains: [{ noble: 'JEA', text: 'JEA\nROS H' }],
    winter: [{ lines: 'R N TST ROS' }],
    special: [{ text: 'P RA ROS' }],
  })

  it('compares with an empty version while nothing is recorded', () => {
    expect(compareOrders(local, server(pending))).toMatchObject({
      any: true,
      chains: { JEA: true },
      winter: true,
      special: true,
    })
    expect(compareOrders(normalizeOrders({}), server(pending)).any).toBe(false)
  })

  it('compares with the saved sketch', () => {
    expect(compareOrders(local, server(draft)).any).toBe(false)
    const edited = { ...local, chains: { JEA: 'ROS A BRU' } }
    expect(compareOrders(edited, server(draft))).toMatchObject({
      any: true,
      chains: { JEA: true },
      winter: false,
      special: false,
    })
  })

  it('compares with the sealed orders, whatever the status', () => {
    expect(compareOrders(local, server(sealed)).any).toBe(false)
    expect(compareOrders({ ...local, special: '' }, server(sealed))).toMatchObject({
      any: true,
      special: true,
    })
  })

  it('sees a chain emptied locally as a difference', () => {
    expect(compareOrders({ ...local, chains: {} }, server(sealed)).chains).toEqual({ JEA: true })
  })
})
