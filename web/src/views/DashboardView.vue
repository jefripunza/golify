<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Doughnut, Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  ArcElement,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
} from 'chart.js'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, Cpu, HardDrive, Server as ServerIcon, Wifi } from '@lucide/vue'
import { useServersStore } from '@/stores'
import { getAuth } from '@/lib/api'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  ArcElement,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
)

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
  mem: SysMem
  disk: SysDisk
  net: SysNet
  uptime: number
  load: [number, number, number]
  host: string
  os: string
}

const sys = ref<SystemInfo | null>(null)
const wsStatus = ref<'connecting' | 'live' | 'down'>('connecting')
let ws: WebSocket | null = null
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

  ws.onclose = () => {
    wsStatus.value = 'down'
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

onMounted(() => {
  connectAnalyticWS()
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

// ── Charts (Chart.js) ─────────────────────────────────────────────────────
// maintainAspectRatio:true + aspectRatio:1 + parent aspect-square → the
// doughnut fills a perfect square box (no more elliptical/tilted gauge).
// animation:false → renders synchronously; RAF-based default animation
// never fires in some embedded/headless contexts (blank canvas).
const gaugeOptions = {
  responsive: true,
  maintainAspectRatio: true,
  aspectRatio: 1,
  animation: false,
  cutout: '70%',
  plugins: { legend: { display: false }, tooltip: { enabled: false } },
}

const sparkData = computed(() => ({
  labels: cpuHistory.value.map((_, i) => `${60 - i}s`),
  datasets: [
    {
      label: 'CPU',
      data: cpuHistory.value,
      borderColor: '#58a6ff',
      backgroundColor: 'rgba(88, 166, 255, 0.15)',
      fill: true,
      tension: 0.4,
      pointRadius: 0,
      borderWidth: 2,
    },
  ],
}))
const sparkOptions = {
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  plugins: { legend: { display: false }, tooltip: { enabled: false } },
  scales: { x: { display: false }, y: { display: false } },
}

// recent activity mock (kept from existing design)
const recent = ref([
  { id: 1, kind: 'deploy', text: 'sawang.tech-website · production · deployed', time: '5m ago' },
  { id: 2, kind: 'build', text: 'hindsight-agent-memory · production · building', time: '12m ago' },
  { id: 3, kind: 'key', text: 'New SSH key added: deploy-key-vps2', time: '1h ago' },
  { id: 4, kind: 'source', text: 'Source connected: gitea-self-hosted', time: '2h ago' },
  { id: 5, kind: 'backup', text: 'Backup uploaded to sawang-backups (1.4 GB)', time: '6h ago' },
])

const stats = computed(() => [
  { label: 'Host', value: sys.value?.host ?? '…', icon: ServerIcon },
  { label: 'CPU', value: `${cpuPercent.value}%`, icon: Cpu },
  { label: 'Memory', value: `${memPercent.value}%`, icon: Activity },
  { label: 'Disk', value: `${diskPercent.value}%`, icon: HardDrive },
])

const gauges = computed(() => [
  { label: 'CPU', value: cpuPercent.value, color: '#58a6ff' },
  { label: 'Memory', value: memPercent.value, color: '#3fb950' },
  { label: 'Disk', value: diskPercent.value, color: '#d29922' },
])

// stable per-gauge chart data — computed, NOT a function call in the template.
// vue-chartjs watches :data/:options props; a fresh object every render makes
// it destroy+recreate the chart on every tick (2s WS) → empty canvas.
const gaugeChartDatasets = computed(() =>
  gauges.value.map((g) => ({
    labels: ['used', 'free'],
    datasets: [
      {
        data: [g.value, 100 - g.value],
        backgroundColor: [g.color, '#21262d'],
        borderWidth: 0,
        circumference: 270,
        rotation: -225,
      },
    ],
  })),
)

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
        <p class="text-sm text-muted-foreground">
          {{ sys?.os ?? 'Loading host info…' }} · uptime {{ uptimeLabel }}
        </p>
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
          </div>
          <component :is="s.icon" class="size-5 text-muted-foreground" />
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <Card v-for="(g, i) in gauges" :key="g.label">
        <CardHeader class="pb-2">
          <CardDescription>{{ g.label }}</CardDescription>
          <CardTitle class="text-3xl">{{ g.value }}%</CardTitle>
        </CardHeader>
        <CardContent>
          <!-- aspect-square keeps the doughnut a perfect circle -->
          <div class="relative aspect-square w-full max-w-[200px] mx-auto">
            <Doughnut :data="gaugeChartDatasets[i]" :options="gaugeOptions" />
          </div>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>CPU activity (live)</CardTitle>
        <CardDescription>Real-time CPU %, updated every second over WebSocket.</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="relative h-[160px]">
          <Line :data="sparkData" :options="sparkOptions" />
        </div>
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
