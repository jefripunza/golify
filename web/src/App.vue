<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, RouterLink, useRoute } from 'vue-router'
import { useHealth } from '@/stores/messages'
import { Bell, MessageSquare, Send } from '@lucide/vue'

const health = useHealth()
const route = useRoute()

const healthData = computed(() => health.data.value)
const isHealthy = computed(() => healthData.value?.status === 'ok')
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <header class="sticky top-0 z-10 border-b border-border bg-card/80 backdrop-blur">
      <div class="mx-auto flex max-w-5xl items-center gap-6 px-4 py-3">
        <RouterLink to="/" class="flex items-center gap-2 font-semibold tracking-tight">
          <Bell class="size-5 text-primary" />
          <span>Gotify</span>
        </RouterLink>
        <nav class="flex flex-1 gap-1">
          <RouterLink
            to="/"
            :class="[
              'rounded-md px-3 py-1.5 text-sm transition-colors',
              route.path === '/' ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-muted',
            ]"
          >Dashboard</RouterLink>
          <RouterLink
            to="/messages"
            :class="[
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors',
              route.path.startsWith('/messages') ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-muted',
            ]"
          ><MessageSquare class="size-4" />Messages</RouterLink>
          <RouterLink
            to="/send"
            :class="[
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors',
              route.path.startsWith('/send') ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-muted',
            ]"
          ><Send class="size-4" />Send</RouterLink>
        </nav>
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <span :class="['size-2 rounded-full', isHealthy ? 'bg-green-500' : 'bg-red-500']" />
          <code class="rounded bg-muted px-1.5 py-0.5">{{ healthData?.app ?? '...' }}</code>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-5xl px-4 py-8">
      <RouterView />
    </main>
  </div>
</template>
