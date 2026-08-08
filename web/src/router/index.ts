import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // Main
    { path: '/', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
    { path: '/projects', name: 'projects', component: () => import('@/views/ProjectsView.vue') },
    {
      path: '/projects/:projectId',
      name: 'project-detail',
      component: () => import('@/views/ProjectDetailView.vue'),
    },
    {
      path: '/projects/:projectId/:envId',
      name: 'env-detail',
      component: () => import('@/views/EnvDetailView.vue'),
    },
    {
      path: '/projects/:projectId/:envId/:serviceId',
      name: 'service-detail',
      component: () => import('@/views/ServiceDetailView.vue'),
    },

    // Infrastructure
    { path: '/servers', name: 'servers', component: () => import('@/views/ServersView.vue') },
    {
      path: '/servers/:serverId',
      name: 'server-detail',
      component: () => import('@/views/ServerDetailView.vue'),
    },
    { path: '/sources', name: 'sources', component: () => import('@/views/SourcesView.vue') },
    { path: '/s3', name: 's3', component: () => import('@/views/S3View.vue') },
    { path: '/variables', name: 'variables', component: () => import('@/views/SharedVarsView.vue') },

    // Security
    { path: '/keys', name: 'keys', component: () => import('@/views/KeysView.vue') },
    { path: '/api-mcp', name: 'api-mcp', component: () => import('@/views/ApiMcpView.vue') },
    { path: '/teams', name: 'teams', component: () => import('@/views/TeamsView.vue') },
    {
      path: '/teams/:teamId',
      name: 'team-detail',
      component: () => import('@/views/TeamDetailView.vue'),
    },

    // Catch-all → dashboard
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

export default router
