// Auth store: login/logout/current user, persisted in localStorage.
// All API calls from other stores should branch through getAuth() to attach the Bearer token.

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getAuth, setAuth, type AuthState } from '@/lib/api'

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
    // mark the login flow BEFORE persisting so racing guards/WS-close
    // handlers never wipe a session that is being created right now
    window.__golify_just_logged_in__ = true
    auth.value = body
    setAuth(body)
  }

  function logout() {
    auth.value = null
    setAuth(null)
    window.__golify_just_logged_in__ = false
  }

  return { auth, isAuthenticated, user, isAdmin, login, logout }
})

function loadAuth(): AuthState | null {
  if (typeof window === 'undefined') return null
  return getAuth()
}