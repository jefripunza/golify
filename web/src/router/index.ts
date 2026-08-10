import { createRouter, createWebHistory } from 'vue-router'
import { getAuth } from '@/lib/api'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // ── Auth layout: standalone login (no sidebar, no topbar, no hamburger)
    {
      path: '/login',
      component: () => import('@/layouts/AuthLayout.vue'),
      children: [
        { path: '', name: 'login', component: () => import('@/views/LoginView.vue') },
      ],
    },

    // ── Onboarding: first-run admin creation (standalone, auth layout)
    {
      path: '/onboarding',
      component: () => import('@/layouts/AuthLayout.vue'),
      children: [
        { path: '', name: 'onboarding', component: () => import('@/views/OnboardingView.vue') },
      ],
    },

    // ── App layout: authenticated dashboard shell (sidebar + topbar)
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      children: [
        { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), ...requireAuth() },

        // Projects
        { path: 'projects', name: 'projects', component: () => import('@/views/ProjectsView.vue'), ...requireAuth() },
        { path: 'projects/:projectId', name: 'project-detail', component: () => import('@/views/ProjectDetailView.vue'), ...requireAuth() },
        { path: 'projects/:projectId/:envId', name: 'env-detail', component: () => import('@/views/EnvDetailView.vue'), ...requireAuth() },
        { path: 'projects/:projectId/:envId/:serviceId', name: 'service-detail', component: () => import('@/views/ServiceDetailView.vue'), ...requireAuth() },

        // Infrastructure
        { path: 'servers', name: 'servers', component: () => import('@/views/ServersView.vue'), ...requireAuth() },
        { path: 'servers/:serverId', name: 'server-detail', component: () => import('@/views/ServerDetailView.vue'), ...requireAuth() },
        { path: 'sources', name: 'sources', component: () => import('@/views/SourcesView.vue'), ...requireAuth() },
        { path: 's3', name: 's3', component: () => import('@/views/S3View.vue'), ...requireAuth() },
        { path: 'variables', name: 'variables', component: () => import('@/views/SharedVarsView.vue'), ...requireAuth() },

        // Security
        { path: 'keys', name: 'keys', component: () => import('@/views/KeysView.vue'), ...requireAuth() },
        { path: 'api-mcp', name: 'api-mcp', component: () => import('@/views/ApiMcpView.vue'), ...requireAuth() },
        { path: 'teams', name: 'teams', component: () => import('@/views/TeamsView.vue'), ...requireAuth() },
        { path: 'teams/:teamId', name: 'team-detail', component: () => import('@/views/TeamDetailView.vue'), ...requireAuth() },
      ],
    },

    // ── 404 (inside AppLayout so it keeps the app chrome; requires auth too)
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/layouts/AppLayout.vue'),
      ...requireAuth(),
      children: [
        { path: '', name: 'not-found-inner', component: () => import('@/views/NotFoundView.vue') },
      ],
    },
  ],
})

function requireAuth() {
  return {
    beforeEnter: async () => {
      const a = readAuthAnywhere()
      if (!a?.token) return { name: 'login' }
      return true
    },
  }
}

// Read auth without relying on a single module instance. In some bundle
// configurations (Vite code-splitting) the store's setAuth() and the guard's
// getAuth() can end up in different module instances — a window-level read
// is the one source of truth that is always shared.
function readAuthAnywhere() {
  const win = (window as any).__golify_auth__
  if (win?.token) return win
  try {
    const ss = sessionStorage.getItem('golify:auth')
    if (ss) {
      const p = JSON.parse(ss)
      if (p?.token) return p
    }
  } catch {
    /* ignore */
  }
  try {
    const ls = localStorage.getItem('golify:auth')
    if (ls) {
      const p = JSON.parse(ls)
      if (p?.token) return p
    }
  } catch {
    /* ignore */
  }
  try {
    const m = document.cookie.match(/(?:^|;\s*)golify:auth=([^;]*)/)
    if (m) {
      const p = JSON.parse(decodeURIComponent(m[1]))
      if (p?.token) return p
    }
  } catch {
    /* ignore */
  }
  return null
}

// ── Global guard:
//   1. Token present → app routes allowed; login/onboarding → dashboard.
//   2. No token → check /auth/status EACH navigation (no cache — so the
//      state after onboarding/logout is always correct):
//      - not onboarded → /login and everything else redirect to /onboarding
//      - onboarded → /onboarding redirects to /login; other routes → /login
router.beforeEach(async (to) => {
  const a = readAuthAnywhere()

  // Debug probe: whenever we bounce a user who just authenticated, report why
  // (only on /login-targeted navigations from app routes to avoid noise).
  if (!a?.token && to.name === 'login' && (window.__golify_auth__ || window.__golify_just_logged_in__)) {
    fetch('/api/report/error', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        app_name: 'Golify Dashboard',
        app_url: window.location.href,
        title: 'Guard bounce: session present but getAuth() null',
        stack: `winAuth=${!!window.__golify_auth__} justLoggedIn=${!!window.__golify_just_logged_in__} ls=${!!localStorage.getItem('golify:auth')} ss=${!!sessionStorage.getItem('golify:auth')} nav=${to.fullPath}\n${new Error().stack || ''}`,
      }),
    }).catch(() => {})
  }

  // Authenticated: login/onboarding pages make no sense → dashboard.
  if (a?.token) {
    if (to.name === 'login' || to.name === 'onboarding') {
      return { name: 'dashboard' }
    }
    return true
  }

  // No token: check onboarding state.
  let needsOnboarding = false
  try {
    const res = await fetch('/api/v1/auth/status')
    const body = (await res.json()) as { onboarded?: boolean }
    needsOnboarding = !body.onboarded
  } catch {
    needsOnboarding = false // API down — fall back to login
  }

  if (needsOnboarding) {
    // Not onboarded yet: /login is NOT allowed — force onboarding.
    if (to.name === 'onboarding') return true
    return { name: 'onboarding' }
  }

  // Onboarded: /onboarding is NOT allowed — force login.
  if (to.name === 'onboarding') return { name: 'login' }
  if (to.name === 'login') return true
  return { name: 'login' }
})

export default router