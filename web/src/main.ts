import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'

import App from './App.vue'
import router from './router'
import './assets/main.css'
import { useTheme } from './composables/useTheme'

// apply saved theme before first paint (avoids light→dark flash)
useTheme().init()

// Global error reporting — catches errors that escape the ErrorBoundary
// (module-load failures, async errors, router navigation errors, unhandled
// rejections) and forwards them to the backend → error.sawang.tech.
function reportGlobalError(title: string, stack: string) {
  const payload = {
    app_name: 'Golify Dashboard',
    app_url: typeof window !== 'undefined' ? window.location.href : '',
    title,
    stack,
  }
  fetch('/api/report/error', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  }).catch(() => {
    /* best-effort */
  })
}

window.addEventListener('error', (e) => {
  const err = e.error
  if (err instanceof Error) {
    reportGlobalError(err.name || 'Error', err.stack || err.message)
  } else {
    reportGlobalError('Uncaught error', String(e.message || err))
  }
})

window.addEventListener('unhandledrejection', (e) => {
  const reason = e.reason
  if (reason instanceof Error) {
    reportGlobalError(reason.name || 'Unhandled rejection', reason.stack || reason.message)
  } else {
    reportGlobalError('Unhandled rejection', String(reason))
  }
})

router.onError((err, to) => {
  reportGlobalError(
    'Router navigation error',
    `${err?.message || String(err)}\n\nto: ${to?.fullPath || ''}\n\n${err?.stack || ''}`,
  )
})

const app = createApp(App)

app.use(createPinia())
app.use(PiniaColada, {})
app.use(router)

app.mount('#app')
