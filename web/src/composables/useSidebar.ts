import { ref, computed, readonly } from 'vue'

// Mobile sidebar state — singleton ref shared across every useSidebar() call.
// The ref is intentionally module-level so all components see the same value.
// We attach GSAP-free logic only; Sidebar.vue handles the slide animation
// with a pure CSS transition (avoids the GSAP-vs-class race that broke the
// first mobile click on iOS Safari).

const g = globalThis as unknown as { __golifySidebar?: { open: ReturnType<typeof ref<boolean>> } }
if (!g.__golifySidebar) {
  g.__golifySidebar = { open: ref(false) }
}
const isMobileOpen = g.__golifySidebar.open

function openMobile() { isMobileOpen.value = true }
function closeMobile() { isMobileOpen.value = false }
function toggleMobile() { isMobileOpen.value = !isMobileOpen.value }

// ESC closes; idempotent handler (HMR-safe).
if (typeof window !== 'undefined') {
  const w = window as unknown as { __golifySidebarKeyHandler?: boolean }
  if (!w.__golifySidebarKeyHandler) {
    w.__golifySidebarKeyHandler = true
    window.addEventListener('keydown', (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isMobileOpen.value) closeMobile()
    })
  }
}

export function useSidebar() {
  return {
    isMobileOpen: readonly(isMobileOpen),
    openMobile,
    closeMobile,
    toggleMobile,
    // mutable alias (used by v-if on overlay, kept for backwards compat)
    isMobileOpenRef: computed({
      get: () => isMobileOpen.value,
      set: (v: boolean) => { isMobileOpen.value = v },
    }),
  }
}