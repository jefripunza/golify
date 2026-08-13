<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useProjectsStore } from '@/stores'
import { getAuth } from '@/lib/api'
import type { Service } from '@/lib/types'
import {
  Play,
  Square,
  RotateCw,
  Box,
  Layers,
  FolderTree,
  ScrollText,
  Activity,
  Globe,
  Plus,
  Trash2,
  Save,
  Loader2,
  Info,
} from '@lucide/vue'

const route = useRoute()
const router = useRouter()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const envId = computed(() => String(route.params.envId))
const serviceId = computed(() => String(route.params.serviceId))
const project = computed(() => store.get(projectId.value))
const env = computed(() => store.getEnv(projectId.value, envId.value))
const service = computed<Service | null>(() => store.getService(projectId.value, envId.value, serviceId.value) ?? null)

// ─── Left sidebar sections (Coolify-style) ────────────────────────────────
const sections = [
  { id: 'general', label: 'General', icon: 'settings' },
  { id: 'advanced', label: 'Advanced', icon: 'sliders' },
  { id: 'envvars', label: 'Environment Variables', icon: 'vars' },
  { id: 'storage', label: 'Persistent Storage', icon: 'disk' },
  { id: 'servers', label: 'Servers', icon: 'server' },
  { id: 'scheduled', label: 'Scheduled Tasks', icon: 'clock' },
  { id: 'webhooks', label: 'Webhooks', icon: 'webhook' },
  { id: 'preview', label: 'Preview Deployments', icon: 'eye' },
  { id: 'healthcheck', label: 'Healthcheck', icon: 'heart' },
  { id: 'rollback', label: 'Rollback', icon: 'undo' },
  { id: 'limits', label: 'Resource Limits', icon: 'gauge' },
  { id: 'resops', label: 'Resource Operations', icon: 'activity' },
  { id: 'metrics', label: 'Metrics', icon: 'chart' },
  { id: 'tags', label: 'Tags', icon: 'tag' },
  { id: 'danger', label: 'Danger Zone', icon: 'alert' },
]
const activeSection = ref('general')

// ─── General form state ───────────────────────────────────────────────────
const form = reactive({
  name: '',
  description: '',
  image: '',
  imageTag: '',
  dockerOptions: '',
  portsExposes: '',
  portMappings: '',
  networkAliases: '',
  basicAuthEnable: false,
  basicAuthUser: '',
  basicAuthPass: '',
})
const saving = ref(false)
const saveError = ref('')
const saveOk = ref(false)

function initForm() {
  const s = service.value
  if (!s) return
  form.name = s.name
  form.description = s.description ?? ''
  form.image = s.image ?? ''
  form.imageTag = s.imageTag ?? 'latest'
  form.dockerOptions = s.dockerOptions ?? ''
  form.portsExposes = s.portsExposes ?? ''
  form.portMappings = (s.portMappings ?? []).join(', ')
  form.networkAliases = (s.networkAliases ?? []).join(', ')
  form.basicAuthEnable = s.basicAuthEnable ?? false
  form.basicAuthUser = s.basicAuthUser ?? ''
  form.basicAuthPass = s.basicAuthPass ?? ''
}
watch(() => service.value?.id, () => { if (service.value) initForm() }, { immediate: true })

async function saveGeneral() {
  saving.value = true
  saveError.value = ''
  saveOk.value = false
  try {
    await store.updateService(projectId.value, envId.value, serviceId.value, {
      name: form.name,
      description: form.description,
      image: form.image,
      image_tag: form.imageTag,
      docker_options: form.dockerOptions,
      ports_exposes: form.portsExposes,
      port_mappings: form.portMappings.split(',').map((s) => s.trim()).filter(Boolean),
      network_aliases: form.networkAliases.split(',').map((s) => s.trim()).filter(Boolean),
      basic_auth_enable: form.basicAuthEnable,
      basic_auth_user: form.basicAuthUser,
      basic_auth_pass: form.basicAuthPass,
    })
    saveOk.value = true
    setTimeout(() => (saveOk.value = false), 2500)
  } catch (e: any) {
    saveError.value = e?.message || 'Failed to save configuration'
  } finally {
    saving.value = false
  }
}

// ─── Service domains (many domains/subdomains per service, each → port) ───
const newDomain = reactive({ host: '', port: '80' })
const domainError = ref('')
async function addDomain() {
  if (!newDomain.host.trim()) return
  domainError.value = ''
  try {
    await store.addServiceDomain(projectId.value, envId.value, serviceId.value, newDomain.host.trim(), newDomain.port.trim() || '80')
    newDomain.host = ''
    newDomain.port = '80'
  } catch (e: any) {
    domainError.value = e?.message || 'Failed to add domain'
  }
}
async function removeDomain(did: string) {
  try {
    await store.removeServiceDomain(projectId.value, envId.value, serviceId.value, did)
  } catch (e: any) {
    domainError.value = e?.message || 'Failed to remove domain'
  }
}

// ─── Status + actions ─────────────────────────────────────────────────────
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
function action(a: 'start' | 'stop' | 'restart') {
  if (!service.value) return
  if (a === 'start') store.start(projectId.value, envId.value, serviceId.value)
  if (a === 'stop') store.stop(projectId.value, envId.value, serviceId.value)
  if (a === 'restart') {
    store.stop(projectId.value, envId.value, serviceId.value)
    setTimeout(() => store.start(projectId.value, envId.value, serviceId.value), 400)
  }
}

// ─── Terminal (kept from previous design) ─────────────────────────────────
const termEl = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fit: FitAddon | null = null
let ws: WebSocket | null = null

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
  if (service.value?.status === 'stopped') {
    term.writeln('\x1b[33mService is stopped — start it to open a terminal.\x1b[0m')
    return
  }
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/api/ws/exec?token=${encodeURIComponent(auth?.token ?? '')}&service_id=${encodeURIComponent(serviceId.value)}`
  try { ws = new WebSocket(url) } catch { ws = null }
  if (ws) {
    ws.onopen = () => {
      term?.writeln('\x1b[36m─── golify terminal ───\x1b[0m')
      term?.writeln(`connected to service: ${service.value?.name ?? serviceId.value} via WS`)
      term?.writeln('')
    }
    ws.onmessage = (ev) => term?.write(String(ev.data))
    ws.onclose = () => term?.writeln('\r\n\x1b[31m[connection closed]\x1b[0m')
    ws.onerror = () => term?.writeln('\r\n\x1b[31m[websocket error]\x1b[0m')
  } else {
    term.writeln('\x1b[31mWS not available\x1b[0m')
  }
  term.onData((data) => ws?.send(data))
})
watch(termEl, () => setTimeout(() => fit?.fit(), 0))
onBeforeUnmount(() => {
  ws?.close()
  ws = null
  term?.dispose()
  term = null
  fit = null
})

// ─── Logs (mock) ──────────────────────────────────────────────────────────
const logs = ref<string[]>([
  `[${new Date().toISOString()}] service ${service.value?.name ?? ''} — stopped`,
  'run `start` to boot the container',
])

// ─── Icons for sidebar ────────────────────────────────────────────────────
const sectionIcons: Record<string, string> = {
  settings: '⚙️', sliders: '🎚️', vars: '🔑', disk: '💾', server: '🖥️',
  clock: '🕐', webhook: '🪝', eye: '👁️', heart: '❤️', undo: '↩️',
  gauge: '📊', activity: '📈', chart: '📉', tag: '🏷️', alert: '⚠️',
}
</script>

<template>
  <div v-if="!project || !env || !service" class="text-sm text-muted-foreground">
    Service not found.
  </div>
  <div v-else class="grid gap-4">
    <!-- Breadcrumb -->
    <div class="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">
        <FolderTree class="inline size-3" /> {{ project.name }}
      </RouterLink>
      <span>/</span>
      <RouterLink :to="`/project/${project.id}/environment/${env.id}/services`" class="hover:text-foreground">
        <Layers class="inline size-3" /> {{ env.name }}
      </RouterLink>
      <span>/</span>
      <span class="flex items-center gap-1 truncate">
        <Box class="inline size-3 shrink-0" /> <span class="truncate">{{ service.name }}</span>
      </span>
    </div>

    <!-- Header: title + status + actions -->
    <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-2xl font-semibold tracking-tight">{{ service.name }}</h1>
        <p class="mt-0.5 flex items-center gap-2 text-sm">
          <span class="inline-block size-2 rounded-full" :class="service.status === 'running' ? 'bg-green-500' : 'bg-yellow-500'" />
          <span :class="service.status === 'running' ? 'text-green-500' : 'text-yellow-500'">{{ service.status }} (unknown)</span>
          <Info class="ml-1 size-3 text-muted-foreground" />
          <RotateCw class="size-3 text-yellow-500" />
        </p>
        <p class="truncate font-mono text-xs text-muted-foreground">
          {{ service.kind }} · {{ service.image || service.composePath || '—' }}:{{ service.imageTag || 'latest' }}
        </p>
      </div>
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
    </header>

    <!-- Top tabs (Coolify-style) -->
    <Tabs default-value="configuration" class="w-full">
      <TabsList class="w-full overflow-x-auto">
        <TabsTrigger value="configuration" class="flex-1"><Box class="mr-1 size-4" />Configuration</TabsTrigger>
        <TabsTrigger value="deployments" class="flex-1"><Activity class="mr-1 size-4" />Deployments</TabsTrigger>
        <TabsTrigger value="logs" class="flex-1"><ScrollText class="mr-1 size-4" />Logs</TabsTrigger>
        <TabsTrigger value="terminal" class="flex-1"><Box class="mr-1 size-4" />Terminal</TabsTrigger>
        <TabsTrigger value="links" class="flex-1"><Globe class="mr-1 size-4" />Links</TabsTrigger>
      </TabsList>

      <TabsContent value="configuration">
        <div class="grid gap-4 md:grid-cols-[220px_1fr]">
          <!-- Left sidebar -->
          <nav class="flex flex-col gap-0.5 rounded-md border bg-card p-2 text-sm">
            <button
              v-for="sec in sections"
              :key="sec.id"
              class="flex items-center gap-2 rounded px-2 py-1.5 text-left transition-colors"
              :class="activeSection === sec.id ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-muted hover:text-foreground'"
              @click="activeSection = sec.id"
            >
              <span class="text-xs">{{ sectionIcons[sec.icon] }}</span>
              <span class="truncate">{{ sec.label }}</span>
            </button>
          </nav>

          <!-- Main content -->
          <div class="min-w-0">
            <template v-if="activeSection === 'general'">
              <div class="mb-3 flex items-center justify-between gap-2">
                <div>
                  <h2 class="text-lg font-semibold">General</h2>
                  <p class="text-xs text-muted-foreground">General configuration for your application.</p>
                </div>
                <Button size="sm" :disabled="saving" @click="saveGeneral">
                  <Save v-if="!saving" class="mr-1 size-4" />
                  <Loader2 v-else class="mr-1 size-4 animate-spin" />
                  {{ saveOk ? 'Saved ✓' : 'Save' }}
                </Button>
              </div>
              <p v-if="saveError" class="mb-2 text-sm text-destructive">{{ saveError }}</p>

              <Card>
                <CardHeader class="pb-2">
                  <CardDescription>Name *</CardDescription>
                  <Input v-model="form.name" placeholder="Service name" />
                </CardHeader>
                <CardHeader class="pb-2">
                  <CardDescription>Description</CardDescription>
                  <Input v-model="form.description" placeholder="Optional description" />
                </CardHeader>
              </Card>

              <!-- Domains -->
              <Card class="mt-3">
                <CardHeader class="pb-2">
                  <CardDescription class="flex items-center gap-1">
                    Domains <Info class="size-3" />
                  </CardDescription>
                  <div class="flex flex-col gap-2 sm:flex-row">
                    <Input v-model="newDomain.host" placeholder="app.example.com" class="flex-1" />
                    <div class="flex gap-2">
                      <Input v-model="newDomain.port" placeholder="80" class="w-20" />
                      <Button size="sm" variant="outline" @click="addDomain"><Plus class="size-4" /></Button>
                    </div>
                  </div>
                  <p v-if="domainError" class="text-xs text-destructive">{{ domainError }}</p>
                </CardHeader>
                <CardContent class="grid gap-1.5">
                  <div v-for="d in service.domains ?? []" :key="d.id" class="flex items-center justify-between gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
                    <span class="truncate font-mono">{{ d.host }}</span>
                    <span class="flex items-center gap-2">
                      <Badge variant="secondary">→ :{{ d.port }}</Badge>
                      <button class="text-destructive hover:text-destructive/80" @click="removeDomain(d.id)"><Trash2 class="size-3.5" /></button>
                    </span>
                  </div>
                  <p v-if="!(service.domains ?? []).length" class="text-xs text-muted-foreground">No domains yet — each domain can point to a different port.</p>
                </CardContent>
              </Card>

              <!-- Docker Registry -->
              <Card class="mt-3">
                <CardHeader class="pb-2">
                  <CardTitle class="text-base">Docker Registry</CardTitle>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <div class="grid gap-1.5">
                    <Label>Docker Image</Label>
                    <Input v-model="form.image" placeholder="jefriherditriyanto/openclaw-for-9router" />
                  </div>
                  <div class="grid gap-1.5">
                    <Label class="flex items-center gap-1">Docker Image Tag or Hash <Info class="size-3 text-muted-foreground" /></Label>
                    <Input v-model="form.imageTag" placeholder="0.0.9" />
                  </div>
                </CardContent>
              </Card>

              <!-- Build -->
              <Card class="mt-3">
                <CardHeader class="pb-2">
                  <CardTitle class="text-base">Build</CardTitle>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <div class="grid gap-1.5">
                    <Label class="flex items-center gap-1">Custom Docker Options <Info class="size-3 text-muted-foreground" /></Label>
                    <Input v-model="form.dockerOptions" placeholder="--privileged" />
                  </div>
                </CardContent>
              </Card>

              <!-- Network -->
              <Card class="mt-3">
                <CardHeader class="pb-2">
                  <CardTitle class="text-base">Network</CardTitle>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <div class="grid gap-1.5">
                    <Label class="flex items-center gap-1">Ports Exposes <Info class="size-3 text-muted-foreground" /></Label>
                    <Input v-model="form.portsExposes" placeholder="8080" />
                  </div>
                  <div class="grid gap-1.5">
                    <Label class="flex items-center gap-1">Port Mappings <Info class="size-3 text-muted-foreground" /></Label>
                    <Input v-model="form.portMappings" placeholder="3000:3000 (comma separated)" />
                  </div>
                  <div class="grid gap-1.5">
                    <Label class="flex items-center gap-1">Network Aliases <Info class="size-3 text-muted-foreground" /></Label>
                    <Input v-model="form.networkAliases" placeholder="alias1, alias2" />
                  </div>
                </CardContent>
              </Card>

              <!-- HTTP Basic Authentication -->
              <Card class="mt-3">
                <CardHeader class="pb-2">
                  <CardTitle class="flex items-center gap-2 text-base">HTTP Basic Authentication <Info class="size-3 text-muted-foreground" /></CardTitle>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <label class="flex items-center gap-2 text-sm">
                    <input v-model="form.basicAuthEnable" type="checkbox" class="size-4 accent-primary" />
                    <span>Enable</span>
                  </label>
                  <template v-if="form.basicAuthEnable">
                    <div class="grid gap-1.5">
                      <Label>Username *</Label>
                      <Input v-model="form.basicAuthUser" placeholder="admin" />
                    </div>
                    <div class="grid gap-1.5">
                      <Label>Password *</Label>
                      <Input v-model="form.basicAuthPass" type="password" placeholder="••••••" />
                    </div>
                  </template>
                </CardContent>
              </Card>
            </template>

            <template v-else-if="activeSection === 'danger'">
              <Card>
                <CardHeader>
                  <CardTitle class="text-base text-destructive">Danger Zone</CardTitle>
                  <CardDescription>Delete this service permanently. This cannot be undone.</CardDescription>
                </CardHeader>
                <CardContent>
                  <Button variant="destructive" @click="router.push(`/project/${project.id}/environment/${env.id}/services`)">Delete service</Button>
                </CardContent>
              </Card>
            </template>

            <template v-else>
              <Card>
                <CardHeader>
                  <CardTitle class="text-base">{{ sections.find((s) => s.id === activeSection)?.label }}</CardTitle>
                  <CardDescription>Coming soon — this section will be implemented next.</CardDescription>
                </CardHeader>
              </Card>
            </template>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="deployments">
        <Card>
          <CardHeader><CardTitle>Deployments</CardTitle><CardDescription>Deployment history will appear here.</CardDescription></CardHeader>
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

      <TabsContent value="terminal">
        <Card>
          <CardContent class="p-2">
            <div ref="termEl" class="h-80 rounded bg-[#0e1117]" />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="links">
        <Card>
          <CardHeader><CardTitle>Links</CardTitle><CardDescription>Open the service's public links.</CardDescription></CardHeader>
          <CardContent class="grid gap-1.5">
            <a
              v-for="d in service.domains ?? []"
              :key="d.id"
              :href="`https://${d.host}`"
              target="_blank"
              rel="noopener"
              class="flex items-center gap-2 rounded-md bg-muted px-3 py-2 text-sm text-primary hover:underline"
            >
              <Globe class="size-4" /> {{ d.host }} → :{{ d.port }}
            </a>
            <p v-if="!(service.domains ?? []).length" class="text-xs text-muted-foreground">No domains configured yet.</p>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>
