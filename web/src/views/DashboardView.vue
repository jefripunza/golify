<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, Cpu, HardDrive, Server as ServerIcon } from '@lucide/vue'
import { useServersStore } from '@/stores'

const servers = useServersStore()

const totalCpu = computed(() => {
  const online = servers.servers.filter((s) => s.status === 'online')
  if (!online.length) return 0
  return Math.round(online.reduce((a, s) => a + s.cpu, 0) / online.length)
})
const totalMem = computed(() => {
  const used = servers.servers.reduce((a, s) => a + s.memory, 0)
  const total = servers.servers.reduce((a, s) => a + s.memoryTotal, 0)
  return total ? Math.round((used / total) * 100) : 0
})
const totalDisk = computed(() => {
  const online = servers.servers.filter((s) => s.status === 'online')
  if (!online.length) return 0
  return Math.round(online.reduce((a, s) => a + s.disk, 0) / online.length)
})

// sparkline (mock timeseries) — pure SVG, no chart lib
const series = Array.from({ length: 24 }, (_, i) => Math.round(20 + Math.random() * 40))
const sparkPoints = computed(() => {
  const max = Math.max(...series)
  const min = Math.min(...series)
  const w = 280
  const h = 64
  const step = w / (series.length - 1)
  return series
    .map((v, i) => {
      const x = i * step
      const y = h - ((v - min) / (max - min || 1)) * (h - 8) - 4
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
})
const sparkArea = computed(() => `0,64 ${sparkPoints.value} 280,64`)

// gauge — pure SVG arc, no chart lib
const gaugePath = (value: number) => {
  const r = 44
  const cx = 60
  const cy = 60
  const start = Math.PI * 0.75 // 135deg (bottom-left)
  const end = Math.PI * 0.25 // 45deg (bottom-right)
  const total = end - start + Math.PI * 2
  const frac = Math.min(1, Math.max(0, value / 100))
  const a = start + total * frac
  const large = a - start > Math.PI ? 1 : 0
  const x = cx + r * Math.cos(a)
  const y = cy + r * Math.sin(a)
  const x0 = cx + r * Math.cos(start)
  const y0 = cy + r * Math.sin(start)
  return `M ${x0} ${y0} A ${r} ${r} 0 ${large} 1 ${x} ${y}`
}
const gaugeBg = () => {
  const r = 44
  const cx = 60
  const cy = 60
  const start = Math.PI * 0.75
  const end = Math.PI * 0.25
  const x0 = cx + r * Math.cos(start)
  const y0 = cy + r * Math.sin(start)
  const x1 = cx + r * Math.cos(end)
  const y1 = cy + r * Math.sin(end)
  return `M ${x0} ${y0} A ${r} ${r} 0 1 1 ${x1} ${y1}`
}

// recent activity mock
const recent = ref([
  { id: 1, kind: 'deploy', text: 'sawang.tech-website · production · deployed', time: '5m ago' },
  { id: 2, kind: 'build', text: 'hindsight-agent-memory · production · building', time: '12m ago' },
  { id: 3, kind: 'key', text: 'New SSH key added: deploy-key-vps2', time: '1h ago' },
  { id: 4, kind: 'source', text: 'Source connected: gitea-self-hosted', time: '2h ago' },
  { id: 5, kind: 'backup', text: 'Backup uploaded to sawang-backups (1.4 GB)', time: '6h ago' },
])

const stats = computed(() => [
  { label: 'Servers online', value: `${servers.onlineCount}/${servers.servers.length}`, icon: ServerIcon },
  { label: 'Avg CPU', value: `${totalCpu.value}%`, icon: Cpu },
  { label: 'Avg Memory', value: `${totalMem.value}%`, icon: Activity },
  { label: 'Avg Disk', value: `${totalDisk.value}%`, icon: HardDrive },
])

const gauges = computed(() => [
  { label: 'CPU', value: totalCpu.value, color: '#58a6ff' },
  { label: 'Memory', value: totalMem.value, color: '#3fb950' },
  { label: 'Disk', value: totalDisk.value, color: '#d29922' },
])
</script>

<template>
  <div class="grid gap-6">
    <header class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p class="text-sm text-muted-foreground">Overview of resources and recent activity.</p>
      </div>
      <Badge variant="secondary">Mock data — BE is still notif server</Badge>
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
      <Card v-for="g in gauges" :key="g.label">
        <CardHeader class="pb-2">
          <CardDescription>{{ g.label }}</CardDescription>
          <CardTitle class="text-3xl">{{ g.value }}%</CardTitle>
        </CardHeader>
        <CardContent>
          <svg viewBox="0 0 120 70" class="w-full max-w-[200px]">
            <path :d="gaugeBg()" fill="none" stroke="#21262d" stroke-width="10" stroke-linecap="round" />
            <path
              :d="gaugePath(g.value)"
              fill="none"
              :stroke="g.color"
              stroke-width="10"
              stroke-linecap="round"
            />
          </svg>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>CPU activity (last 24h)</CardTitle>
        <CardDescription>Sample timeseries, mocked.</CardDescription>
      </CardHeader>
      <CardContent>
        <svg viewBox="0 0 280 64" class="w-full">
          <polygon :points="sparkArea" fill="#58a6ff" fill-opacity="0.15" />
          <polyline
            :points="sparkPoints"
            fill="none"
            stroke="#58a6ff"
            stroke-width="2"
            stroke-linejoin="round"
            stroke-linecap="round"
          />
        </svg>
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
