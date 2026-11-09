import { describe, expect, it } from 'vitest'

import {
  buildMentionIndex,
  tokenizeMentions,
  type MentionSegment,
} from '@/lib/chat-mentions'

const index = buildMentionIndex({
  territories: [
    { id: 'RVE', name: 'Rochevent' },
    { id: 'HAU', name: 'Hautebourg' },
    { id: 'MIC', name: 'Mi-Cap' },
    { id: 'LSA', name: 'Les Saules' },
    { id: 'BLA', name: 'Blanche' },
  ],
  nobles: [
    { code: 'HUO', name: 'Baron Hugon de Rochevent', firstName: 'Hugon' },
    { code: 'AGN', name: 'Dame Agnès de Mi-Cap', firstName: 'Agnès' },
    { code: 'AG2', name: 'Sieur Agnès de Hautebourg', firstName: 'Agnès' },
    { code: 'BL1', name: 'Dame Blanche de Rochevent', firstName: 'Blanche' },
  ],
})

/** The mentions of a text, as « written → territory|nobles ». */
function mentions(text: string): string[] {
  return tokenizeMentions(text, index)
    .filter(
      (segment): segment is Extract<MentionSegment, { kind: 'mention' }> =>
        segment.kind === 'mention',
    )
    .map(
      ({ text: written, target }) =>
        `${written} → ${[target.territory ?? '', ...target.nobles].join('|')}`,
    )
}

describe('tokenizeMentions', () => {
  it('keeps the whole text, in order', () => {
    const text = "Nos armées passeront à Rochevent, près d'Hautebourg."
    const segments = tokenizeMentions(text, index)
    expect(segments.map((segment) => segment.text).join('')).toBe(text)
    expect(segments.map((segment) => segment.kind)).toEqual([
      'text',
      'mention',
      'text',
      'mention',
      'text',
    ])
  })

  it('finds communes, elided or compound', () => {
    expect(mentions("Rochevent tombera, puis d'Hautebourg à Mi-Cap.")).toEqual([
      'Rochevent → RVE',
      'Hautebourg → HAU',
      'Mi-Cap → MIC',
    ])
    expect(mentions('le marché d’Hautebourg')).toEqual(['Hautebourg → HAU'])
  })

  it('accepts a lower-case leading article only', () => {
    expect(mentions('vers les Saules et Les Saules')).toEqual([
      'les Saules → LSA',
      'Les Saules → LSA',
    ])
    expect(mentions('les saules, rochevent, ROCHEVENT')).toEqual([])
  })

  it('never matches part of a word', () => {
    expect(
      mentions('Rocheventais, Saint-Rochevent, Rochevent-sur-Mer, Mi-Capitale, Hugonnet'),
    ).toEqual([])
  })

  it('prefers the longest name', () => {
    expect(mentions('Hugon de Rochevent et Baron Hugon de Rochevent')).toEqual([
      'Hugon de Rochevent → |HUO',
      'Baron Hugon de Rochevent → |HUO',
    ])
    expect(mentions('Hugon garde Rochevent')).toEqual(['Hugon → |HUO', 'Rochevent → RVE'])
  })

  it('points a shared name to everyone who bears it', () => {
    expect(mentions('Agnès')).toEqual(['Agnès → |AGN|AG2'])
    expect(mentions('Blanche')).toEqual(['Blanche → BLA|BL1'])
    expect(mentions('Agnès de Mi-Cap')).toEqual(['Agnès de Mi-Cap → |AGN'])
  })

  it('finds territory and noble codes, in capitals and whole', () => {
    expect(mentions('HUO marche de RVE sur MIC (via HAU).')).toEqual([
      'HUO → |HUO',
      'RVE → RVE',
      'MIC → MIC',
      'HAU → HAU',
    ])
    expect(mentions('rve, Rve, RVEX, XRVE, RVE-HAU, AGN2')).toEqual([])
  })

  it('points a code shared by a commune and a noble to both', () => {
    // Codes are unique per kind only: Beaufort and Béatrix are both BEA.
    const shared = buildMentionIndex({
      territories: [{ id: 'BEA', name: 'Beaufort' }],
      nobles: [{ code: 'BEA', name: 'Dame Béatrix de Beaufort', firstName: 'Béatrix' }],
    })
    expect(tokenizeMentions('BEA', shared)).toEqual([
      { kind: 'mention', text: 'BEA', target: { territory: 'BEA', nobles: ['BEA'] } },
    ])
  })

  it('finds names typed with decomposed accents', () => {
    expect(mentions('Agne\u0300s')).toEqual(['Agnès → |AGN|AG2'])
  })

  it('never indexes a form of address without a first name', () => {
    const bare = buildMentionIndex({
      territories: [],
      nobles: [{ code: 'HUO', name: 'Baron Hugon de Rochevent' }],
    })
    expect(tokenizeMentions('Baron, Hugon', bare)).toEqual([
      { kind: 'text', text: 'Baron, Hugon' },
    ])
    expect(tokenizeMentions('HUO', bare)[0].kind).toBe('mention')
  })

  it('leaves text alone without names', () => {
    const empty = buildMentionIndex({ territories: [], nobles: [] })
    expect(tokenizeMentions('Rochevent', empty)).toEqual([
      { kind: 'text', text: 'Rochevent' },
    ])
    expect(tokenizeMentions('', index)).toEqual([])
  })

  it('treats names as text, not patterns', () => {
    const odd = buildMentionIndex({
      territories: [{ id: 'ODD', name: 'Val (Haut)' }],
      nobles: [],
    })
    expect(tokenizeMentions('au Val (Haut).', odd)[1]).toEqual({
      kind: 'mention',
      text: 'Val (Haut)',
      target: { territory: 'ODD', nobles: [] },
    })
    expect(tokenizeMentions('<b>Val</b>', odd)).toEqual([
      { kind: 'text', text: '<b>Val</b>' },
    ])
  })
})
