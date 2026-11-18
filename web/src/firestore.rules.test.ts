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
  deleteDoc,
  doc,
  getDoc,
  getDocs,
  orderBy,
  query,
  setDoc,
  updateDoc,
  where,
} from 'firebase/firestore'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const rules = fs.readFileSync(path.join(projectRoot, 'firestore.rules'), 'utf8')
const emulatorConfigured = Boolean(process.env.FIRESTORE_EMULATOR_HOST)

type RulesFirestore = ReturnType<
  ReturnType<RulesTestEnvironment['authenticatedContext']>['firestore']
>

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
        updatedAt: new Date('2026-01-01T12:00:00Z'),
      })
      await setDoc(doc(db, 'games/game-2'), {
        memberUids: ['carol', 'dan'],
        ownerUid: 'carol',
        schemaVersion: 1,
        updatedAt: new Date('2026-01-02T12:00:00Z'),
      })
      await setDoc(doc(db, 'games/game-3'), {
        memberUids: ['alice', 'dan'],
        ownerUid: 'dan',
        schemaVersion: 1,
        updatedAt: new Date('2026-01-03T12:00:00Z'),
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
      await setDoc(doc(db, 'games/game-1/lobby/bob'), { uid: 'bob', seedRanking: [1, 0, 2] })
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
    // A seed vote stays private, even to its author: only the API reads it.
    await assertFails(getDoc(doc(alice, 'games/game-1/lobby/bob')))
    await assertFails(
      setDoc(doc(alice, 'games/game-1/lobby/alice'), { uid: 'alice', seedRanking: [0] }),
    )
  })

  it('serves the game list query only for the caller’s own memberships', async () => {
    const games = (db: RulesFirestore, uid: string) =>
      query(
        collection(db, 'games'),
        where('memberUids', 'array-contains', uid),
        orderBy('updatedAt', 'desc'),
      )
    const alice = environment.authenticatedContext('alice').firestore()
    const snapshot = await assertSucceeds(getDocs(games(alice, 'alice')))
    expect(snapshot.docs.map((entry) => entry.id)).toEqual(['game-3', 'game-1'])
    // A spectator host is a member too, and an account without any game gets
    // an empty list rather than an error.
    const host = environment.authenticatedContext('host').firestore()
    const hosted = await assertSucceeds(getDocs(games(host, 'host')))
    expect(hosted.docs.map((entry) => entry.id)).toEqual(['game-1'])
    const bob = environment.authenticatedContext('bob').firestore()
    const none = await assertSucceeds(getDocs(games(bob, 'bob')))
    expect(none.empty).toBe(true)
    // Nobody may list another account's games, nor the whole collection.
    await assertFails(getDocs(games(bob, 'alice')))
    await assertFails(getDocs(collection(alice, 'games')))
    const unauthenticated = environment.unauthenticatedContext().firestore()
    await assertFails(getDocs(games(unauthenticated, 'alice')))
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
    const history = (db: RulesFirestore) =>
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

  it('denies every operation on backend-only documents to every kind of caller', async () => {
    const callers: Record<string, RulesFirestore> = {
      unauthenticated: environment.unauthenticatedContext().firestore(),
      member: environment.authenticatedContext('alice').firestore(),
      wrongMember: environment.authenticatedContext('carol').firestore(),
      nonMember: environment.authenticatedContext('mallory').firestore(),
      spectator: environment.authenticatedContext('host').firestore(),
    }
    const paths = [
      'games/game-1/canonical/current',
      'games/game-1/turns/1/submissions/alice',
      'games/game-1/reports/1',
      'games/game-1/chat_state/alice',
      'games/game-1/notify_state/alice',
      'invitations/hash',
      'bugReports/some-report',
      'player_activity/alice',
      'unknown/anything',
    ]
    for (const db of Object.values(callers)) {
      for (const p of paths) {
        await assertFails(getDoc(doc(db, p)))
        await assertFails(setDoc(doc(db, p), { forged: true }))
        await assertFails(updateDoc(doc(db, p), { forged: true }))
        await assertFails(deleteDoc(doc(db, p)))
      }
    }
    // Collection-level lists are denied too.
    for (const db of Object.values(callers)) {
      await assertFails(getDocs(collection(db, 'invitations')))
      await assertFails(getDocs(collection(db, 'games/game-1/turns/1/submissions')))
      await assertFails(getDocs(collection(db, 'games/game-1/reports')))
      await assertFails(getDocs(collection(db, 'games/game-1/canonical')))
    }
  })

  it('denies create, update and delete of member-readable documents', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const carol = environment.authenticatedContext('carol').firestore()
    const dan = environment.authenticatedContext('dan').firestore()
    const host = environment.authenticatedContext('host').firestore()
    const existing: [RulesFirestore, string][] = [
      [alice, 'games/game-1'],
      [alice, 'games/game-1/views/alice'],
      [alice, 'games/game-1/reports/alice/turns/1'],
      [alice, 'games/game-1/chat_public/m1'],
      [alice, 'games/game-1/chat_reads/alice'],
      [alice, 'players/alice'],
      [host, 'games/game-1/observer/host'],
    ]
    for (const [db, p] of existing) {
      await assertFails(updateDoc(doc(db, p), { forged: true }))
      await assertFails(deleteDoc(doc(db, p)))
    }
    // A player cannot forge the document that would be theirs.
    const created: [RulesFirestore, string, object][] = [
      [carol, 'games/game-9', { memberUids: ['carol'], ownerUid: 'carol' }],
      [carol, 'games/game-1/views/carol', { uid: 'carol' }],
      [carol, 'games/game-1/reports/carol/turns/1', { uid: 'carol' }],
      [carol, 'games/game-1/chat_reads/carol', { uid: 'carol', channels: {} }],
      [carol, 'players/carol', { uid: 'carol' }],
      [dan, 'games/game-1/observer/dan', { uid: 'dan' }],
    ]
    for (const [db, p, data] of created) {
      await assertFails(setDoc(doc(db, p), data))
    }
  })

  it('isolates private projections between members and between games', async () => {
    const alice = environment.authenticatedContext('alice').firestore()
    const carol = environment.authenticatedContext('carol').firestore()
    const outsider = environment.authenticatedContext('mallory').firestore()
    const unauthenticated = environment.unauthenticatedContext().firestore()
    // A member reads neither another member's view nor their report, nor the
    // observer projection (which is for the spectator only).
    await assertFails(getDoc(doc(carol, 'games/game-1/views/alice')))
    await assertFails(getDoc(doc(carol, 'games/game-1/reports/alice/turns/1')))
    await assertFails(getDoc(doc(alice, 'games/game-1/observer/host')))
    await assertFails(getDocs(collection(carol, 'games/game-1/reports/alice/turns')))
    // Membership of one game grants nothing in another: alice is not in game-2.
    await assertFails(getDoc(doc(alice, 'games/game-2')))
    await assertFails(getDoc(doc(alice, 'games/game-2/chat_public/m1')))
    await assertFails(getDoc(doc(alice, 'games/game-2/views/alice')))
    // Non-members and anonymous callers read no projection at all.
    for (const db of [outsider, unauthenticated]) {
      await assertFails(getDoc(doc(db, 'games/game-1/views/alice')))
      await assertFails(getDoc(doc(db, 'games/game-1/observer/host')))
      await assertFails(getDoc(doc(db, 'games/game-1/reports/alice/turns/1')))
      await assertFails(getDoc(doc(db, 'games/game-1/chat_reads/alice')))
      await assertFails(getDoc(doc(db, 'games/game-1/chat_private/P1_P2/messages/m2')))
      await assertFails(getDocs(collection(db, 'games/game-1/chat_public')))
    }
  })

  it('documents how many access calls each rule can trigger', () => {
    // The emulator does not report auxiliary reads, so they are counted
    // statically from the deployed rules: every get/getAfter/exists/existsAfter
    // is an access call, whether written in an allow condition or reached
    // through a helper such as memberOf. Firestore allows 10 access calls per
    // single-document request (20 for multi-document ones) and bills one read
    // per distinct document. The report is informational; only that platform
    // limit can fail the test.
    const accessCalls = (text: string) =>
      (text.match(/\b(?:get|getAfter|exists|existsAfter)\(\/databases/g) ?? []).length
    const helperCalls: Record<string, number> = {}
    for (const match of rules.matchAll(/function (\w+)\([^)]*\) \{([\s\S]*?)\n {4}\}/g)) {
      helperCalls[match[1]] = accessCalls(match[2])
    }
    expect(Object.keys(helperCalls)).toContain('memberOf')
    const perAllow: Record<string, number> = {}
    for (const match of rules.matchAll(/allow ([\w, ]+): if ([\s\S]*?);/g)) {
      const condition = match[2]
      const helpers = Object.entries(helperCalls).reduce(
        (sum, [name, calls]) =>
          sum + (condition.match(new RegExp(`\\b${name}\\(`, 'g')) ?? []).length * calls,
        0,
      )
      const key = `${match[1]}: ${condition.replace(/\s+/g, ' ').slice(0, 60)}`
      perAllow[key] = accessCalls(condition) + helpers
    }
    console.info('Firestore rules: access calls per helper', helperCalls)
    console.info('Firestore rules: access calls per allow condition', perAllow)
    expect(Math.max(0, ...Object.values(perAllow))).toBeLessThanOrEqual(10)
  })

  it('keeps the rules test explicit when no emulator is configured', () => {
    expect(emulatorConfigured).toBe(true)
  })
})
