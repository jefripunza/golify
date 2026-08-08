<script setup lang="ts">
import { computed, ref } from 'vue'
import Highcharts from 'highcharts'
import { Chart } from 'highcharts-vue'
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

// sparkline (mock timeseries)
const series = Array.from({ length: 24 }, (_, i) => Math.round(20 + Math.random() * 40))
const cpuOptions = computed(() => ({
  chart: { type: 'areaspline', height: 80, backgroundColor: 'transparent' },
  title: { text: undefined },
  xAxis: { visible: false },
  yAxis: { visible: false },
  legend: { enabled: false },
  credits: { enabled: false },
  plotOptions: { areaspline: { marker: { enabled: false } } },
  series: [{ name: 'cpu', data: series, color: '#58a6ff', fillOpacity: 0.2 }],
}))

const gaugeOpts = (value: number, color: string) => ({
  chart: { type: 'solidgauge', height: 140, backgroundColor: 'transparent' },
  title: { text: undefined },
  pane: {
    center: ['50%', '85%'],
    size: '140%',
    startAngle: -90,
    endAngle: 90,
    background: [{ backgroundColor: '#21262d', innerRadius: '60%', outerRadius: '100%', shape: 'arc', borderWidth: 0 }],
  },
  tooltip: { enabled: false },
  yAxis: {
    min: 0,
    max: 100,
    lineWidth: 0,
    tickWidth: 0,
    minorTickInterval: null,
    tickAmount: 0,
    title: { text: undefined },
    labels: { enabled: false },
  },
  credits: { enabled: false },
  plotOptions: {
    solidgauge: {
      dataLabels: { enabled: false },
      innerRadius: '60%',
    },
  },
  series: [{
    name: 'value',
    data: [value],
    dataLabels: { enabled: false },
    color,
  }],
})

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
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>CPU</CardDescription>
          <CardTitle class="text-3xl">{{ totalCpu }}%</CardTitle>
        </CardHeader>
        <CardContent>
          <ClientOnly>
            <Chart :options="gaugeOpts(totalCpu, '#58a6ff')" />
          </ClientOnly>
        </CardContent>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Memory</CardDescription>
          <CardTitle class="text-3xl">{{ totalMem }}%</CardTitle>
        </CardHeader>
        <CardContent>
          <ClientOnly>
            <Chart :options="gaugeOpts(totalMem, '#3fb950')" />
          </ClientOnly>
        </CardContent>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Disk</CardDescription>
          <CardTitle class="text-3xl">{{ totalDisk }}%</CardTitle>
        </CardHeader>
        <CardContent>
          <ClientOnly>
            <Chart :options="gaugeOpts(totalDisk, '#d29922')" />
          </ClientOnly>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>CPU activity (last 24h)</CardTitle>
        <CardDescription>Sample timeseries, mocked.</CardDescription>
      </CardHeader>
      <CardContent>
        <ClientOnly>
          <Chart :options="cpuOptions" />
        </ClientOnly>
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
