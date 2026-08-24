#!/usr/bin/env -S npx tsx

import { spawnSync } from 'node:child_process'

import { applicationDefault, getApps, initializeApp } from 'firebase-admin/app'
import { getAuth } from 'firebase-admin/auth'

const USAGE = 'Usage: npm run grant -- <email> [grant|revoke]'

function failUsage(): never {
  process.stderr.write(`${USAGE}\n`)
  process.exit(2)
}

const args = process.argv.slice(2)
if (args.length < 1 || args.length > 2) failUsage()

const email = args[0]?.trim()
const action = (args[1] ?? 'grant').trim().toLowerCase()
if (!email || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) failUsage()
if (action !== 'grant' && action !== 'revoke') failUsage()

function resolveProjectId(): string {
  const fromFirebaseEnv = process.env.FIREBASE_PROJECT_ID?.trim()
  if (fromFirebaseEnv) return fromFirebaseEnv

  const fromGoogleEnv = process.env.GOOGLE_CLOUD_PROJECT?.trim()
  if (fromGoogleEnv) return fromGoogleEnv

  const result = spawnSync('gcloud', ['config', 'get-value', 'project'], {
    encoding: 'utf8',
  })
  const fromGcloud = result.stdout?.trim() ?? ''
  if (result.status === 0 && fromGcloud && fromGcloud !== '(unset)') return fromGcloud

  throw new Error(
    'FIREBASE_PROJECT_ID is not set and `gcloud config get-value project` returned nothing',
  )
}

async function main(): Promise<void> {
  const projectId = resolveProjectId()
  if (!getApps().length) {
    initializeApp({ credential: applicationDefault(), projectId })
  }

  const auth = getAuth()
  const user = await auth.getUserByEmail(email)
  const claims = { ...(user.customClaims ?? {}) }
  if (action === 'grant') claims.game_creator = true
  else delete claims.game_creator

  await auth.setCustomUserClaims(user.uid, claims)
  const updatedUser = await auth.getUser(user.uid)
  process.stdout.write(
    `${JSON.stringify(
      {
        email: updatedUser.email,
        uid: updatedUser.uid,
        action,
        customClaims: updatedUser.customClaims ?? {},
      },
      null,
      2,
    )}\n`,
  )
}

main().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : String(error)
  process.stderr.write(`failed: ${message}\n`)
  process.exit(1)
})
