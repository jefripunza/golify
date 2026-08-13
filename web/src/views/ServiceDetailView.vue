<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
const activeTab = ref('configuration')
const topTabs = [
  { id: 'configuration', label: 'Configuration', icon: Box },
  { id: 'deployments', label: 'Deployments', icon: Activity },
  { id: 'logs', label: 'Logs', icon: ScrollText },
  { id: 'terminal', label: 'Terminal', icon: Box },
  { id: 'links', label: 'Links', icon: Globe },
]

// ─── General form state ───────────────────────────────────────────────────
const form = reactive({
  name: '',
  description: '',
  image: '',
  imageTag: 'latest',
  dockerOptions: '',
  portMappings: '',
  basicAuthEnable: false,
  basicAuthUser: '',
  basicAuthPass: '',
  replicasMode: 'fix' as 'fix' | 'range',
  replicas: 1,
  replicasMin: 1,
  replicasMax: 1,
})
const saving = ref(false)
const saveError = ref('')
const saveOk = ref(false)

// Split a docker ref like "nginx:latest" or "ghcr.io/org/app:v1" into
// image name + tag (only split at the LAST colon if it's not a registry port).
function splitImageRef(ref: string): { image: string; tag: string } {
  const r = (ref || '').trim()
  if (!r) return { image: '', tag: 'latest' }
  // Registry with port (e.g. localhost:5000/nginx) — the first segment
  // before "/" may contain ":" so only the LAST colon before a "/"-less
  // tail is a tag separator.
  const lastColon = r.lastIndexOf(':')
  const lastSlash = r.lastIndexOf('/')
  if (lastColon > lastSlash && lastColon > 0) {
    return { image: r.slice(0, lastColon), tag: r.slice(lastColon + 1) }
  }
  return { image: r, tag: 'latest' }
}

function initForm() {
  const s = service.value
  if (!s) return
  form.name = s.name
  form.description = s.description ?? ''
  const { image, tag } = splitImageRef(s.image ?? '')
  form.image = image
  form.imageTag = s.imageTag && s.imageTag !== 'latest' ? s.imageTag : tag
  form.dockerOptions = s.dockerOptions ?? ''
  form.portMappings = (s.portMappings ?? []).join(', ')
  form.basicAuthEnable = s.basicAuthEnable ?? false
  form.basicAuthUser = s.basicAuthUser ?? ''
  form.basicAuthPass = s.basicAuthPass ?? ''
  form.replicasMode = s.replicasMode === 'range' ? 'range' : 'fix'
  form.replicas = s.replicas ?? 1
  form.replicasMin = s.replicasMin ?? 1
  form.replicasMax = s.replicasMax ?? 1
}
watch(() => service.value?.id, () => { if (service.value) initForm() }, { immediate: true })

// Rebuild the full image ref from image + tag on save.
function fullImageRef(): string {
  if (!form.image.trim()) return form.image.trim()
  return `${form.image.trim()}:${form.imageTag.trim() || 'latest'}`
}

async function saveGeneral() {
  saving.value = true
  saveError.value = ''
  saveOk.value = false
  try {
    await store.updateService(projectId.value, envId.value, serviceId.value, {
      name: form.name,
      description: form.description,
      image: fullImageRef(),
      image_tag: form.imageTag.trim() || 'latest',
      docker_options: form.dockerOptions,
      port_mappings: form.portMappings.split(',').map((s) => s.trim()).filter(Boolean),
      basic_auth_enable: form.basicAuthEnable,
      basic_auth_user: form.basicAuthUser,
      basic_auth_pass: form.basicAuthPass,
      replicas_mode: form.replicasMode,
      replicas: Number(form.replicas) || 1,
      replicas_min: Number(form.replicasMin) || 1,
      replicas_max: Number(form.replicasMax) || 1,
    })
    saveOk.value = true
    setTimeout(() => (saveOk.value = false), 2500)
  } catch (e: any) {
    saveError.value = e?.message || 'Failed to save configuration'
  } finally {
    saving.value = false
  }
}

// ─── Root domains (dropdown for subdomain picker) ─────────────────────────
const rootDomains = computed(() => store.rootDomains)
const selectedRootDomain = ref('')
onMounted(() => {
  store.fetchRootDomains()
})
function availableRootDomains(): string[] {
  const roots = rootDomains.value
  if (selectedRootDomain.value && !roots.includes(selectedRootDomain.value)) {
    return [selectedRootDomain.value, ...roots]
  }
  return roots
}

// ─── Service domains (subdomain + root domain + port → host) ──────────────
const newDomain = reactive({ subdomain: '', port: '80' })
const domainError = ref('')
function buildDomainHost(): string {
  const sub = newDomain.subdomain.trim().replace(/^\.+/, '')
  const root = selectedRootDomain.value.trim().replace(/^\.+/, '')
  if (!root) return sub
  return sub ? `${sub}.${root}` : root
}
async function addDomain() {
  const host = buildDomainHost()
  if (!host) {
    domainError.value = 'Fill the subdomain or pick a root domain first.'
    return
  }
  domainError.value = ''
  try {
    await store.addServiceDomain(projectId.value, envId.value, serviceId.value, host, newDomain.port.trim() || '80')
    newDomain.subdomain = ''
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

// ─── Network port mappings (2 fields + add button) ────────────────────────
const newMapping = reactive({ from: '', to: '' })
const mappingError = ref('')
function addMapping() {
  const from = newMapping.from.trim()
  const to = newMapping.to.trim()
  if (!from) {
    mappingError.value = 'Fill the host port.'
    return
  }
  mappingError.value = ''
  const entry = to ? `${from}:${to}` : from
  const existing = form.portMappings.split(',').map((s) => s.trim()).filter(Boolean)
  if (existing.includes(entry)) {
    mappingError.value = 'Mapping already exists.'
    return
  }
  existing.push(entry)
  form.portMappings = existing.join(', ')
  newMapping.from = ''
  newMapping.to = ''
}
function removeMapping(entry: string) {
  const existing = form.portMappings.split(',').map((s) => s.trim()).filter(Boolean)
  form.portMappings = existing.filter((m) => m !== entry).join(', ')
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

// ─── Terminal (initialized lazily when the Terminal tab is opened) ────────
const termEl = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fit: FitAddon | null = null
let ws: WebSocket | null = null

function initTerminal() {
  if (!termEl.value || term) return
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
}
watch(activeTab, (tab) => {
  if (tab === 'terminal') {
    setTimeout(() => {
      initTerminal()
      fit?.fit()
    }, 0)
  } else {
    ws?.close()
    ws = null
  }
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

    <!-- Top tab bar (Coolify-style, simple buttons — no flex-1 stretch) -->
    <div class="flex flex-wrap gap-1 border-b text-sm">
      <button
        v-for="t in topTabs"
        :key="t.id"
        class="flex items-center gap-1.5 border-b-2 px-3 py-2 transition-colors"
        :class="activeTab === t.id ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = t.id"
      >
        <component :is="t.icon" class="size-4" /> {{ t.label }}
      </button>
    </div>

    <!-- Configuration: sidebar LEFT + form content -->
    <div v-if="activeTab === 'configuration'" class="grid gap-4 md:grid-cols-[220px_1fr]">
      <!-- Left sidebar -->
      <nav class="flex flex-col gap-0.5 rounded-md border bg-card p-2 text-sm md:h-fit">
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

          <div class="grid grid-cols-12 gap-3">
            <!-- Basic Info (col 6) -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardTitle class="text-base">Basic Info</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-3">
                <div class="grid gap-1.5">
                  <Label>Name *</Label>
                  <Input v-model="form.name" placeholder="Service name" />
                </div>
                <div class="grid gap-1.5">
                  <Label>Description</Label>
                  <Input v-model="form.description" placeholder="Optional description" />
                </div>
              </CardContent>
            </Card>

            <!-- Docker Registry (col 6) -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardTitle class="text-base">Docker Registry</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-3">
                <div class="grid gap-1.5">
                  <Label>Docker Image</Label>
                  <Input v-model="form.image" placeholder="nginx" />
                </div>
                <div class="grid gap-1.5">
                  <Label class="flex items-center gap-1">Docker Image Tag or Hash <Info class="size-3 text-muted-foreground" /></Label>
                  <Input v-model="form.imageTag" placeholder="latest" />
                </div>
              </CardContent>
            </Card>

            <!-- Domains (col 6) — subdomain + root domain dropdown + port -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardDescription class="flex items-center gap-1">
                  Domains <Info class="size-3" />
                </CardDescription>
                <div class="flex flex-col gap-2">
                  <div class="flex flex-col gap-2 sm:flex-row">
                    <Input v-model="newDomain.subdomain" placeholder="subdomain" class="flex-1" />
                    <select
                      v-model="selectedRootDomain"
                      class="h-9 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <option value="">— domain —</option>
                      <option v-for="d in availableRootDomains()" :key="d" :value="d">{{ d }}</option>
                    </select>
                  </div>
                  <div class="flex gap-2">
                    <Input v-model="newDomain.port" placeholder="80" class="w-20" />
                    <Button size="sm" variant="outline" @click="addDomain"><Plus class="size-4" /> Add</Button>
                  </div>
                  <p class="text-xs text-muted-foreground">
                    Preview: <span class="font-mono">{{ buildDomainHost() || '—' }}</span>
                  </p>
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

            <!-- Network (col 6) — port mapping 2 fields + add button -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardTitle class="text-base">Network</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-3">
                <Label class="flex items-center gap-1">Port Mappings <Info class="size-3 text-muted-foreground" /></Label>
                <div class="flex flex-col gap-2 sm:flex-row">
                  <Input v-model="newMapping.from" placeholder="host port (3000)" class="flex-1" />
                  <div class="flex gap-2">
                    <Input v-model="newMapping.to" placeholder="container port (3000)" class="flex-1" />
                    <Button size="sm" variant="outline" @click="addMapping"><Plus class="size-4" /> Add</Button>
                  </div>
                </div>
                <p v-if="mappingError" class="text-xs text-destructive">{{ mappingError }}</p>
                <div v-for="m in form.portMappings.split(',').map((s) => s.trim()).filter(Boolean)" :key="m" class="flex items-center justify-between gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
                  <span class="truncate font-mono">{{ m }}</span>
                  <button class="text-destructive hover:text-destructive/80" @click="removeMapping(m)"><Trash2 class="size-3.5" /></button>
                </div>
                <p v-if="!form.portMappings.split(',').map((s) => s.trim()).filter(Boolean).length" class="text-xs text-muted-foreground">No port mappings yet.</p>
              </CardContent>
            </Card>

            <!-- Build (col 12) -->
            <Card class="col-span-12">
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

            <!-- HTTP Basic Authentication (col 6) + Replicas (col 6) -->
            <Card class="col-span-12 md:col-span-6">
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

            <!-- Replicas (col 6) — fix or range -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardTitle class="text-base">Replicas</CardTitle>
              </CardHeader>
              <CardContent class="grid gap-3">
                <div class="flex gap-1 rounded-md border p-1">
                  <button
                    class="flex-1 rounded px-3 py-1.5 text-sm transition-colors"
                    :class="form.replicasMode === 'fix' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
                    @click="form.replicasMode = 'fix'"
                  >
                    Fix
                  </button>
                  <button
                    class="flex-1 rounded px-3 py-1.5 text-sm transition-colors"
                    :class="form.replicasMode === 'range' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
                    @click="form.replicasMode = 'range'"
                  >
                    Range (auto)
                  </button>
                </div>
                <template v-if="form.replicasMode === 'fix'">
                  <div class="grid gap-1.5">
                    <Label>Replicas *</Label>
                    <Input v-model.number="form.replicas" type="number" min="1" placeholder="1" />
                  </div>
                </template>
                <template v-else>
                  <div class="grid grid-cols-2 gap-3">
                    <div class="grid gap-1.5">
                      <Label>Min *</Label>
                      <Input v-model.number="form.replicasMin" type="number" min="1" placeholder="1" />
                    </div>
                    <div class="grid gap-1.5">
                      <Label>Max *</Label>
                      <Input v-model.number="form.replicasMax" type="number" min="1" placeholder="5" />
                    </div>
                  </div>
                </template>
              </CardContent>
            </Card>
          </div>
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

    <!-- Deployments -->
    <div v-else-if="activeTab === 'deployments'">
      <Card>
        <CardHeader><CardTitle>Deployments</CardTitle><CardDescription>Deployment history will appear here.</CardDescription></CardHeader>
      </Card>
    </div>

    <!-- Logs -->
    <div v-else-if="activeTab === 'logs'">
      <Card>
        <CardHeader>
          <CardTitle>Logs</CardTitle>
          <CardDescription>Last {{ logs.length }} lines.</CardDescription>
        </CardHeader>
        <CardContent>
          <pre class="max-h-80 overflow-auto rounded bg-muted p-3 font-mono text-xs leading-relaxed"><code>{{ logs.join('\n') }}</code></pre>
        </CardContent>
      </Card>
    </div>

    <!-- Terminal -->
    <div v-else-if="activeTab === 'terminal'">
      <Card>
        <CardContent class="p-2">
          <div ref="termEl" class="h-80 rounded bg-[#0e1117]" />
        </CardContent>
      </Card>
    </div>

    <!-- Links -->
    <div v-else-if="activeTab === 'links'">
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
    </div>
  </div>
</template>
