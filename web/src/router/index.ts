import { createRouter, createWebHistory } from 'vue-router'
import { getAuth } from '@/lib/api'

// ── Lazy-loaded views ───────────────────────────────────────────────────────
// Setiap view di-chunk terpisah (code-split otomatis) — hanya dimuat saat
// route dikunjungi. View berat (Dashboard/Highcharts, ServiceDetail/Monaco)
// tidak lagi membebani bundle awal.
const AuthLayout = () => import('@/layouts/AuthLayout.vue')
const AppLayout = () => import('@/layouts/AppLayout.vue')
const LoginView = () => import('@/views/LoginView.vue')
const OnboardingView = () => import('@/views/OnboardingView.vue')
const DashboardView = () => import('@/views/DashboardView.vue')
const DomainsView = () => import('@/views/DomainsView.vue')
const ProjectsView = () => import('@/views/ProjectsView.vue')
const ProjectDetailView = () => import('@/views/ProjectDetailView.vue')
const EnvDetailView = () => import('@/views/EnvDetailView.vue')
const ServiceDetailView = () => import('@/views/ServiceDetailView.vue')
const DeployDetailView = () => import('@/views/DeployDetailView.vue')
const ServersView = () => import('@/views/ServersView.vue')
const ServerDetailView = () => import('@/views/ServerDetailView.vue')
const SourcesView = () => import('@/views/SourcesView.vue')
const S3View = () => import('@/views/S3View.vue')
const SharedVarsView = () => import('@/views/SharedVarsView.vue')
const KeysView = () => import('@/views/KeysView.vue')
const ApiKeysView = () => import('@/views/ApiKeysView.vue')
const TeamsView = () => import('@/views/TeamsView.vue')
const TeamDetailView = () => import('@/views/TeamDetailView.vue')
const NotFoundView = () => import('@/views/NotFoundView.vue')

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // ── Auth layout: standalone login (no sidebar, no topbar, no hamburger)
    {
      path: '/login',
      component: AuthLayout,
      children: [
        { path: '', name: 'login', component: LoginView },
      ],
    },

    // ── Onboarding: first-run admin creation (standalone, auth layout)
    {
      path: '/onboarding',
      component: AuthLayout,
      children: [
        { path: '', name: 'onboarding', component: OnboardingView },
      ],
    },

    // ── App layout: authenticated dashboard shell (sidebar + topbar)
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '', name: 'dashboard', component: DashboardView, ...requireAuth() },

        // Domains
        { path: 'domains', name: 'domains', component: DomainsView, ...requireAuth() },

        // Projects
        { path: 'projects', name: 'projects', component: ProjectsView, ...requireAuth() },
        { path: 'project/:projectId/environments', name: 'project-environments', component: ProjectDetailView, ...requireAuth() },
        { path: 'project/:projectId/environment/:envId/services', name: 'env-services', component: EnvDetailView, ...requireAuth() },
        { path: 'project/:projectId/environment/:envId/service/:serviceId', name: 'service-detail', component: ServiceDetailView, ...requireAuth() },
        { path: 'project/:projectId/environment/:envId/service/:serviceId/deploy/:deployId', name: 'deploy-detail', component: DeployDetailView, ...requireAuth() },

        // Infrastructure
        { path: 'servers', name: 'servers', component: ServersView, ...requireAuth() },
        { path: 'servers/:serverId', name: 'server-detail', component: ServerDetailView, ...requireAuth() },
        { path: 'sources', name: 'sources', component: SourcesView, ...requireAuth() },
        { path: 's3', name: 's3', component: S3View, ...requireAuth() },
        { path: 'variables', name: 'variables', component: SharedVarsView, ...requireAuth() },

        // Security
        { path: 'keys', name: 'keys', component: KeysView, ...requireAuth() },
        { path: 'api-keys', name: 'api-keys', component: ApiKeysView, ...requireAuth() },
        { path: 'teams', name: 'teams', component: TeamsView, ...requireAuth() },
        { path: 'teams/:teamId', name: 'team-detail', component: TeamDetailView, ...requireAuth() },
      ],
    },

    // ── 404 (inside AppLayout so it keeps the app chrome; requires auth too)
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: AppLayout,
      ...requireAuth(),
      children: [
        { path: '', name: 'not-found-inner', component: NotFoundView },
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

  // If a login flow just completed but every storage layer looks empty, the
  // state is mid-race (setAuth may not have finished). Never bounce a user
  // who just authenticated — let the next navigation settle it.
  const justLoggedIn = !!window.__golify_just_logged_in__
  if (!a?.token && justLoggedIn && to.name !== 'login') {
    return true
  }

  // Debug probe: whenever we bounce a user who just authenticated, report why
  // (only on /login-targeted navigations from app routes to avoid noise).
  if (!a?.token && to.name === 'login' && (window.__golify_auth__ || justLoggedIn)) {
    fetch('/api/report/error', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        app_name: 'Golify Dashboard',
        app_url: window.location.href,
        title: 'Guard bounce: session present but getAuth() null',
        stack: `winAuth=${!!window.__golify_auth__} justLoggedIn=${justLoggedIn} ls=${!!localStorage.getItem('golify:auth')} ss=${!!sessionStorage.getItem('golify:auth')} cookie=${!!document.cookie.match(/(?:^|;\s*)golify:auth=([^;]*)/)} nav=${to.fullPath}\n${new Error().stack || ''}`,
      }),
    }).catch(() => {})
    // one report per login attempt is enough — reset the flag so we don't
    // spam the error channel on every navigation
    window.__golify_just_logged_in__ = false
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
