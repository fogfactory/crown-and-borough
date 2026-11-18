import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { GameLobby, orderCandidates } from '@/components/GameLobby'
import { MapPreview } from '@/components/MapPreview'
import { LanguageProvider } from '@/i18n/LanguageContext'
import type { GameLobby as LobbyData, CandidateMapData } from '@/types'

const square = (x: number, y: number): Array<[number, number]> => [
  [x, y],
  [x + 10, y],
  [x + 10, y + 10],
  [x, y + 10],
]

const candidateMap: CandidateMapData = {
  territories: [
    { id: 'AAA', name: 'A', terrain: 'plain', village: false, points: square(0, 0), adjacencies: ['BBB'], impassable: [] },
    { id: 'BBB', name: 'B', terrain: 'forest', village: true, points: square(10, 0), adjacencies: ['AAA', 'CCC'], impassable: [] },
    { id: 'CCC', name: 'C', terrain: 'hill', village: false, points: square(20, 0), adjacencies: ['BBB'], impassable: [] },
  ],
  regions: [
    { id: 'R1', seed: 'AAA', territories: ['AAA', 'BBB'] },
    { id: 'R2', seed: 'CCC', territories: ['CCC'] },
  ],
  starts: [],
}

const fetchMock = vi.fn()

const lobby: LobbyData = {
  seedCandidates: [
    { index: 0, seed: 'aaaa' },
    { index: 1, seed: 'bbbb' },
    { index: 2, seed: 'cccc' },
  ],
  seedRanking: [],
  houseName: '',
}

function renderLobby(overrides: Partial<LobbyData> = {}, onVote = vi.fn().mockResolvedValue(undefined)) {
  render(
    <LanguageProvider initialLanguage="en">
      <GameLobby
        gameId="game-1"
        tokenProvider={{ getIdToken: async () => 'token' }}
        lobby={{ ...lobby, ...overrides }}
        houseName=""
        onRename={vi.fn().mockResolvedValue(undefined)}
        onVote={onVote}
      />
    </LanguageProvider>,
  )
  return onVote
}

describe('MapPreview', () => {
  it('draws every territory with its terrain, tinted by region and bordered by region', () => {
    const { container } = render(<MapPreview map={candidateMap} label="Preview" />)
    expect(screen.getByRole('img', { name: 'Preview' })).toBeInTheDocument()
    expect(container.querySelectorAll('[data-preview-territory]')).toHaveLength(3)
    expect(container.querySelectorAll('[data-preview-terrain]')).toHaveLength(3)
    expect(container.querySelectorAll('path[data-preview-region]')).toHaveLength(2)
    // No text: names stay off the preview.
    expect(container.querySelector('text')).toBeNull()
  })
})

describe('orderCandidates', () => {
  it('puts the saved ranking first, then the unranked candidates', () => {
    expect(orderCandidates(lobby.seedCandidates, [2])).toEqual([2, 0, 1])
    expect(orderCandidates(lobby.seedCandidates, lobby.seedRanking)).toEqual([0, 1, 2])
  })
})

describe('GameLobby', () => {
  beforeEach(() => {
    fetchMock.mockReset()
    fetchMock.mockImplementation(async () =>
      new Response(JSON.stringify(candidateMap), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
  })

  it('shows a preview of every candidate map', async () => {
    renderLobby()
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(3))
    const urls = fetchMock.mock.calls.map((call) => String(call[0]))
    expect(urls).toEqual(
      expect.arrayContaining([
        '/api/games/game-1/seed-candidates/0/map',
        '/api/games/game-1/seed-candidates/1/map',
        '/api/games/game-1/seed-candidates/2/map',
      ]),
    )
    expect(screen.getByText('You have not voted yet.')).toBeInTheDocument()
  })

  it('reorders the candidates and saves the ranking', async () => {
    const onVote = renderLobby()
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(3))
    fireEvent.click(screen.getByRole('button', { name: 'Move map 3 up' }))
    fireEvent.click(screen.getByRole('button', { name: 'Move map 2 up' }))
    const cards = screen.getAllByRole('listitem')
    expect(cards.map((card) => card.getAttribute('data-seed-candidate'))).toEqual(['2', '0', '1'])
    expect(within(cards[0]).getByText('cccc')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save my vote' }))
    await waitFor(() => expect(onVote).toHaveBeenCalledWith([2, 0, 1]))
  })

  it('names a game whose seed the creator imposed without a vote', () => {
    renderLobby({ seedCandidates: [] })
    expect(screen.getByText('The seed of this game was chosen by its creator.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save my vote' })).toBeNull()
  })

  it('offers the house name field with its note', () => {
    renderLobby()
    expect(screen.getByLabelText('Your house name')).toBeInTheDocument()
    expect(screen.getByText(/until you submit your first orders/)).toBeInTheDocument()
  })
})
