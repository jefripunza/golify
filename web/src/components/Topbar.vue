<script setup lang="ts">
import { Menu, LogOut } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { useSidebar } from '@/composables/useSidebar'
import { useHealth } from '@/stores/health'
import { useAuthStore } from '@/stores/auth'

const { toggleMobile } = useSidebar()
const health = useHealth()
const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.replace({ name: 'login' })
}
</script>

<template>
  <!--
    Single sticky topbar across breakpoints. On mobile, the layout is:
      [hamburger] [title]                      [logout]
    On desktop:
      [title]                       [health] [user] [logout]
    The hamburger is rendered FIRST in the DOM so it is never overlapped by
    other absolutely-positioned UI, and it sits at z-50 above the sticky
    header (z-30). The button is type="button" to avoid form-submit defaults,
    has explicit aria-label, and uses active:scale for tap feedback.
  -->
  <header
    class="sticky top-0 z-30 flex h-12 items-center gap-2 border-b border-border bg-card/80 px-4 backdrop-blur"
  >
    <button
      type="button"
      class="relative z-50 inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card shadow-sm transition-transform active:scale-95 md:hidden"
      aria-label="Open menu"
      data-testid="sidebar-toggle"
      @click="toggleMobile"
    >
      <Menu class="size-5" />
    </button>

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
        aria-label="Sign out"
        title="Sign out"
        @click="logout"
      >
        <LogOut class="size-3.5" />
      </button>
    </div>

    <!-- Mobile-only logout (no health/user pill to keep the bar compact) -->
    <button
      v-if="auth.user"
      type="button"
      class="ml-auto inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card md:hidden"
      aria-label="Sign out"
      title="Sign out"
      @click="logout"
    >
      <LogOut class="size-4" />
    </button>
  </header>
</template>