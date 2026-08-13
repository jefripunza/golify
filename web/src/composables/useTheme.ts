// Theme composable — dark/light toggle with localStorage persistence.
// Toggles the `.dark` class on <html> (Tailwind v4 dark variant).
import { ref } from 'vue'

const KEY = 'golify.theme'

export function useTheme() {
  const isDark = ref(false)

  function apply(v: boolean) {
    isDark.value = v
    if (typeof document !== 'undefined') {
      document.documentElement.classList.toggle('dark', v)
    }
    try {
      localStorage.setItem(KEY, v ? 'dark' : 'light')
    } catch {
      /* ignore */
    }
  }

  function init() {
    let v = false
    try {
      const saved = localStorage.getItem(KEY)
      v = saved === 'dark'
      if (!saved) {
        // default: follow system preference
        v = window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false
      }
    } catch {
      /* ignore */
    }
    apply(v)
  }

  function toggle() {
    apply(!isDark.value)
  }

  return { isDark, init, toggle }
}
