import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Load web/.env (VITE_ prefix). Change ports/targets without touching code.
  const env = loadEnv(mode, process.cwd(), '')
  const devPort = Number(env.VITE_DEV_PORT || 5173)
  const apiTarget = env.VITE_DEV_PROXY_API || 'http://127.0.0.1:20001'
  const wsTarget = env.VITE_DEV_PROXY_WS || 'http://127.0.0.1:20002'

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
    server: {
      port: devPort,
      strictPort: true,
      host: '0.0.0.0',
      allowedHosts: ['golify.jefripunza.com'],
      proxy: {
        // proxy /api to the Go backend in dev
        '/api': apiTarget,
        // proxy /ws (xterm WebSocket) to the Go dashboard server (same port as SPA)
        '/ws': {
          target: wsTarget,
          ws: true,
        },
      },
    },
  }
})
