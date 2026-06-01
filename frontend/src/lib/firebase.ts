// Firebase initialization — guarded so the app works without Firebase env vars
// and without the `firebase` npm package installed.
//
// When Firebase is not configured or the package is absent every exported
// helper returns null and callers fall back to their REST-polling strategy.
//
// To enable Firebase:
//   1. pnpm add firebase
//   2. Fill in NEXT_PUBLIC_FIREBASE_* vars in .env.local (see .env.local.example)
//
// NOTE: dynamic imports use /* @vite-ignore */ / eslint-disable comments so
// that tsc does NOT require the `firebase` types to be present.

export interface FirebaseConfig {
  apiKey: string;
  authDomain: string;
  projectId: string;
  messagingSenderId: string;
  appId: string;
}

function getFirebaseConfig(): FirebaseConfig | null {
  const apiKey = process.env.NEXT_PUBLIC_FIREBASE_API_KEY;
  const authDomain = process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN;
  const projectId = process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID;
  const messagingSenderId = process.env.NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID;
  const appId = process.env.NEXT_PUBLIC_FIREBASE_APP_ID;

  if (!apiKey || !authDomain || !projectId || !messagingSenderId || !appId) {
    return null;
  }
  return { apiKey, authDomain, projectId, messagingSenderId, appId };
}

export const firebaseConfig = getFirebaseConfig();

/**
 * Returns a Firestore instance if Firebase is configured and the package is
 * installed, otherwise returns null.
 *
 * Uses indirect string-based dynamic import so tsc does not require the
 * firebase types when the package is absent.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function getFirestoreDb(): Promise<any | null> {
  if (!firebaseConfig) return null;
  try {
    // Indirect import key prevents tsc from resolving the module at compile time.
    const firebaseAppPkg = 'firebase/app';
    const firestorePkg = 'firebase/firestore';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { initializeApp, getApps, getApp } = await import(/* @vite-ignore */ firebaseAppPkg) as any;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { getFirestore } = await import(/* @vite-ignore */ firestorePkg) as any;
    const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApp();
    return getFirestore(app);
  } catch {
    // firebase package not installed or initialization failed — polling fallback active.
    return null;
  }
}

/**
 * Returns an FCM Messaging instance if Firebase is fully configured,
 * otherwise returns null.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function getFCMMessaging(): Promise<any | null> {
  if (!firebaseConfig) return null;
  if (!process.env.NEXT_PUBLIC_FIREBASE_VAPID_KEY) return null;
  try {
    const firebaseAppPkg = 'firebase/app';
    const messagingPkg = 'firebase/messaging';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { initializeApp, getApps, getApp } = await import(/* @vite-ignore */ firebaseAppPkg) as any;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { getMessaging, isSupported } = await import(/* @vite-ignore */ messagingPkg) as any;
    if (!(await isSupported())) return null;
    const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApp();
    return getMessaging(app);
  } catch {
    return null;
  }
}
