import ky from 'ky'

// All FE→BE traffic goes through /api/* which the Go backend serves directly.
// In dev, Vite proxies /api to http://127.0.0.1:20001.
// In prod (embedded build), requests go to the same origin.
export const http = ky.create({
  prefix: '/',
  timeout: 15_000,
  retry: { limit: 2, methods: ['get'] },
})

export interface Message {
  id: number
  title: string
  message: string
  priority: number
  created_at: string
}
