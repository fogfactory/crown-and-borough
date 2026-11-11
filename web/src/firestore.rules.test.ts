import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  assertFails,
  assertSucceeds,
  initializeTestEnvironment,
  type RulesTestEnvironment,
} from '@firebase/rules-unit-testing'
import {
  collection,
  doc,
  getDoc,
  getDocs,
  orderBy,
  query,
  setDoc,
  where,
} from 'firebase/firestore'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const rules = fs.readFileSync(path.join(projectRoot, 'firestore.rules'), 'utf8')
const emulatorConfigured = Boolean(process.env.FIRESTORE_EMULATOR_HOST)

const suite = emulatorConfigured ? describe : describe.skip

suite('Firestore security rules', () => {
  let environment: RulesTestEnvironment

  beforeAll(async () => {
    environment = await initializeTestEnvironment({
      projectId: 'crown-and-borough-rules-test',
      firestore: { rules },
    })
    await environment.withSecurityRulesDisabled(async (context) => {
      const db = context.firestore()
      await setDoc(doc(db, 'games/game-1'), {
        memberUids: ['alice', 'carol', 'dan', 'host'],
        ownerUid: 'alice',
        spectatorUid: 'host',
        schemaVersion: 1,
      })
      await setDoc(doc(db, 'games/game-1/chat_public/m1'), {
        channel: 'public',
        authorHouseId: 'P1',
        body: 'hello',
        createdAt: new Date('2026-01-01T12:00:00Z'),
      })
      await setDoc(doc(db, 'games/game-1/chat_private/P1_P2/messages/m2'), {
        channel: 'P1_P2',
        authorHouseId: 'P1',
        body: 'secret',
        createdAt: new Date('2026-01-01T12:00:01Z'),
        participantUids: ['alice', 'carol'],
      })
      await setDoc(doc(db, 'games/game-1/chat_private/P1_P3/messages/m3'), {
        channel: 'P1_P3',
        authorHouseId: 'P1',
        body: 'other secret',
        createdAt: new Date('2026-01-01T12:00:02Z'),
        participantUids: ['alice', 'dan'],
      })
      await setDoc(doc(db, 'games/game-1/chat_private/P2_P3/messages/m4'), {
        channel: 'P2_P3',
        authorHouseId: 'P2',
        body: 'between others',
        createdAt: new Date('2026-01-01T12:00:03Z'),
        participantUids: ['carol', 'dan'],
      })
      await setDoc(doc(db, 'games/game-1/chat_reads/alice'), {
        uid: 'alice',
        channels: {},
      })
      await setDoc(doc(db, 'games/game-1/chat_state/alice'), { lastSentAt: new Date() })
      await setDoc(doc(db, 'games/game-1/notify_state/alice'), {
        messageEmailAt: new Date(),
      })
      await setDoc(doc(db, 'player_activity/alice'), { lastActiveAt: new Date() })
      await setDoc(doc(db, 'players/alice'), { uid: 'alice', displayName: 'Alice' })
      await setDoc(doc(db, 'games/game-1/views/alice'), { uid: 'alice', revision: 1 })
      await setDoc(doc(db, 'games/game-1/views/bob'), { uid: 'bob', revision: 1 })
      await setDoc(doc(db, 'games/game-1/observer/host'), { uid: 'host', revision: 1 })
      await setDoc(doc(db, 'games/game-1/reports/alice/turns/1'), {
        uid: 'alice',
        turn: 1,
      })
      await setDoc(doc(db, 'games/game-1/canonical/current'), { state: {} })
      await setDoc(doc(db, 'games/game-1/turns/1/submissions/alice'), { uid: 'alice' })
      await setDoc(doc(db, 'games/game-1/reports/1'), { turn: 1 })
      await setDoc(doc(db, 'invitations/hash'), { gameId: 'game-1' })
    })
  })

  afterAll(async () => {
    await environment.cleanup()
  })

  it('allows a member to read only the public summary and own projections', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    await assertSucceeds(getDoc(doc(alice, 'games/game-1')))
    await assertSucceeds(getDoc(doc(alice, 'games/game-1/views/alice')))
    await assertSucceeds(getDoc(doc(alice, 'games/game-1/reports/alice/turns/1')))
    await assertFails(getDoc(doc(alice, 'games/game-1/views/bob')))
    await assertFails(getDoc(doc(alice, 'games/game-1/reports/bob/turns/1')))
    await assertFails(getDoc(doc(alice, 'games/game-1/canonical/current')))
  })

  it('allows the spectator host to read only the observer projection', async () => {
    const host = environment.authenticatedContext('host').firestore()
    await assertSucceeds(getDoc(doc(host, 'games/game-1')))
    await assertSucceeds(getDoc(doc(host, 'games/game-1/observer/host')))
    await assertFails(getDoc(doc(host, 'games/game-1/observer/alice')))
  })

  it('denies non-members, writes, raw reports, submissions, and invitations', async () => {
    const bob = environment.authenticatedContext('bob').firestore()
    await assertFails(getDoc(doc(bob, 'games/game-1')))
    await assertFails(getDoc(doc(bob, 'games/game-1/reports/alice/turns/1')))

    const alice = environment.authenticatedContext('alice').firestore()
    await assertFails(setDoc(doc(alice, 'games/game-1'), { memberUids: ['bob'] }))
    await assertFails(setDoc(doc(alice, 'games/game-1/views/alice'), { revision: 2 }))
    await assertFails(
      setDoc(doc(alice, 'games/game-1/reports/alice/turns/1'), { turn: 2 }),
    )
    await assertFails(
      setDoc(doc(alice, 'games/game-1/canonical/current'), { state: { forged: true } }),
    )
    await assertFails(
      setDoc(doc(alice, 'games/game-1/turns/1/submissions/alice'), { uid: 'alice' }),
    )
    await assertFails(setDoc(doc(alice, 'games/game-1/reports/1'), { turn: 2 }))
    await assertFails(setDoc(doc(alice, 'invitations/hash'), { gameId: 'game-1' }))
    await assertFails(getDoc(doc(alice, 'games/game-1/reports/1')))
    await assertFails(getDoc(doc(alice, 'games/game-1/turns/1/submissions/alice')))
    await assertFails(getDoc(doc(alice, 'invitations/hash')))
    await assertFails(getDoc(doc(alice, 'bugReports/game-t1-alice-abc123')))
    await assertFails(
      setDoc(doc(alice, 'bugReports/game-t1-alice-abc123'), { gameViewJson: '{}' }),
    )
  })

  it('lets every member read the public chat room but nobody write it', async () => {
    const carol = environment.authenticatedContext('carol').firestore()
    const outsider = environment.authenticatedContext('mallory').firestore()
    await assertSucceeds(getDoc(doc(carol, 'games/game-1/chat_public/m1')))
    await assertSucceeds(getDocs(collection(carol, 'games/game-1/chat_public')))
    await assertFails(getDoc(doc(outsider, 'games/game-1/chat_public/m1')))
    await assertFails(
      setDoc(doc(carol, 'games/game-1/chat_public/forged'), {
        authorHouseId: 'P1',
        body: 'forged',
      }),
    )
  })

  it('limits a private conversation to its participants and the observer', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const carol = environment.authenticatedContext('carol').firestore()
    const dan = environment.authenticatedContext('dan').firestore()
    const host = environment.authenticatedContext('host').firestore()
    const outsider = environment.authenticatedContext('mallory').firestore()
    const secret = 'games/game-1/chat_private/P1_P2/messages/m2'
    await assertSucceeds(getDoc(doc(alice, secret)))
    await assertSucceeds(getDoc(doc(carol, secret)))
    await assertSucceeds(getDoc(doc(host, secret)))
    await assertFails(getDoc(doc(dan, secret)))
    await assertFails(getDoc(doc(outsider, secret)))
  })

  it('gives the game creator who plays no more than any player', async () => {
    // alice created the game and plays P1: she reads her own conversations only.
    const alice = environment.authenticatedContext('alice').firestore()
    const others = 'games/game-1/chat_private/P2_P3/messages'
    await assertFails(getDoc(doc(alice, `${others}/m4`)))
    await assertFails(getDocs(collection(alice, others)))
    const host = environment.authenticatedContext('host').firestore()
    await assertSucceeds(getDoc(doc(host, `${others}/m4`)))
  })

  it('serves the paginated private conversation query only to a participant', async () => {
    const messages = 'games/game-1/chat_private/P1_P2/messages'
    const history = (
      db: ReturnType<
        ReturnType<RulesTestEnvironment['authenticatedContext']>['firestore']
      >,
    ) =>
      query(
        collection(db, messages),
        where('participantUids', 'array-contains', 'carol'),
        orderBy('createdAt', 'desc'),
      )
    const carol = environment.authenticatedContext('carol').firestore()
    const dan = environment.authenticatedContext('dan').firestore()
    const snapshot = await assertSucceeds(getDocs(history(carol)))
    expect(snapshot.docs.map((entry) => entry.data().body)).toEqual(['secret'])
    // Cleo cannot claim to be a participant, nor list the conversation bare.
    await assertFails(getDocs(history(dan)))
    await assertFails(getDocs(collection(dan, messages)))
    // The observer lists every conversation without any filter.
    const host = environment.authenticatedContext('host').firestore()
    await assertSucceeds(getDocs(collection(host, messages)))
  })

  it('denies every client write to private messages, cursors, and rate-limit state', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const host = environment.authenticatedContext('host').firestore()
    await assertFails(
      setDoc(doc(alice, 'games/game-1/chat_private/P1_P2/messages/forged'), {
        participantUids: ['alice', 'carol'],
        body: 'forged',
      }),
    )
    await assertFails(
      setDoc(doc(host, 'games/game-1/chat_private/P1_P2/messages/m2'), {
        body: 'edited',
      }),
    )
    await assertFails(
      setDoc(doc(alice, 'games/game-1/chat_reads/alice'), { channels: {} }),
    )
    await assertFails(getDoc(doc(alice, 'games/game-1/chat_state/alice')))
    await assertFails(
      setDoc(doc(alice, 'games/game-1/chat_state/alice'), { lastSentAt: new Date(0) }),
    )
  })

  it('keeps the email notification state backend-only', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    await assertFails(getDoc(doc(alice, 'games/game-1/notify_state/alice')))
    await assertFails(
      setDoc(doc(alice, 'games/game-1/notify_state/alice'), {
        messageEmailAt: new Date(0),
      }),
    )
    await assertFails(getDoc(doc(alice, 'player_activity/alice')))
    await assertFails(
      setDoc(doc(alice, 'player_activity/alice'), { lastActiveAt: new Date() }),
    )
  })

  it('lets a player read only their own chat read cursors', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const carol = environment.authenticatedContext('carol').firestore()
    await assertSucceeds(getDoc(doc(alice, 'games/game-1/chat_reads/alice')))
    await assertFails(getDoc(doc(carol, 'games/game-1/chat_reads/alice')))
  })

  it('restricts profiles to the authenticated uid', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const bob = environment.authenticatedContext('bob').firestore()
    await assertSucceeds(getDoc(doc(alice, 'players/alice')))
    await assertFails(getDoc(doc(alice, 'players/bob')))
    await assertFails(
      setDoc(doc(alice, 'players/alice'), { uid: 'alice', displayName: 'Forged' }),
    )
    await assertFails(getDoc(doc(bob, 'players/alice')))
  })

  it('denies unauthenticated access', async () => {
    const unauthenticated = environment.unauthenticatedContext().firestore()
    await assertFails(getDoc(doc(unauthenticated, 'games/game-1')))
    await assertFails(getDoc(doc(unauthenticated, 'players/alice')))
  })

  it('keeps the rules test explicit when no emulator is configured', () => {
    expect(emulatorConfigured).toBe(true)
  })
})
