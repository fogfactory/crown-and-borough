import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, apiRequest } from '@/lib/api'
import { useLanguage } from '@/i18n/LanguageContext'
import type { Translate } from '@/i18n/messages'

interface JoinResponse {
  id?: string
  gameId?: string
}

function joinError(error: unknown, t: Translate): string {
  if (error instanceof ApiError) {
    if (error.code === 'invalid_invitation') return t('join.invalid')
    if (error.code === 'game_full') return t('join.full')
    if (error.code === 'not_found') return t('join.notFound')
    return error.message
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
  const started = useRef(false)
  const gameId = searchParams.get('gameId')?.trim() ?? ''
  const inviteCode = searchParams.get('inviteCode')?.trim() ?? ''

  useEffect(() => {
    if (started.current) return
    started.current = true
    if (!gameId || !inviteCode) {
      setJoining(false)
      setError(t('join.invalid'))
      return
    }

    void apiRequest<JoinResponse>(
      { getIdToken },
      `/api/games/${encodeURIComponent(gameId)}/join`,
      {
        method: 'POST',
        body: JSON.stringify({ inviteCode }),
      },
    )
      .then((response) => {
        const destination = response.id ?? response.gameId ?? gameId
        navigate(`/games/${encodeURIComponent(destination)}`, { replace: true })
      })
      .catch((joinFailure: unknown) => {
        if (joinFailure instanceof ApiError && joinFailure.status === 401) {
          void signOut().catch(() => undefined)
          const returnTo = `${location.pathname}${location.search}`
          navigate(`/signin?redirect=${encodeURIComponent(returnTo)}`, { replace: true })
          return
        }
        setJoining(false)
        setError(joinError(joinFailure, t))
      })
  }, [
    gameId,
    getIdToken,
    inviteCode,
    location.pathname,
    location.search,
    navigate,
    signOut,
    t,
  ])

  return (
    <Card className="mx-auto max-w-xl border-line bg-paper-50 shadow-panel">
      <CardHeader>
        <CardTitle className="font-display text-3xl text-ink-900">
          {t('join.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {joining ? (
          <p className="text-sm italic text-ink-500" role="status">
            {t('join.validating')}
          </p>
        ) : error ? (
          <>
            <p
              role="alert"
              className="rounded-lg border border-vermilion-600/30 bg-vermilion-100 px-3 py-3 text-sm text-vermilion-700"
            >
              {error}
            </p>
            <Button type="button" variant="outline" onClick={() => navigate('/')}>
              {t('online.backHome')}
            </Button>
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}
