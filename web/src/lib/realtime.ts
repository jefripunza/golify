// Realtime WS subscription — connects to /api/ws/realtime whenever the user
// is logged in and stays connected (auto-reconnect with backoff). Mutation
// events (project/env/service/domain/deployment) trigger targeted refetches
// instead of the old polling loops.
import { onBeforeUnmount, ref, watch } from 'vue'
import { getAuth } from '@/lib/api'
import { useProjectsStore } from '@/stores'

export interface RealtimeEvent {
  type: string // project | environment | service | domain | deployment | health
  action: string // created | updated | deleted | finished
  id?: string
  at?: number
}

const connected = ref(false)
let ws: WebSocket | null = null
let retry = 0
let timer: number | null = null
let closed = false

function connect() {
  if (closed) return
  const auth = getAuth()
  if (!auth?.token) {
    // not logged in yet — retry shortly (guard/login will set auth)
    scheduleRetry(2000)
    return
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/api/ws/realtime?token=${encodeURIComponent(auth.token)}`
  try {
    ws = new WebSocket(url)
  } catch {
    scheduleRetry(3000)
    return
  }
  ws.onopen = () => {
    connected.value = true
    retry = 0
  }
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(String(ev.data)) as RealtimeEvent
      if (data.type === 'ping') return
      handleEvent(data)
    } catch {
      /* ignore malformed */
    }
  }
  ws.onclose = () => {
    connected.value = false
    ws = null
    scheduleRetry(backoff())
  }
  ws.onerror = () => {
    ws?.close()
  }
}

function backoff(): number {
  const base = [2000, 3000, 5000, 8000, 12000]
  const ms = base[Math.min(retry, base.length - 1)]
  retry++
  return ms
}

function scheduleRetry(ms: number) {
  if (closed || timer !== null) return
  timer = window.setTimeout(() => {
    timer = null
    connect()
  }, ms)
}

function handleEvent(ev: RealtimeEvent) {
  const store = useProjectsStore()
  switch (ev.type) {
    case 'project':
      // full project list changed (create/update/delete)
      void store.refresh()
      break
    case 'environment':
    case 'service':
    case 'domain':
      // project tree changed → refresh the project list (nested envs/services)
      void store.refresh()
      break
    case 'deployment':
      // deployment created/finished → refresh project (service status + list)
      void store.refresh()
      break
  }
}

// Connect on app mount; keep trying while the app is open.
export function startRealtime() {
  closed = false
  connect()
}

export function stopRealtime() {
  closed = true
  if (timer !== null) clearTimeout(timer)
  timer = null
  ws?.close()
  ws = null
  connected.value = false
}

// Watch auth: connect as soon as a token appears (login), disconnect on logout.
export function useRealtime() {
  const auth = ref(getAuth())
  watch(auth, () => {
    if (getAuth()?.token) connect()
  })
  // simple poll-free auth watcher: re-check on a short interval is what we're
  // removing — instead hook into storage events + route guard.
  return { connected }
}
