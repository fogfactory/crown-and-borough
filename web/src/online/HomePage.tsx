import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ApiError, apiRequest } from '@/lib/api'
import { normalizeGameSummary, useGameListSubscription } from '@/lib/subscription'
import { useLanguage } from '@/i18n/LanguageContext'
import type { GameSummary, Season } from '@/types'

interface Invitation {
  gameId: string
  inviteCode: string
  inviteUrl: string
}

const seasonLabels: Record<Season, string> = {
  spring: 'Spring',
  summer: 'Summer',
  autumn: 'Autumn',
  winter: 'Winter',
}

const fallbackSeed = 'adelaide-de-beaufort'

function actionError(error: unknown, fallback: string): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return fallback
}

async function copyText(value: string): Promise<boolean> {
  if (!navigator.clipboard) return false
  await navigator.clipboard.writeText(value)
  return true
}

function GameCard({ game }: { game: GameSummary }) {
  const { t } = useLanguage()
  return (
    <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="font-serif text-xl text-[#30291f]">
              {game.name}
            </CardTitle>
            <CardDescription className="mt-1 text-[#806f57]">
              {t('online.currentTurn', {
                turn: game.turn,
                season: seasonLabels[game.season],
              })}
            </CardDescription>
            <p className="mt-1 font-mono text-xs text-[#806f57]">
              {t('home.seed')}: {game.seed || '-'}
            </p>
          </div>
          <span className="rounded-full border border-[#376341]/30 bg-[#e8f1e3] px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-[#376341]">
            {game.status === 'finished'
              ? t('home.statusFinished')
              : t('home.statusPlaying')}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <ul className="grid gap-2 sm:grid-cols-2" aria-label={t('online.lobby')}>
          {game.players.map((player) => (
            <li
              key={player.id}
              className="flex items-center gap-2 rounded-md bg-[#f3ead9] px-3 py-2 text-sm"
            >
              <span
                aria-hidden="true"
                className="size-3 shrink-0 rounded-full border border-[#30291f]/30"
                style={{ backgroundColor: player.color }}
              />
              <span className="min-w-0 flex-1 truncate">
                {player.name || t('online.emptySlot')}
              </span>
              <span className="shrink-0 text-[10px] uppercase tracking-[0.08em] text-[#806f57]">
                {player.submitted ? t('home.submitted') : t('home.notSubmitted')}
              </span>
            </li>
          ))}
        </ul>
        {game.winner && (
          <p className="rounded-md border border-[#815f1e]/40 bg-[#f8e8ae]/60 px-3 py-2 text-sm font-semibold text-[#6d5118]">
            {t('online.victory')}:{' '}
            {game.players.find((player) => player.id === game.winner)?.name ??
              game.winner}
          </p>
        )}
        <Link to={`/games/${encodeURIComponent(game.id)}`}>
          <Button type="button" className="w-full">
            {t('home.openGame')}
          </Button>
        </Link>
      </CardContent>
    </Card>
  )
}

function CreateGameForm({ onCreated }: { onCreated: (invitation: Invitation) => void }) {
  const { getIdToken } = useAuth()
  const { t } = useLanguage()
  const [name, setName] = useState('')
  const [seed, setSeed] = useState(fallbackSeed)
  const [players, setPlayers] = useState(4)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    void apiRequest<{ seed?: string }>({ getIdToken }, '/api/seed')
      .then((response) => {
        if (active && typeof response.seed === 'string' && response.seed.trim() !== '') {
          setSeed(response.seed)
        }
      })
      .catch(() => undefined)
    return () => {
      active = false
    }
  }, [getIdToken])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setCreating(true)
    setError(null)
    try {
      const response = await apiRequest<Invitation & Record<string, unknown>>(
        { getIdToken },
        '/api/games',
        {
          method: 'POST',
          body: JSON.stringify({ name: name.trim(), seed: seed.trim(), players }),
        },
      )
      onCreated({
        gameId: response.id as string,
        inviteCode: response.inviteCode as string,
        inviteUrl: response.inviteUrl as string,
      })
    } catch (createError) {
      if (createError instanceof ApiError && createError.code === 'creator_not_allowed') {
        setError(t('error.creatorNotAllowed'))
      } else {
        setError(actionError(createError, t('error.gameCreationFailed')))
      }
    } finally {
      setCreating(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
      <label className="block space-y-1.5">
        <span className="text-sm font-semibold text-[#594b3c]">{t('home.gameName')}</span>
        <input
          type="text"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder={t('home.gameNamePlaceholder')}
          className="h-10 w-full rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
        />
      </label>
      <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
        <label className="block space-y-1.5">
          <span className="text-sm font-semibold text-[#594b3c]">{t('home.seed')}</span>
          <input
            type="text"
            value={seed}
            onChange={(event) => setSeed(event.target.value)}
            placeholder={t('home.seedPlaceholder')}
            className="h-10 w-full rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
          />
        </label>
        <label className="block space-y-1.5">
          <span className="text-sm font-semibold text-[#594b3c]">
            {t('home.playerCount')}
          </span>
          <select
            value={players}
            onChange={(event) => setPlayers(Number(event.target.value))}
            className="h-10 rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm outline-none focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
          >
            {Array.from({ length: 7 }, (_, index) => index + 2).map((count) => (
              <option key={count} value={count}>
                {count}
              </option>
            ))}
          </select>
        </label>
      </div>
      {error && (
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
        >
          {error}
        </p>
      )}
      <Button type="submit" className="w-full" disabled={creating}>
        {creating ? t('app.creating') : t('home.createSubmit')}
      </Button>
    </form>
  )
}

function InvitationCard({ invitation }: { invitation: Invitation }) {
  const { t } = useLanguage()
  const [copied, setCopied] = useState<string | null>(null)
  const copy = async (kind: 'code' | 'url') => {
    const value = kind === 'code' ? invitation.inviteCode : invitation.inviteUrl
    if (await copyText(value)) setCopied(kind)
  }

  return (
    <Card className="border-[#815f1e]/50 bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
      <CardHeader>
        <CardTitle className="font-serif text-2xl text-[#30291f]">
          {t('home.invitationTitle')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#806f57]">
            {t('home.invitationCode')}
          </p>
          <div className="mt-1 flex items-center gap-2">
            <code className="flex-1 rounded-md bg-[#f3ead9] px-3 py-2 font-mono text-lg tracking-[0.2em]">
              {invitation.inviteCode}
            </code>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void copy('code')}
            >
              {copied === 'code' ? t('home.copied') : t('home.copy')}
            </Button>
          </div>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#806f57]">
            {t('home.invitationLink')}
          </p>
          <div className="mt-1 flex items-center gap-2">
            <input
              readOnly
              value={invitation.inviteUrl}
              aria-label={t('home.invitationLink')}
              className="min-w-0 flex-1 rounded-md border border-[#b7a786] bg-[#f8f0e2] px-3 py-2 text-xs"
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void copy('url')}
            >
              {copied === 'url' ? t('home.copied') : t('home.copy')}
            </Button>
          </div>
        </div>
        <Link to={`/games/${encodeURIComponent(invitation.gameId)}`}>
          <Button type="button" className="w-full">
            {t('home.openGame')}
          </Button>
        </Link>
      </CardContent>
    </Card>
  )
}

export function HomePage() {
  const { user, getIdToken, signOut } = useAuth()
  const { t } = useLanguage()
  const navigate = useNavigate()
  const realtime = useGameListSubscription(user?.uid)
  const [initialGames, setInitialGames] = useState<GameSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [invitation, setInvitation] = useState<Invitation | null>(null)

  useEffect(() => {
    let active = true
    void apiRequest<unknown[]>({ getIdToken }, '/api/games')
      .then((payload) => {
        if (!active) return
        setInitialGames(
          Array.isArray(payload)
            ? payload.map((game) => normalizeGameSummary(game as Record<string, unknown>))
            : [],
        )
        setLoading(false)
      })
      .catch((loadError: unknown) => {
        if (!active) return
        if (loadError instanceof ApiError && loadError.status === 401) {
          void signOut().catch(() => undefined)
          navigate('/signin', { replace: true })
          return
        }
        setError(actionError(loadError, t('error.serverUnavailable')))
        setLoading(false)
      })
    return () => {
      active = false
    }
  }, [getIdToken, navigate, signOut, t])

  const games =
    realtime.enabled && !realtime.loading && !realtime.error
      ? realtime.games
      : initialGames
  const combinedError = error ?? (realtime.error ? t('online.realtimeError') : null)

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[#a84632]">
            Crown &amp; Borough
          </p>
          <h1 className="mt-1 font-serif text-4xl font-semibold text-[#30291f]">
            {t('home.title')}
          </h1>
        </div>
        <Button
          type="button"
          onClick={() => {
            setInvitation(null)
            setShowCreate((current) => !current)
          }}
        >
          {t('home.create')}
        </Button>
      </div>

      {showCreate && (
        <Card className="border-[#b7a786] bg-[#fffaf0]">
          <CardHeader>
            <CardTitle className="font-serif text-2xl">{t('home.create')}</CardTitle>
          </CardHeader>
          <CardContent>
            {invitation ? (
              <InvitationCard invitation={invitation} />
            ) : (
              <CreateGameForm
                onCreated={(created) => {
                  setInvitation(created)
                  setShowCreate(true)
                }}
              />
            )}
          </CardContent>
        </Card>
      )}

      {combinedError && (
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-2 text-sm text-[#8d321e]"
        >
          {combinedError}
        </p>
      )}
      {loading && games.length === 0 ? (
        <p className="py-16 text-center font-serif text-lg italic text-[#806f57]">
          {t('online.loading')}
        </p>
      ) : games.length === 0 ? (
        <div className="rounded-xl border border-dashed border-[#b7a786] bg-[#f8f0e2] px-6 py-16 text-center">
          <p className="font-serif text-lg italic text-[#806f57]">{t('home.empty')}</p>
        </div>
      ) : (
        <section
          className="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
          aria-label={t('home.title')}
        >
          {games.map((game) => (
            <GameCard key={game.id} game={game} />
          ))}
        </section>
      )}
    </div>
  )
}
