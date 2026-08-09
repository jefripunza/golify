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

export function getAuth(): AuthState | null {
  try {
    const raw = localStorage.getItem(AUTH_KEY)
    return raw ? (JSON.parse(raw) as AuthState) : null
  } catch {
    return null
  }
}

export function setAuth(auth: AuthState | null) {
  if (auth) localStorage.setItem(AUTH_KEY, JSON.stringify(auth))
  else localStorage.removeItem(AUTH_KEY)
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
        },
      })
    : http
}