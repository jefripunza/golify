import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/messages', name: 'messages', component: () => import('@/views/MessagesView.vue') },
    { path: '/send', name: 'send', component: () => import('@/views/SendView.vue') },
  ],
})

export default router
