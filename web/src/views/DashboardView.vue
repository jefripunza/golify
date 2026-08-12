<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import Highcharts from 'highcharts/es-modules/masters/highcharts.src.js'
import 'highcharts/es-modules/masters/highcharts-more.src.js'
import GaugeChart from '@/components/GaugeChart.vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Boxes, Globe, Monitor, Server as ServerIcon } from '@lucide/vue'
import { useServersStore } from '@/stores'
import { getAuth, setAuth, validateSession } from '@/lib/api'
import { useRouter } from 'vue-router'

const servers = useServersStore()

// ── Real host metrics via WebSocket /ws/analytic ─────────────────────────
// Backend runs a 1 Hz gopsutil worker; this socket connects on mount and
// closes on unmount (lifetime == time spent on the dashboard page).
interface SysMem {
  used: number
  total: number
}
interface SysDisk {
  used: number
  total: number
}
interface SysNet {
  rx: number
  tx: number
}
interface SystemInfo {
  cpu: number
  cores: number
  mem: SysMem
  disk: SysDisk
  net: SysNet
  uptime: number
  load: [number, number, number]
  host: string
  os: string
  ipLocal: string
  ipPublic: string
}

const sys = ref<SystemInfo | null>(null)
const wsStatus = ref<'connecting' | 'live' | 'down'>('connecting')
let ws: WebSocket | null = null
let wsOpened = false
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0

const cpuHistory = ref<number[]>(Array.from({ length: 60 }, () => 0))
const MAX_POINTS = 60

function connectAnalyticWS() {
  const auth = getAuth()
  if (!auth?.token) return
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/ws/analytic?token=${encodeURIComponent(auth.token)}`

  wsStatus.value = 'connecting'
  ws = new WebSocket(url)

  ws.onopen = () => {
    wsStatus.value = 'live'
    wsOpened = true
    reconnectAttempts = 0
  }

  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data) as SystemInfo
      sys.value = data
      cpuHistory.value = [...cpuHistory.value.slice(1), data.cpu]
    } catch {
      /* ignore malformed frame */
    }
  }

  ws.onclose = (ev) => {
    wsStatus.value = 'down'
    // close code 4001 = auth rejected (backend sends this when token invalid)
    if (ev.code === 4001) {
      redirectToLogin('ws-4001')
      return
    }
    // if the socket died before ever opening, re-validate the session —
    // a dead token now bounces to /login instead of looping offline
    if (!wsOpened) {
      // never bounce right after a fresh login — the WS handshake may have
      // raced with setAuth (token not yet visible). Just reconnect.
      if ((window as any).__golify_just_logged_in__) {
        scheduleReconnect()
        return
      }
      validateSession().then((ok) => {
        if (!ok) redirectToLogin('ws-never-opened')
        else scheduleReconnect()
      })
      return
    }
    scheduleReconnect()
  }
  ws.onerror = () => {
    ws?.close()
  }
}

function scheduleReconnect() {
  if (reconnectTimer) return
  const delay = Math.min(1000 * 2 ** reconnectAttempts, 15000)
  reconnectAttempts += 1
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    connectAnalyticWS()
  }, delay)
}

const router = useRouter()

// Token invalid (expired / backend restarted with fresh key) → clear session
// and bounce to /login instead of showing a dead "Offline" dashboard.
function redirectToLogin(reason = 'unknown') {
  const hadToken = !!getAuth()
  const justLoggedIn = !!(window as any).__golify_just_logged_in__
  // Never wipe a session that was JUST created (login flow in progress —
  // races between WS close / guard / validate can fire before setAuth
  // finishes). The router guard will handle the redirect correctly.
  if (!justLoggedIn) {
    setAuth(null)
  }
  // debug: report why we're bouncing (only when a token existed — otherwise
  // this would fire on every anonymous /login visit)
  if (hadToken || justLoggedIn) {
    fetch('/api/report/error', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        app_name: 'Golify Dashboard',
        app_url: window.location.href,
        title: `Bounce to login (${reason})`,
        stack: new Error().stack || '',
      }),
    }).catch(() => {})
  }
  router.replace('/login')
}

onMounted(async () => {
  // The router guard already validated the session before this view mounts.
  // No extra /auth/me check here — it has repeatedly raced with the login
  // flow (setAuth finishing a tick later) and bounced fresh sessions.
  // If the token is dead, the WS handshake gets 401 → ws-4001 → redirect.
  connectAnalyticWS()
  fetchContainerCount()
})

onBeforeUnmount(() => {
  // close the socket when leaving the dashboard page
  if (reconnectTimer) clearTimeout(reconnectTimer)
  ws?.close()
  ws = null
})

// ── Computed metrics (from real WS data, fallback to servers store) ──────
const memPercent = computed(() => {
  if (sys.value?.mem.total) return Math.round((sys.value.mem.used / sys.value.mem.total) * 100)
  const used = servers.servers.reduce((a, s) => a + s.memory, 0)
  const total = servers.servers.reduce((a, s) => a + s.memoryTotal, 0)
  return total ? Math.round((used / total) * 100) : 0
})
const diskPercent = computed(() => {
  if (sys.value?.disk.total) return Math.round((sys.value.disk.used / sys.value.disk.total) * 100)
  return 35
})
const cpuPercent = computed(() => {
  if (sys.value) return Math.round(sys.value.cpu)
  const online = servers.servers.filter((s) => s.status === 'online')
  if (!online.length) return 0
  return Math.round(online.reduce((a, s) => a + s.cpu, 0) / online.length)
})
const uptimeLabel = computed(() => {
  const up = sys.value?.uptime ?? 0
  const d = Math.floor(up / 86400)
  const h = Math.floor((up % 86400) / 3600)
  const m = Math.floor((up % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
})
const netLabel = computed(() => {
  if (!sys.value) return '—'
  const rx = sys.value.net.rx
  const tx = sys.value.net.tx
  const fmt = (b: number) => {
    if (b > 1024 ** 3) return `${(b / 1024 ** 3).toFixed(1)} GB`
    if (b > 1024 ** 2) return `${(b / 1024 ** 2).toFixed(1)} MB`
    return `${(b / 1024).toFixed(0)} KB`
  }
  return `↓ ${fmt(rx)} · ↑ ${fmt(tx)}`
})

// ── Charts (Highcharts only) ──────────────────────────────────────────────
// Gauges: reusable GaugeChart.vue (Highcharts SVG speedometer).
// Sparkline: Highcharts area chart — updates via ref, no canvas.
const sparklineRef = ref<HTMLDivElement | null>(null)
let sparkChart: Highcharts.Chart | null = null

const sparkOptions: Highcharts.Options = {
  chart: { type: 'area', height: 140, backgroundColor: 'transparent', spacing: [4, 4, 4, 4] },
  title: { text: undefined },
  credits: { enabled: false },
  legend: { enabled: false },
  tooltip: { enabled: false },
  xAxis: { visible: false },
  yAxis: { visible: false, min: 0, max: 100 },
  plotOptions: {
    area: {
      lineColor: '#58a6ff',
      lineWidth: 2,
      fillColor: {
        linearGradient: { x1: 0, y1: 0, x2: 0, y2: 1 },
        stops: [
          [0, 'rgba(88, 166, 255, 0.35)'],
          [1, 'rgba(88, 166, 255, 0.02)'],
        ],
      },
      marker: { enabled: false },
      animation: false,
    },
  },
  series: [{ type: 'area', name: 'CPU', data: [] }],
}

onMounted(() => {
  if (sparklineRef.value) {
    sparkChart = Highcharts.chart(sparklineRef.value, sparkOptions)
  }
})

// push updates into the existing chart (redraw:true — otherwise the path
// stays stale because we disabled chart-level animation)
watch(
  () => cpuHistory.value,
  (hist) => {
    if (sparkChart) {
      sparkChart.series[0]?.setData(hist.slice(), true)
    }
  },
  { deep: true },
)

// recent activity mock (kept from existing design)
const recent = ref([
  { id: 1, kind: 'deploy', text: 'sawang.tech-website · production · deployed', time: '5m ago' },
  { id: 2, kind: 'build', text: 'hindsight-agent-memory · production · building', time: '12m ago' },
  { id: 3, kind: 'key', text: 'New SSH key added: deploy-key-vps2', time: '1h ago' },
  { id: 4, kind: 'source', text: 'Source connected: gitea-self-hosted', time: '2h ago' },
  { id: 5, kind: 'backup', text: 'Backup uploaded to sawang-backups (1.4 GB)', time: '6h ago' },
])

// ── Labels: actual values (not %) ─────────────────────────────────────────
// CPU: total logical cores. Memory/Disk: free/total in GB (MB if < 1 GB).
const fmtSize = (gb: number): string => {
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${Math.round(gb * 1024)} MB`
}

const cpuTotalLabel = computed(() => {
  if (sys.value?.cores) return `${sys.value.cores} cores`
  return '—'
})

// Total container (podman/docker) — fetched from /api/v1/system/containers
const containerCount = ref<number | null>(null)
const containerRuntime = ref('')

async function fetchContainerCount() {
  try {
    const auth = getAuth()
    const res = await fetch('/api/v1/system/containers', {
      headers: auth?.token ? { Authorization: `Bearer ${auth.token}` } : {},
    })
    if (res.ok) {
      const d = await res.json()
      containerCount.value = d.count
      containerRuntime.value = d.runtime
    }
  } catch {
    /* non-fatal */
  }
}

const containerLabel = computed(() => {
  if (containerCount.value === null) return '…'
  return `${containerCount.value}`
})

const hostLabel = computed(() => {
  if (!sys.value?.host) return '…'
  const ip = sys.value?.ipLocal
  return ip ? `${sys.value.host} / ${ip}` : sys.value.host
})

const pubIpLabel = computed(() => sys.value?.ipPublic || '…')

interface StatItem {
  label: string
  value: string
  sub?: string
  icon: Component
}

const stats = computed<StatItem[]>(() => [
  {
    label: 'Total Container',
    value: containerLabel.value,
    sub: containerRuntime.value,
    icon: Boxes,
  },
  {
    label: 'OS',
    value: sys.value?.os || '…',
    sub: sys.value?.host ? undefined : 'Loading host info…',
    icon: Monitor,
  },
  { label: 'Host', value: hostLabel.value, icon: ServerIcon },
  { label: 'IP Public', value: pubIpLabel.value, icon: Globe },
])

const gauges = computed(() => [
  { label: 'CPU', value: cpuPercent.value, color: '#58a6ff' },
  { label: 'Memory', value: memPercent.value, color: '#3fb950' },
  { label: 'Disk', value: diskPercent.value, color: '#d29922' },
])

const wsBadge = computed(() =>
  wsStatus.value === 'live'
    ? { text: 'Live', cls: 'bg-emerald-500/15 text-emerald-400' }
    : wsStatus.value === 'connecting'
      ? { text: 'Connecting…', cls: 'bg-amber-500/15 text-amber-400' }
      : { text: 'Offline', cls: 'bg-destructive/15 text-destructive' },
)
</script>

<template>
  <div class="grid gap-6">
    <header class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p class="text-sm text-muted-foreground">uptime {{ uptimeLabel }}</p>
      </div>
      <div class="flex items-center gap-2">
        <Badge variant="secondary" class="font-mono text-xs">{{ netLabel }}</Badge>
        <Badge :class="wsBadge.cls" class="font-mono text-xs">
          {{ wsBadge.text }}
        </Badge>
      </div>
    </header>

    <div class="grid gap-4 md:grid-cols-4">
      <Card v-for="s in stats" :key="s.label">
        <CardContent class="flex items-center justify-between p-4">
          <div>
            <p class="text-xs uppercase tracking-wider text-muted-foreground">{{ s.label }}</p>
            <p class="mt-1 text-2xl font-semibold">{{ s.value }}</p>
            <p v-if="s.sub" class="text-xs text-muted-foreground">{{ s.sub }}</p>
          </div>
          <component :is="s.icon" class="size-5 text-muted-foreground" />
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <Card v-for="g in gauges" :key="g.label">
        <CardHeader class="pb-2">
          <CardDescription>{{ g.label }}</CardDescription>
          <CardTitle class="text-2xl">
            {{
              g.label === 'CPU'
                ? cpuTotalLabel
                : `${g.value}%`
            }}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <!-- needle stays % — only the big number is actual value -->
          <GaugeChart
            :value="g.value"
            :min="0"
            :max="100"
            unit="%"
            :color="g.color"
            :plot-bands="[
              { from: 0, to: 70, color: '#3fb950' },
              { from: 70, to: 90, color: '#d29922' },
              { from: 90, to: 100, color: '#f85149' },
            ]"
          />
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>CPU activity (live)</CardTitle>
        <CardDescription>Real-time CPU %, updated every second over WebSocket.</CardDescription>
      </CardHeader>
      <CardContent>
        <div ref="sparklineRef" class="w-full" />
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle>Recent activity</CardTitle>
      </CardHeader>
      <CardContent>
        <ul class="divide-y divide-border">
          <li v-for="r in recent" :key="r.id" class="flex items-center justify-between py-2 text-sm">
            <span class="flex items-center gap-2">
              <Badge variant="outline" class="font-mono text-xs">{{ r.kind }}</Badge>
              <span>{{ r.text }}</span>
            </span>
            <span class="text-xs text-muted-foreground">{{ r.time }}</span>
          </li>
        </ul>
      </CardContent>
    </Card>
  </div>
</template>
