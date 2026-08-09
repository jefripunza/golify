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
      const a = getAuth()
      if (!a?.token) return { name: 'login' }
      return true
    },
  }
}

// ── Global guard:
//   1. If there's a stored token, the app routes are allowed.
//   2. Unauthenticated users get sent to /login (or /onboarding if the
//      backend reports no users yet).
let onboardingChecked = false
let needsOnboarding = false

router.beforeEach(async (to) => {
  const a = getAuth()
  const isPublic = to.name === 'login' || to.name === 'onboarding'

  // Authenticated user visiting login/onboarding → dashboard.
  if (isPublic && a?.token) {
    return { name: 'dashboard' }
  }

  // Public route, no token: check onboarding state once.
  if (!a?.token && !isPublic) {
    if (!onboardingChecked) {
      try {
        const res = await fetch('/api/v1/auth/status')
        const body = (await res.json()) as { onboarded?: boolean }
        needsOnboarding = !body.onboarded
      } catch {
        needsOnboarding = false // API down — fall back to login
      }
      onboardingChecked = true
    }
    return { name: needsOnboarding ? 'onboarding' : 'login' }
  }

  return true
})

export default router