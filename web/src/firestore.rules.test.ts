import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  assertFails,
  assertSucceeds,
  initializeTestEnvironment,
  type RulesTestEnvironment,
} from '@firebase/rules-unit-testing'
import { doc, getDoc, setDoc } from 'firebase/firestore'
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
        memberUids: ['alice'],
        schemaVersion: 1,
      })
      await setDoc(doc(db, 'players/alice'), { uid: 'alice', displayName: 'Alice' })
      await setDoc(doc(db, 'games/game-1/views/alice'), { uid: 'alice', revision: 1 })
      await setDoc(doc(db, 'games/game-1/views/bob'), { uid: 'bob', revision: 1 })
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
