// Auth store: login/logout/current user, persisted in localStorage.
// All API calls from other stores should branch through getAuth() to attach the Bearer token.

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { AUTH_KEY, setAuth, type AuthState } from '@/lib/api'

export const useAuthStore = defineStore('auth', () => {
  const auth = ref<AuthState | null>(loadAuth())

  const isAuthenticated = computed(() => !!auth.value?.token)
  const user = computed(() => auth.value?.user ?? null)
  const isAdmin = computed(() => !!auth.value?.user.admin)

  async function login(email: string, password: string): Promise<void> {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { error?: string }
      throw new Error(body.error ?? `login failed (${res.status})`)
    }
    const body = (await res.json()) as AuthState
    auth.value = body
    setAuth(body)
  }

  function logout() {
    auth.value = null
    setAuth(null)
  }

  return { auth, isAuthenticated, user, isAdmin, login, logout }
})

function loadAuth(): AuthState | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = localStorage.getItem(AUTH_KEY)
    return raw ? (JSON.parse(raw) as AuthState) : null
  } catch {
    return null
  }
}