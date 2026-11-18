import { useCallback, useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { orderCandidates, SeedRankingList } from '@/components/GameLobby'
import { HouseNameField } from '@/components/HouseNameField'
import { Seal } from '@/components/Seal'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, apiRequest } from '@/lib/api'
import { apiErrorMessage } from '@/lib/api-error'
import { useLanguage } from '@/i18n/LanguageContext'
import type { Translate } from '@/i18n/messages'
import type { JoinPreview } from '@/types'

interface JoinResponse {
  id?: string
  gameId?: string
}

function joinError(error: unknown, t: Translate): string {
  if (error instanceof ApiError) {
    if (error.code === 'invalid_invitation') return t('join.invalid')
    if (error.code === 'game_full') return t('join.full')
    if (error.code === 'not_found') return t('join.notFound')
    return apiErrorMessage(error, t)
  }
  return error instanceof Error ? error.message : t('join.invalid')
}

export function JoinPage() {
  const { getIdToken, signOut } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const [error, setError] = useState<string | null>(null)
  const [joining, setJoining] = useState(true)
  const [preview, setPreview] = useState<JoinPreview | null>(null)
  const [houseName, setHouseName] = useState('')
  const [ranking, setRanking] = useState<number[]>([])
  const started = useRef(false)
  const gameId = searchParams.get('gameId')?.trim() ?? ''
  const inviteCode = searchParams.get('inviteCode')?.trim() ?? ''

  const reportFailure = useCallback(
    (failure: unknown) => {
      if (failure instanceof ApiError && failure.status === 401) {
        void signOut().catch(() => undefined)
        const returnTo = `${location.pathname}${location.search}`
        navigate(`/signin?redirect=${encodeURIComponent(returnTo)}`, { replace: true })
        return
      }
      setJoining(false)
      setError(joinError(failure, t))
    },
    [location.pathname, location.search, navigate, signOut, t],
  )

  const join = useCallback(
    async (entry: { houseName?: string; seedRanking?: number[] }) => {
      setJoining(true)
      setError(null)
      try {
        const response = await apiRequest<JoinResponse>(
          { getIdToken },
          `/api/games/${encodeURIComponent(gameId)}/join`,
          { method: 'POST', body: JSON.stringify({ inviteCode, ...entry }) },
        )
        const destination = response.id ?? response.gameId ?? gameId
        navigate(`/games/${encodeURIComponent(destination)}`, { replace: true })
      } catch (failure) {
        reportFailure(failure)
      }
    },
    [gameId, getIdToken, inviteCode, navigate, reportFailure],
  )

  useEffect(() => {
    if (started.current) return
    started.current = true
    if (!gameId || !inviteCode) {
      setJoining(false)
      setError(t('join.invalid'))
      return
    }
    // A game still in its lobby lets the joiner prepare a house name and a
    // vote first; anything else is joined at once.
    void apiRequest<JoinPreview>(
      { getIdToken },
      `/api/games/${encodeURIComponent(gameId)}/join-preview?inviteCode=${encodeURIComponent(inviteCode)}`,
    )
      .then((response) => {
        if (response.started === false && response.member !== true) {
          setPreview(response)
          setJoining(false)
          return
        }
        return join({})
      })
      .catch(reportFailure)
  }, [gameId, getIdToken, inviteCode, join, reportFailure, t])

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    void join({
      ...(houseName.trim() !== '' ? { houseName: houseName.trim() } : {}),
      ...(ranking.length > 0 ? { seedRanking: ranking } : {}),
    })
  }

  return (
    <Card className="mx-auto max-w-5xl border-line">
      <CardHeader>
        <Seal variant="full" className="mx-auto mb-2 size-32" />
        <CardTitle className="font-display text-3xl font-bold text-ink-900">
          {t('join.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <>
            <Alert role="alert">{error}</Alert>
            <Button type="button" variant="outline" onClick={() => navigate('/')}>
              {t('online.backHome')}
            </Button>
          </>
        )}
        {!error && joining && (
          <p className="text-sm italic text-ink-500" role="status">
            {t('join.validating')}
          </p>
        )}
        {!error && !joining && preview && (
          <form className="space-y-6" onSubmit={submit}>
            <div>
              <h2 className="font-display text-xl font-semibold text-ink-900">{preview.name}</h2>
              <p className="text-sm text-ink-500">
                {t('online.waitingForPlayers', {
                  seated: preview.seated,
                  total: preview.players,
                })}
              </p>
              <p className="mt-1 text-xs text-ink-500">{t('join.prepare')}</p>
            </div>
            <div className="space-y-1.5">
              <HouseNameField value={houseName} onChange={setHouseName} />
            </div>
            <SeedRankingList
              gameId={gameId}
              inviteCode={inviteCode}
              tokenProvider={{ getIdToken }}
              candidates={preview.seedCandidates}
              order={orderCandidates(preview.seedCandidates, ranking)}
              onOrderChange={setRanking}
            />
            {preview.seedCandidates.length > 0 && (
              <p className="text-sm text-ink-500">{t('lobby.abstainHint')}</p>
            )}
            <Button type="submit" className="w-full">
              {t('join.submit')}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
