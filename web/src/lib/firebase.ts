import { getApp, getApps, initializeApp, type FirebaseApp } from 'firebase/app'
import { connectAuthEmulator, getAuth, type Auth } from 'firebase/auth'
import {
  connectFirestoreEmulator,
  getFirestore,
  type Firestore,
} from 'firebase/firestore'

export interface FirebaseServices {
  app: FirebaseApp
  auth: Auth
  firestore: Firestore
}

const config = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
}

export const firebaseConfigured = Object.values(config).every(
  (value) => typeof value === 'string' && value.trim() !== '',
)

let services: FirebaseServices | null = null

function parseEmulatorHost(value: string | undefined, defaultPort: number) {
  const source = value?.trim()
  if (!source) return null

  const withProtocol = source.includes('://') ? source : `http://${source}`
  try {
    const url = new URL(withProtocol)
    return {
      host: url.hostname,
      port: Number(url.port) || defaultPort,
    }
  } catch {
    return null
  }
}

export function getFirebaseServices(): FirebaseServices | null {
  if (!firebaseConfigured) return null
  if (services) return services

  const app = getApps().length > 0 ? getApp() : initializeApp(config)
  const auth = getAuth(app)
  const firestore = getFirestore(app)

  const authEmulator = parseEmulatorHost(
    import.meta.env.VITE_FIREBASE_AUTH_EMULATOR_HOST,
    9099,
  )
  if (authEmulator) {
    connectAuthEmulator(auth, `http://${authEmulator.host}:${authEmulator.port}`, {
      disableWarnings: true,
    })
  }

  const firestoreEmulator = parseEmulatorHost(
    import.meta.env.VITE_FIREBASE_FIRESTORE_EMULATOR_HOST,
    8081,
  )
  if (firestoreEmulator) {
    connectFirestoreEmulator(firestore, firestoreEmulator.host, firestoreEmulator.port)
  }

  services = { app, auth, firestore }
  return services
}
