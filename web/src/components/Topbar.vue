<script setup lang="ts">
import { Menu, Bell } from '@lucide/vue'
import { useSidebar } from '@/composables/useSidebar'
import { useHealth } from '@/stores/health'

const { openMobile } = useSidebar()
const health = useHealth()
</script>

<template>
  <header
    class="sticky top-0 z-30 flex h-12 items-center justify-between border-b border-border bg-card/80 px-4 backdrop-blur"
  >
    <!-- Hamburger (mobile only) -->
    <button
      class="fixed top-3 right-3 z-40 rounded-md border border-border bg-card p-2 shadow-sm md:hidden"
      aria-label="Open menu"
      @click="openMobile"
    >
      <Menu class="size-4" />
    </button>

    <div class="flex items-center gap-2 text-sm text-muted-foreground">
      <Bell class="size-4" />
      <span>Gotify Dashboard</span>
    </div>

    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <span
        :class="[
          'size-2 rounded-full',
          health.data.value?.status === 'ok' ? 'bg-green-500' : 'bg-red-500',
        ]"
      />
      <code class="rounded bg-muted px-1.5 py-0.5">
        {{ health.data.value?.app ?? '...' }}
      </code>
    </div>
  </header>
</template>
