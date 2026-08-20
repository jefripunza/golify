import ky from 'ky'

// All FE→BE traffic goes through /api/* which the Go backend serves directly.
// In dev, Vite proxies /api to http://127.0.0.1:3000.
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
// already-logged-in user straight back to /login.
//
// Layered persistence (window → sessionStorage → localStorage):
// - window: survives module HMR (Vite dev) and SPA navigation
// - sessionStorage: survives full page reloads, works in Safari private mode
//   (unlike localStorage, which Safari can silently drop or quota-throw)
// - localStorage: best-effort cross-session persistence (regular browsing)
//
// IMPORTANT: the window mirror lives on `window`, NOT module scope. In Vite
// dev, HMR can instantiate the same module twice (different import chains) —
// a module-level `let memoryAuth` would give LoginView and the router guard
// two DIFFERENT mirrors. A window global is shared by every module instance.
declare global {
  interface Window {
    __golify_auth__?: AuthState | null
    __golify_just_logged_in__?: boolean
  }
}

function readMemoryAuth(): AuthState | null {
  return window.__golify_auth__ ?? null
}

function tryParse(raw: string | null): AuthState | null {
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthState
  } catch {
    return null
  }
}

export function getAuth(): AuthState | null {
  const mem = readMemoryAuth()
  if (mem) return mem
  // sessionStorage next — survives reload, works in Safari private mode
  try {
    const s = sessionStorage.getItem(AUTH_KEY)
    if (s) return tryParse(s)
  } catch {
    /* ignore */
  }
  try {
    const raw = localStorage.getItem(AUTH_KEY)
    if (raw) return tryParse(raw)
  } catch {
    /* ignore */
  }
  // cookie last resort — survives reload even when storage APIs throw
  return readCookie()
}

export function setAuth(auth: AuthState | null) {
  window.__golify_auth__ = auth
  if (auth) {
    const json = JSON.stringify(auth)
    try {
      sessionStorage.setItem(AUTH_KEY, json)
    } catch {
      /* ignore */
    }
    try {
      localStorage.setItem(AUTH_KEY, json)
    } catch {
      // localStorage unavailable (private mode / quota) — window + session
      // mirrors keep the session alive
    }
    // cookie layer: survives full page reloads even when storage APIs are
    // blocked (Safari private mode / ITP). Non-persistent (session cookie).
    try {
      document.cookie = `${AUTH_KEY}=${encodeURIComponent(json)}; path=/; SameSite=Lax`
    } catch {
      /* ignore */
    }
  } else {
    try {
      sessionStorage.removeItem(AUTH_KEY)
    } catch {
      /* ignore */
    }
    try {
      localStorage.removeItem(AUTH_KEY)
    } catch {
      /* ignore */
    }
    try {
      document.cookie = `${AUTH_KEY}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`
    } catch {
      /* ignore */
    }
  }
}

function readCookie(): AuthState | null {
  try {
    const m = document.cookie.match(/(?:^|;\s*)golify:auth=([^;]*)/)
    if (!m) return null
    return tryParse(decodeURIComponent(m[1]))
  } catch {
    return null
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
            ({ request, options }: { request: Request; options: { headers?: HeadersInit } }) => {
              // ky v2: hook receives ({ request, options }). Set the token
              // on both request.headers and options.headers defensively.
              try {
                request.headers.set('Authorization', `Bearer ${auth.token}`)
              } catch {
                /* ignore */
              }
              try {
                const headers = new Headers(options.headers)
                headers.set('Authorization', `Bearer ${auth.token}`)
                options.headers = headers
              } catch {
                /* ignore */
              }
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
  if (!auth?.token) {
    // window/sessionStorage mirror may hold it even if this module's
    // getAuth() sees nothing (bundle split instances)
    const win = (window as any).__golify_auth__
    if (win?.token) return true // token present in mirror — trust it
    // A login just completed (store sets this flag right after setAuth) —
    // never bounce in the middle of that flow; let the router guard decide.
    if ((window as any).__golify_just_logged_in__) return true
    return false
  }
  try {
    const res = await authed().get('api/v1/auth/me')
    return res.status === 200
  } catch {
    // 401 → invalid/expired token; clear it
    setAuth(null)
    return false
  }
}