<script setup lang="ts">
import { Menu, LogOut, Sun, Moon } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { useSidebar } from '@/composables/useSidebar'
import { useHealth } from '@/stores/health'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'

const { toggleMobile } = useSidebar()
const health = useHealth()
const auth = useAuthStore()
const router = useRouter()
const theme = useTheme()

function logout() {
  auth.logout()
  router.replace({ name: 'login' })
}
</script>

<template>
  <!--
    Sticky topbar. On mobile the hamburger sits at the RIGHT (per Pak Jefri's
    preference), title at left, logout on the right of the hamburger.
    Desktop keeps health pill + username + logout on the right, no hamburger.
  -->
  <header
    class="sticky top-0 z-30 flex h-12 items-center gap-2 border-b border-border bg-card/80 px-4 backdrop-blur"
  >
    <div class="flex flex-1 items-center gap-2 text-sm text-muted-foreground">
      <span>Golify Dashboard</span>
    </div>

    <!-- Desktop-only actions -->
    <div class="hidden items-center gap-3 text-xs text-muted-foreground md:flex">
      <span
        :class="[
          'size-2 rounded-full',
          health.data?.status === 'ok' ? 'bg-green-500' : 'bg-red-500',
        ]"
      />
      <code class="rounded bg-muted px-1.5 py-0.5">
        {{ health.data?.app ?? '...' }}
      </code>
      <span v-if="auth.user" class="rounded bg-muted px-1.5 py-0.5">{{ auth.user.username }}</span>
      <button
        type="button"
        class="rounded-md border border-border p-1.5 hover:bg-muted"
        :aria-label="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        :title="theme.isDark ? 'Light mode' : 'Dark mode'"
        @click="theme.toggle"
      >
        <Sun v-if="theme.isDark" class="size-3.5" />
        <Moon v-else class="size-3.5" />
      </button>
      <button
        type="button"
        class="rounded-md border border-border p-1.5 hover:bg-muted"
        aria-label="Sign out"
        title="Sign out"
        @click="logout"
      >
        <LogOut class="size-3.5" />
      </button>
    </div>

    <!-- Mobile actions: [theme] [logout] [hamburger] — hamburger on the RIGHT -->
    <div class="flex items-center gap-2 md:hidden">
      <button
        type="button"
        class="inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card active:scale-95"
        :aria-label="theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        :title="theme.isDark ? 'Light mode' : 'Dark mode'"
        @click="theme.toggle"
      >
        <Sun v-if="theme.isDark" class="size-4" />
        <Moon v-else class="size-4" />
      </button>
      <button
        v-if="auth.user"
        type="button"
        class="inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card active:scale-95"
        aria-label="Sign out"
        title="Sign out"
        @click="logout"
      >
        <LogOut class="size-4" />
      </button>
      <button
        type="button"
        class="inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card shadow-sm transition-transform active:scale-95"
        aria-label="Open menu"
        data-testid="sidebar-toggle"
        @click="toggleMobile"
      >
        <Menu class="size-5" />
      </button>
    </div>
  </header>
</template>