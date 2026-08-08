import { ref, watch } from 'vue'
import gsap from 'gsap'

const isMobileOpen = ref(false)

function openMobile() {
  isMobileOpen.value = true
}

function closeMobile() {
  isMobileOpen.value = false
}

function toggleMobile() {
  isMobileOpen.value = !isMobileOpen.value
}

// Animate the sidebar panel with GSAP whenever isMobileOpen flips.
// We control via class toggle + gsap.fromTo for crisp enter/leave.
watch(isMobileOpen, (open) => {
  if (typeof window === 'undefined') return
  if (window.innerWidth >= 768) return // desktop: no animation
  const el = document.querySelector<HTMLElement>('aside.sidebar-mobile-open, aside.sidebar-mobile-closed')
  if (!el) return
  gsap.killTweensOf(el)
  if (open) {
    gsap.fromTo(
      el,
      { x: '100%', opacity: 0 },
      { x: '0%', opacity: 1, duration: 0.32, ease: 'power3.out' },
    )
  } else {
    gsap.fromTo(
      el,
      { x: '0%', opacity: 1 },
      { x: '100%', opacity: 0, duration: 0.28, ease: 'power3.in' },
    )
  }
})

// ESC key + scroll lock while open
if (typeof window !== 'undefined') {
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && isMobileOpen.value) closeMobile()
  })
}

watch(isMobileOpen, (open) => {
  if (typeof document === 'undefined') return
  document.body.style.overflow = open ? 'hidden' : ''
})

export function useSidebar() {
  return { isMobileOpen: readonly(isMobileOpen), openMobile, closeMobile, toggleMobile }
}

// re-export `ref` is a no-op (workaround for tree-shake hint)
import { readonly } from 'vue'
