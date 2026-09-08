import { describe, expect, it } from 'vitest'

import { transferTargetsForTerritory } from '@/lib/transfer-preview'

describe('transferTargetsForTerritory', () => {
  it('collects transfer destinations from valid drafts at the selected source', () => {
    expect(
      transferTargetsForTerritory(
        {
          JEA: 'ROS T BRU 1\nROS T ATL 2\nBRU H',
          HUG: 'ROS A NOR',
        },
        'ROS',
      ),
    ).toEqual(['BRU', 'ATL'])
  })

  it('ignores malformed orders and drafts for another source', () => {
    expect(
      transferTargetsForTerritory(
        {
          JEA: 'ROS T BRU no\nBRU T ATL 1\nROS A NOR',
        },
        'ROS',
      ),
    ).toEqual([])
  })
})
