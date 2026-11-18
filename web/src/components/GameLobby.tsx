import { useEffect, useState } from 'react'
import { IconArrowDown, IconArrowUp } from '@tabler/icons-react'

import { HouseNameForm } from '@/components/HouseNameForm'
import { MapPreview } from '@/components/MapPreview'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import { apiRequest, type TokenProvider } from '@/lib/api'
import { errorText } from '@/lib/api-error'
import type { GameLobby as LobbyData, MapData, SeedCandidate } from '@/types'

/**
 * Candidate seeds in the order the player ranks them: the saved ranking first,
 * then the candidates left unranked in their own order.
 */
export function orderCandidates(candidates: SeedCandidate[], ranking: number[]): number[] {
  const indexes = candidates.map((candidate) => candidate.index)
  const ranked = ranking.filter((index) => indexes.includes(index))
  return [...ranked, ...indexes.filter((index) => !ranked.includes(index))]
}

export function rankingOrder(lobby: LobbyData): number[] {
  return orderCandidates(lobby.seedCandidates, lobby.seedRanking)
}

function moved(order: number[], position: number, delta: -1 | 1): number[] {
  const target = position + delta
  if (target < 0 || target >= order.length) return order
  const next = [...order]
  ;[next[position], next[target]] = [next[target], next[position]]
  return next
}

function CandidatePreview({
  gameId,
  index,
  rank,
  tokenProvider,
  inviteCode,
}: {
  gameId: string
  index: number
  rank: number
  tokenProvider: TokenProvider
  inviteCode?: string
}) {
  const { t } = useLanguage()
  const [map, setMap] = useState<MapData | null>(null)
  const [failed, setFailed] = useState(false)
  const { getIdToken } = tokenProvider

  useEffect(() => {
    let active = true
    setMap(null)
    setFailed(false)
    void apiRequest<MapData>(
      { getIdToken },
      `/api/games/${encodeURIComponent(gameId)}/seed-candidates/${index}/map${
        inviteCode ? `?inviteCode=${encodeURIComponent(inviteCode)}` : ''
      }`,
    )
      .then((response) => {
        if (active) setMap(response)
      })
      .catch(() => {
        if (active) setFailed(true)
      })
    return () => {
      active = false
    }
  }, [gameId, getIdToken, index, inviteCode])

  if (failed) {
    return <Alert role="alert">{t('lobby.previewFailed')}</Alert>
  }
  if (!map) {
    return <p className="py-8 text-center text-sm italic text-ink-500">{t('lobby.previewLoading')}</p>
  }
  return <MapPreview map={map} label={t('lobby.previewLabel', { rank })} />
}

/**
 * The candidate maps side by side (stacked on a phone), in the order given,
 * with the buttons to reorder them. The order is the caller's: this list only
 * reports every reordering.
 */
export function SeedRankingList({
  gameId,
  candidates,
  order,
  onOrderChange,
  tokenProvider,
  inviteCode,
}: {
  gameId: string
  candidates: SeedCandidate[]
  order: number[]
  onOrderChange: (order: number[]) => void
  tokenProvider: TokenProvider
  /** Candidates and previews of a game the caller has not joined yet. */
  inviteCode?: string
}) {
  const { t } = useLanguage()
  if (candidates.length === 0) {
    return <Alert tone="info">{t('lobby.fixedSeed')}</Alert>
  }
  const seedOf = (index: number) =>
    candidates.find((candidate) => candidate.index === index)?.seed ?? ''

  return (
    <section className="space-y-3" aria-labelledby="seed-vote-title">
      <div>
        <h2 id="seed-vote-title" className="font-display text-xl font-semibold text-ink-900">
          {t('lobby.seedTitle')}
        </h2>
        <p className="mt-1 text-xs text-ink-500">{t('lobby.seedDescription')}</p>
      </div>
      <ol className="grid gap-4 md:grid-cols-3">
        {order.map((index, position) => (
          <li
            key={index}
            data-seed-candidate={index}
            className="space-y-2 rounded-lg border border-line bg-paper-50 p-3"
          >
            <div className="flex items-center gap-2">
              <span className="font-display text-lg font-semibold text-ink-900">
                {t('lobby.candidate', { rank: position + 1 })}
              </span>
              <code className="min-w-0 flex-1 truncate text-xs text-ink-500">
                {seedOf(index)}
              </code>
              <Button
                type="button"
                variant="outline"
                size="sm"
                aria-label={t('lobby.moveUp', { rank: position + 1 })}
                disabled={position === 0}
                onClick={() => onOrderChange(moved(order, position, -1))}
              >
                <IconArrowUp aria-hidden="true" className="size-4" />
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                aria-label={t('lobby.moveDown', { rank: position + 1 })}
                disabled={position === order.length - 1}
                onClick={() => onOrderChange(moved(order, position, 1))}
              >
                <IconArrowDown aria-hidden="true" className="size-4" />
              </Button>
            </div>
            <CandidatePreview
              gameId={gameId}
              index={index}
              rank={position + 1}
              tokenProvider={tokenProvider}
              inviteCode={inviteCode}
            />
          </li>
        ))}
      </ol>
    </section>
  )
}

/** The ranking of a member of the lobby, saved with a button. */
export function SeedVote({
  gameId,
  lobby,
  tokenProvider,
  onVote,
}: {
  gameId: string
  lobby: LobbyData
  tokenProvider: TokenProvider
  onVote: (ranking: number[]) => Promise<void>
}) {
  const { t } = useLanguage()
  const saved = lobby.seedRanking.join(',')
  const syncKey = `${saved}|${lobby.seedCandidates.map((candidate) => candidate.index).join(',')}`
  const [order, setOrder] = useState(() => rankingOrder(lobby))
  const [syncedKey, setSyncedKey] = useState(syncKey)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // The ranking follows the saved vote, not every render of the lobby.
  if (syncedKey !== syncKey) {
    setSyncedKey(syncKey)
    setOrder(rankingOrder(lobby))
  }
  const dirty = order.join(',') !== saved

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      await onVote(order)
    } catch (failure) {
      setError(errorText(failure, t, t('error.serverUnavailable')))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <SeedRankingList
        gameId={gameId}
        candidates={lobby.seedCandidates}
        order={order}
        onOrderChange={setOrder}
        tokenProvider={tokenProvider}
      />
      {lobby.seedCandidates.length > 0 && (
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" disabled={saving || (!dirty && saved !== '')} onClick={() => void save()}>
            {saving ? t('lobby.savingVote') : t('lobby.saveVote')}
          </Button>
          <p className="text-sm text-ink-500" role="status">
            {saved !== '' && !dirty ? t('lobby.voteSaved') : saved === '' ? t('lobby.noVote') : ''}
          </p>
        </div>
      )}
      {error && <Alert role="alert">{error}</Alert>}
    </div>
  )
}

/** The lobby of an online game: who joined, the house name, the seed vote. */
export function GameLobby({
  gameId,
  tokenProvider,
  lobby,
  houseName,
  onRename,
  onVote,
}: {
  gameId: string
  tokenProvider: TokenProvider
  lobby: LobbyData
  houseName: string
  onRename: (name: string) => Promise<void>
  onVote: (ranking: number[]) => Promise<void>
}) {
  return (
    <div className="space-y-6">
      <HouseNameForm
        key={houseName}
        currentName={houseName}
        onRename={onRename}
      />
      <SeedVote gameId={gameId} lobby={lobby} tokenProvider={tokenProvider} onVote={onVote} />
    </div>
  )
}
