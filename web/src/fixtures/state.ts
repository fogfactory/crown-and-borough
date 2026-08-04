import type { StateData } from '@/types'

// TRANSITIONAL demo fixture: materializes the neutral villages of the dev
// seed's map ("crown-and-borough-dev", VillageCount = 5) as ownerless
// village infrastructures on uncontrolled territories, so the front renders
// them via the state layer. Valid as long as the dev seed and generation
// stay put; P1.2f replaces this stub with a real generated demo state
// (internal/engine/demo.DemoState).
export const stateFixture: StateData = {
  turn: 1,
  season: 'spring',
  asOf: {},
  territories: [
    {
      id: 'T19',
      owner: null,
      resources: 0,
      troops: [],
      infrastructures: [{ type: 'village', level: 1 }],
    },
    {
      id: 'T36',
      owner: null,
      resources: 0,
      troops: [],
      infrastructures: [{ type: 'village', level: 1 }],
    },
    {
      id: 'T46',
      owner: null,
      resources: 0,
      troops: [],
      infrastructures: [{ type: 'village', level: 1 }],
    },
    {
      id: 'T61',
      owner: null,
      resources: 0,
      troops: [],
      infrastructures: [{ type: 'village', level: 1 }],
    },
    {
      id: 'T64',
      owner: null,
      resources: 0,
      troops: [],
      infrastructures: [{ type: 'village', level: 1 }],
    },
  ],
  nobles: [],
}