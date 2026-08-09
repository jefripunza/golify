import { createRouter, createWebHistory } from 'vue-router'
import { getAuth } from '@/lib/api'

const requireAuth = () => ({
  // Route guard — redirect to /login when there's no stored token.
  beforeEnter: async () => {
    const a = getAuth()
    if (!a?.token) {
      return { name: 'login' }
    }
    return true
  },
})

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // Public
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue') },

    // Main
    { path: '/', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), ...requireAuth() },
    { path: '/projects', name: 'projects', component: () => import('@/views/ProjectsView.vue'), ...requireAuth() },
    {
      path: '/projects/:projectId',
      name: 'project-detail',
      component: () => import('@/views/ProjectDetailView.vue'),
      ...requireAuth(),
    },
    {
      path: '/projects/:projectId/:envId',
      name: 'env-detail',
      component: () => import('@/views/EnvDetailView.vue'),
      ...requireAuth(),
    },
    {
      path: '/projects/:projectId/:envId/:serviceId',
      name: 'service-detail',
      component: () => import('@/views/ServiceDetailView.vue'),
      ...requireAuth(),
    },

    // Infrastructure
    { path: '/servers', name: 'servers', component: () => import('@/views/ServersView.vue'), ...requireAuth() },
    {
      path: '/servers/:serverId',
      name: 'server-detail',
      component: () => import('@/views/ServerDetailView.vue'),
      ...requireAuth(),
    },
    { path: '/sources', name: 'sources', component: () => import('@/views/SourcesView.vue'), ...requireAuth() },
    { path: '/s3', name: 's3', component: () => import('@/views/S3View.vue'), ...requireAuth() },
    { path: '/variables', name: 'variables', component: () => import('@/views/SharedVarsView.vue'), ...requireAuth() },

    // Security
    { path: '/keys', name: 'keys', component: () => import('@/views/KeysView.vue'), ...requireAuth() },
    { path: '/api-mcp', name: 'api-mcp', component: () => import('@/views/ApiMcpView.vue'), ...requireAuth() },
    { path: '/teams', name: 'teams', component: () => import('@/views/TeamsView.vue'), ...requireAuth() },
    {
      path: '/teams/:teamId',
      name: 'team-detail',
      component: () => import('@/views/TeamDetailView.vue'),
      ...requireAuth(),
    },

    // Catch-all → dashboard
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

export default router