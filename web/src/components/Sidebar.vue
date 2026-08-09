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
  ChevronRight,
} from '@lucide/vue'
import { useSidebar } from '@/composables/useSidebar'

const route = useRoute()
const { isMobileOpen, isMobileOpenRef, closeMobile } = useSidebar()

interface MenuItem {
  label: string
  to: string
  icon: typeof LayoutDashboard
  group: 'main' | 'infra' | 'security'
}

const items: MenuItem[] = [
  { label: 'Dashboard', to: '/', icon: LayoutDashboard, group: 'main' },
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
  <!-- Backdrop overlay (mobile only) — placed behind the panel via z-index -->
  <Transition name="fade">
    <div
      v-if="isMobileOpen"
      class="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm md:hidden"
      @click="closeMobile"
    />
  </Transition>

  <!-- Sidebar: persistent on desktop, slide-in panel on mobile -->
  <aside
    :class="[
      'flex flex-col border-border bg-card text-card-foreground',
      // desktop: always visible, sticky, in-flow (md:flex parent = row)
      'md:sticky md:top-0 md:h-screen md:w-60 md:shrink-0 md:border-r md:translate-x-0 md:opacity-100',
      // mobile: fixed off-canvas to the right, slides in via CSS transition
      'fixed top-0 right-0 z-50 h-screen w-72 border-l transition-transform transition-opacity duration-300 ease-out',
      isMobileOpen ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0 pointer-events-none md:pointer-events-auto',
      // mobile-only when closed
      'md:translate-x-0',
    ]"
    :aria-hidden="!isMobileOpen && (typeof window !== 'undefined' && window.innerWidth < 768)"
    aria-label="Primary navigation"
  >
    <div class="flex items-center justify-between border-b border-border px-4 py-3">
      <div class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="size-2 rounded-full bg-primary" />
        <span>Golify</span>
      </div>
      <button
        type="button"
        class="rounded-md p-1.5 text-muted-foreground hover:bg-muted md:hidden"
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
  </aside>
</template>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>