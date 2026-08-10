<script setup lang="ts">
import { computed, ref } from 'vue'
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
import { Activity, Cpu, HardDrive, Server as ServerIcon } from '@lucide/vue'
import { useServersStore } from '@/stores'

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

// sparkline (mock timeseries) — Chart.js line chart
const series = Array.from({ length: 24 }, (_, i) => Math.round(20 + Math.random() * 40))
const sparkData = computed(() => ({
  labels: series.map((_, i) => `${i}h`),
  datasets: [
    {
      label: 'CPU',
      data: series,
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
  plugins: { legend: { display: false }, tooltip: { enabled: false } },
  scales: { x: { display: false }, y: { display: false } },
}

// gauge — Chart.js doughnut (arc gauge)
const gaugeChartData = (value: number, color: string) => ({
  labels: ['used', 'free'],
  datasets: [
    {
      data: [value, 100 - value],
      backgroundColor: [color, '#21262d'],
      borderWidth: 0,
      circumference: 270,
      rotation: -225,
    },
  ],
})
const gaugeOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '70%',
  plugins: { legend: { display: false }, tooltip: { enabled: false } },
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
          <div class="relative h-[140px]">
            <Doughnut :data="gaugeChartData(g.value, g.color)" :options="gaugeOptions" />
          </div>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>CPU activity (last 24h)</CardTitle>
        <CardDescription>Sample timeseries, mocked.</CardDescription>
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
