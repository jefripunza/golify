<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useProjectsStore } from '@/stores'
import { getAuth } from '@/lib/api'
import {
  Play,
  Square,
  RotateCw,
  Box,
  Layers,
  FolderTree,
  ScrollText,
  Activity,
} from '@lucide/vue'

const route = useRoute()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const envId = computed(() => String(route.params.envId))
const serviceId = computed(() => String(route.params.serviceId))
const project = computed(() => store.get(projectId.value))
const env = computed(() => store.getEnv(projectId.value, envId.value))
const service = computed(() => store.getService(projectId.value, envId.value, serviceId.value))

const termEl = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fit: FitAddon | null = null

const logs = ref<string[]>([
  `[${new Date().toISOString()}] container ${service.value?.name ?? ''} starting…`,
  `[${new Date().toISOString()}] mounting volumes…`,
  `[${new Date().toISOString()}] healthcheck ok`,
])

function statusVariant(s?: string) {
  switch (s) {
    case 'running': return 'default'
    case 'stopped': return 'secondary'
    case 'building':
    case 'deploying': return 'outline'
    case 'error': return 'destructive'
    default: return 'secondary'
  }
}

function appendLog(line: string) {
  logs.value.push(line)
  if (logs.value.length > 200) logs.value.shift()
}

onMounted(() => {
  if (!termEl.value) return
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    theme: {
      background: '#0e1117',
      foreground: '#e6edf3',
      cursor: '#58a6ff',
      selectionBackground: '#21262d',
    },
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(termEl.value)
  fit.fit()

  // connect to the real WS exec endpoint (same-origin — dev: vite proxies /ws → :20004)
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/ws/exec?token=${encodeURIComponent(auth?.token ?? '')}&service_id=${encodeURIComponent(serviceId.value)}`
  try {
    ws = new WebSocket(url)
  } catch {
    ws = null
  }
  if (ws) {
    ws.onopen = () => {
      term?.writeln('\x1b[36m─── golify terminal ───\x1b[0m')
      term?.writeln(`connected to service: ${service.value?.name ?? serviceId.value} via WS`)
      term?.writeln('')
    }
    ws.onmessage = (ev) => term?.write(String(ev.data))
    ws.onclose = () => {
      term?.writeln('\r\n\x1b[31m[connection closed]\x1b[0m')
    }
    ws.onerror = () => {
      term?.writeln('\r\n\x1b[31m[websocket error]\x1b[0m')
    }
  } else {
    term.writeln('\x1b[31mWS not available — check server :20004\x1b[0m')
  }
  term.onData((data) => {
    ws?.send(data)
  })
})

let ws: WebSocket | null = null

watch(termEl, () => {
  setTimeout(() => fit?.fit(), 0)
})

onBeforeUnmount(() => {
  ws?.close()
  ws = null
  term?.dispose()
  term = null
  fit = null
})

function action(a: 'start' | 'stop' | 'restart') {
  if (!project.value || !env.value || !service.value) return
  if (a === 'start') store.start(project.value.id, env.value.id, service.value.id)
  if (a === 'stop') store.stop(project.value.id, env.value.id, service.value.id)
  if (a === 'restart') {
    store.stop(project.value.id, env.value.id, service.value.id)
    setTimeout(() => store.start(project.value.id!, env.value!.id, service.value!.id), 400)
  }
  appendLog(`[${new Date().toISOString()}] action: ${a}`)
}
</script>

<template>
  <div v-if="!project || !env || !service" class="text-sm text-muted-foreground">
    Service not found.
  </div>
  <div v-else class="grid gap-4">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">Projects</RouterLink>
      <span>/</span>
      <RouterLink :to="`/projects/${project.id}`" class="hover:text-foreground">
        <FolderTree class="inline size-3" /> {{ project.name }}
      </RouterLink>
      <span>/</span>
      <RouterLink :to="`/projects/${project.id}/${env.id}`" class="hover:text-foreground">
        <Layers class="inline size-3" /> {{ env.name }}
      </RouterLink>
      <span>/</span>
      <span class="flex items-center gap-1">
        <Box class="inline size-3" /> {{ service.name }}
      </span>
    </div>

    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">{{ service.name }}</h1>
        <p class="font-mono text-xs text-muted-foreground">
          {{ service.kind }} · {{ service.image || service.composePath || '—' }}
        </p>
      </div>
      <Badge :variant="statusVariant(service.status)">{{ service.status }}</Badge>
    </header>

    <div class="flex flex-wrap gap-2">
      <Button size="sm" :disabled="service.status === 'running'" @click="action('start')">
        <Play class="mr-1 size-4" />Start
      </Button>
      <Button size="sm" variant="outline" :disabled="service.status === 'stopped'" @click="action('stop')">
        <Square class="mr-1 size-4" />Stop
      </Button>
      <Button size="sm" variant="outline" @click="action('restart')">
        <RotateCw class="mr-1 size-4" />Restart
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-3">
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>CPU</CardDescription>
          <CardTitle class="text-2xl">{{ service.cpu }}%</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Memory</CardDescription>
          <CardTitle class="text-2xl">{{ service.memory }} MB</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription>Ports</CardDescription>
          <CardTitle class="font-mono text-sm">
            {{ service.ports.join(', ') || '—' }}
          </CardTitle>
        </CardHeader>
      </Card>
    </div>

    <Tabs default-value="terminal" class="w-full">
      <TabsList>
        <TabsTrigger value="terminal"><Box class="mr-1 size-4" />Terminal</TabsTrigger>
        <TabsTrigger value="logs"><ScrollText class="mr-1 size-4" />Logs</TabsTrigger>
        <TabsTrigger value="metrics"><Activity class="mr-1 size-4" />Metrics</TabsTrigger>
      </TabsList>
      <TabsContent value="terminal">
        <Card>
          <CardContent class="p-2">
            <div ref="termEl" class="h-80 rounded bg-[#0e1117]" />
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="logs">
        <Card>
          <CardHeader>
            <CardTitle>Logs</CardTitle>
            <CardDescription>Last {{ logs.length }} lines.</CardDescription>
          </CardHeader>
          <CardContent>
            <pre class="max-h-80 overflow-auto rounded bg-muted p-3 font-mono text-xs leading-relaxed"><code>{{ logs.join('\n') }}</code></pre>
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="metrics">
        <Card>
          <CardHeader>
            <CardTitle>Metrics</CardTitle>
            <CardDescription>Plug a Prometheus/VictoriaMetrics scrape here.</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="grid grid-cols-2 gap-2 text-sm">
              <div class="rounded-md bg-muted p-3">
                <p class="text-xs uppercase text-muted-foreground">CPU throttling</p>
                <p class="mt-1 font-mono">0.5%</p>
              </div>
              <div class="rounded-md bg-muted p-3">
                <p class="text-xs uppercase text-muted-foreground">Network RX</p>
                <p class="mt-1 font-mono">12.4 MB/s</p>
              </div>
              <div class="rounded-md bg-muted p-3">
                <p class="text-xs uppercase text-muted-foreground">Network TX</p>
                <p class="mt-1 font-mono">4.8 MB/s</p>
              </div>
              <div class="rounded-md bg-muted p-3">
                <p class="text-xs uppercase text-muted-foreground">IO read</p>
                <p class="mt-1 font-mono">3.2 MB/s</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>
