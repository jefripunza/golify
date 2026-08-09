import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
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
    port: 5173,
    strictPort: true,
    host: '0.0.0.0',
    allowedHosts: ['golify.jefripunza.com'],
    proxy: {
      // proxy /api to the Go backend in dev
      '/api': 'http://127.0.0.1:20001',
      // proxy /ws (xterm WebSocket) to the Go WS server in dev
      '/ws': {
        target: 'http://127.0.0.1:20004',
        ws: true,
      },
    },
  },
})
