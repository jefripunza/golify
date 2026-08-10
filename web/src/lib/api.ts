import ky from 'ky'

// All FE→BE traffic goes through /api/* which the Go backend serves directly.
// In dev, Vite proxies /api to http://127.0.0.1:20001.
// In prod (embedded build), requests go to the same origin.
export const http = ky.create({
  prefix: '/',
  timeout: 15_000,
  retry: { limit: 2, methods: ['get'] },
})

export const AUTH_KEY = 'golify:auth'

export interface AuthState {
  token: string
  user: { id: number; username: string; email?: string; admin: boolean }
}

// In-memory auth cache. Safari iOS (private mode / ITP) can silently drop
// localStorage writes — the guard would then see "no token" and bounce an
// already-logged-in user straight back to /login. Keeping a module-level
// mirror makes auth state survive even when localStorage is unavailable.
let memoryAuth: AuthState | null = null

export function getAuth(): AuthState | null {
  if (memoryAuth) return memoryAuth
  try {
    const raw = localStorage.getItem(AUTH_KEY)
    return raw ? (JSON.parse(raw) as AuthState) : null
  } catch {
    return null
  }
}

export function setAuth(auth: AuthState | null) {
  memoryAuth = auth
  if (auth) {
    try {
      localStorage.setItem(AUTH_KEY, JSON.stringify(auth))
    } catch {
      // localStorage unavailable (private mode / quota) — memory mirror keeps
      // the session alive for this page load
    }
  } else {
    try {
      localStorage.removeItem(AUTH_KEY)
    } catch {
      /* ignore */
    }
  }
}

// authed() returns a ky instance that attaches the Bearer token.
export function authed() {
  const auth = getAuth()
  return auth
    ? ky.create({
        prefix: '/',
        timeout: 15_000,
        retry: { limit: 2, methods: ['get'] },
        hooks: {
          beforeRequest: [
            (request) => {
              request.headers.set('Authorization', `Bearer ${auth.token}`)
            },
          ],
        } as any,
      })
    : http
}

// validateSession() checks the stored token against /api/v1/auth/me.
// Returns true if the session is still valid, false otherwise (and clears it).
export async function validateSession(): Promise<boolean> {
  const auth = getAuth()
  if (!auth?.token) return false
  try {
    const res = await authed().get('api/v1/auth/me')
    return res.status === 200
  } catch {
    // 401 → invalid/expired token; clear it
    setAuth(null)
    return false
  }
}