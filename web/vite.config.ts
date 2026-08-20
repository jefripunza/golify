import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load web/.env (VITE_ prefix). Change ports/targets without touching code.
  const env = loadEnv(mode, process.cwd(), '')
  const devPort = Number(env.VITE_DEV_PORT || 20003)
  const apiTarget = env.VITE_DEV_PROXY_API || 'http://127.0.0.1:3000'

  return {
    plugins: [
      vue(),
      vueDevTools(),
      tailwindcss(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    optimizeDeps: {
      include: ['monaco-editor'],
    },
    worker: {
      format: 'es',
    },
    server: {
      port: devPort,
      strictPort: true,
      host: '0.0.0.0',
      allowedHosts: ['golify.jefripunza.com', 'simtaru.online', 'wajadi.online', '.simtaru.online', '.wajadi.online'],
      proxy: {
        // proxy /api to the Go BE in dev — WS dashboard lives at /api/ws/*
        // (ws:true so WebSocket upgrades on /api/ws pass through)
        '/api': {
          target: apiTarget,
          ws: true,
        },
      },
    },
  }
})
