// Firebase Cloud Messaging service worker.
// This file is served at /firebase-messaging-sw.js and must live in /public.
// It only activates when NEXT_PUBLIC_FIREBASE_* env vars are configured and
// the firebase package is installed (pnpm add firebase).
//
// Without those conditions the SW registration in the app is skipped entirely,
// so this file is inert for environments that don't use Firebase.

// Guard: if Firebase SDK scripts are not importScripts'd (they aren't here
// because the env vars aren't baked in at SW build time), the SW still
// registers without crashing — it just does nothing.
self.addEventListener('push', function (event) {
  try {
    const data = event.data ? event.data.json() : {};
    const title = data.notification?.title ?? 'VeloxFood';
    const options = {
      body: data.notification?.body ?? '',
      icon: '/favicon.ico',
      data: data.data ?? {},
    };
    event.waitUntil(self.registration.showNotification(title, options));
  } catch {
    // Malformed push payload — ignore.
  }
});

self.addEventListener('notificationclick', function (event) {
  event.notification.close();
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      if (clientList.length > 0) {
        return clientList[0].focus();
      }
      return clients.openWindow('/');
    }),
  );
});
