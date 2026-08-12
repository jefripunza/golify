<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  LayoutDashboard,
  FolderTree,
  Server,
  GitBranch,
  Database,
  KeyRound,
  Variable,
  Plug,
  Users,
  Globe,
  ChevronRight,
} from '@lucide/vue'
import { useSidebar } from '@/composables/useSidebar'

const route = useRoute()
const { isMobileOpen, closeMobile } = useSidebar()

interface MenuItem {
  label: string
  to: string
  icon: typeof LayoutDashboard
  group: 'main' | 'infra' | 'security'
}

const items: MenuItem[] = [
  { label: 'Dashboard', to: '/', icon: LayoutDashboard, group: 'main' },
  { label: 'Domains', to: '/domains', icon: Globe, group: 'main' },
  { label: 'Projects', to: '/projects', icon: FolderTree, group: 'main' },
  { label: 'Servers', to: '/servers', icon: Server, group: 'infra' },
  { label: 'Sources', to: '/sources', icon: GitBranch, group: 'infra' },
  { label: 'S3 Storages', to: '/s3', icon: Database, group: 'infra' },
  { label: 'Shared Variables', to: '/variables', icon: Variable, group: 'infra' },
  { label: 'Keys', to: '/keys', icon: KeyRound, group: 'security' },
  { label: 'API Keys & MCP', to: '/api-mcp', icon: Plug, group: 'security' },
  { label: 'Teams', to: '/teams', icon: Users, group: 'security' },
]

const groups = [
  { id: 'main', label: 'Main' },
  { id: 'infra', label: 'Infrastructure' },
  { id: 'security', label: 'Security' },
] as const

const grouped = computed(() =>
  groups.map((g) => ({ ...g, items: items.filter((i) => i.group === g.id) })),
)

function isActive(to: string) {
  if (to === '/') return route.path === '/'
  return route.path === to || route.path.startsWith(to + '/')
}
</script>

<template>
  <!--
    Mobile overlay + panel. Safari/iOS-safe approach:
    - `v-if` (display:none when closed) — NO transform/transition dependency
      for hiding, so Safari cannot get stuck showing an invisible panel.
    - open animation: CSS @keyframes slideInRight (uses `transform`, which all
      Safari versions support). Closing: quick fade via CSS class.
  -->

  <!-- Backdrop (mobile only) -->
  <div
    v-if="isMobileOpen"
    class="sidebar-backdrop fixed inset-0 z-40 bg-black/40 backdrop-blur-sm md:hidden"
    @click="closeMobile"
  />

  <!-- Desktop sidebar: persistent, in-flow (md:flex parent row) -->
  <aside
    class="hidden md:sticky md:top-0 md:flex md:h-screen md:w-60 md:shrink-0 md:flex-col md:border-r md:border-border md:bg-card md:text-card-foreground"
  >
    <div class="flex items-center justify-between border-b border-border px-4 py-3">
      <div class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="size-2 rounded-full bg-primary" />
        <span>Golify</span>
      </div>
    </div>

    <nav class="flex-1 overflow-y-auto px-2 py-3">
      <div v-for="g in grouped" :key="g.id" class="mb-4">
        <p class="px-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {{ g.label }}
        </p>
        <ul class="grid gap-0.5">
          <li v-for="item in g.items" :key="item.to">
            <RouterLink
              :to="item.to"
              :class="[
                'flex items-center justify-between rounded-md px-3 py-2 text-sm transition-colors',
                isActive(item.to)
                  ? 'bg-accent text-accent-foreground font-medium'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              ]"
            >
              <span class="flex items-center gap-2">
                <component :is="item.icon" class="size-4" />
                <span>{{ item.label }}</span>
              </span>
              <ChevronRight v-if="isActive(item.to)" class="size-3.5 opacity-60" />
            </RouterLink>
          </li>
        </ul>
      </div>
    </nav>

    <div class="border-t border-border px-4 py-3 text-xs text-muted-foreground">
      <p>v0.1.0 · build FE-only</p>
    </div>
  </aside>

  <!-- Mobile sidebar panel: v-if + CSS animation (Safari-safe) -->
  <div
    v-if="isMobileOpen"
    class="sidebar-mobile fixed right-0 top-0 z-50 flex h-full w-72 flex-col border-l border-border bg-card text-card-foreground"
  >
    <div class="flex items-center justify-between border-b border-border px-4 py-3">
      <div class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="size-2 rounded-full bg-primary" />
        <span>Golify</span>
      </div>
      <button
        type="button"
        class="rounded-md p-1.5 text-muted-foreground hover:bg-muted"
        aria-label="Close menu"
        @click="closeMobile"
      >
        ✕
      </button>
    </div>

    <nav class="flex-1 overflow-y-auto px-2 py-3">
      <div v-for="g in grouped" :key="g.id" class="mb-4">
        <p class="px-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {{ g.label }}
        </p>
        <ul class="grid gap-0.5">
          <li v-for="item in g.items" :key="item.to">
            <RouterLink
              :to="item.to"
              :class="[
                'flex items-center justify-between rounded-md px-3 py-2 text-sm transition-colors',
                isActive(item.to)
                  ? 'bg-accent text-accent-foreground font-medium'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              ]"
              @click="closeMobile"
            >
              <span class="flex items-center gap-2">
                <component :is="item.icon" class="size-4" />
                <span>{{ item.label }}</span>
              </span>
              <ChevronRight v-if="isActive(item.to)" class="size-3.5 opacity-60" />
            </RouterLink>
          </li>
        </ul>
      </div>
    </nav>

    <div class="border-t border-border px-4 py-3 text-xs text-muted-foreground">
      <p>v0.1.0 · build FE-only</p>
    </div>
  </div>
</template>

<style>
/* Backdrop fade (Safari-safe) */
.sidebar-backdrop {
  animation: sidebar-fade-in 0.2s ease-out;
}
@keyframes sidebar-fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* Mobile panel slide-in from right (Safari-safe: uses transform + keyframes) */
.sidebar-mobile {
  animation: sidebar-slide-in 0.28s cubic-bezier(0.32, 0.72, 0, 1);
  will-change: transform;
}
@keyframes sidebar-slide-in {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}
</style>