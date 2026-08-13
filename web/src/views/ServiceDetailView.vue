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
  Plus,
  Pencil,
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
  { id: 'webhooks', label: 'Webhooks', icon: 'webhook' },
  { id: 'healthcheck', label: 'Healthcheck', icon: 'heart' },
  { id: 'limits', label: 'Resource Limits', icon: 'gauge' },
  { id: 'metrics', label: 'Metrics', icon: 'chart' },
  { id: 'danger', label: 'Danger Zone', icon: 'alert' },
]
const activeSection = ref('general')
const activeTab = ref('configuration')
const topTabs = [
  { id: 'configuration', label: 'Configuration', icon: Box },
  { id: 'deployments', label: 'Deployments', icon: Activity },
  { id: 'logs', label: 'Logs', icon: ScrollText },
  { id: 'terminal', label: 'Terminal', icon: Box },
]

// ─── General form state ───────────────────────────────────────────────────
const form = reactive({
  name: '',
  description: '',
  image: '',
  imageTag: 'latest',
  dockerOptions: '',
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
const editingDomainId = ref<string | null>(null)
function buildDomainHost(): string {
  const sub = newDomain.subdomain.trim().replace(/^\.+/, '')
  const root = selectedRootDomain.value.trim().replace(/^\.+/, '')
  if (!root) return sub
  return sub ? `${sub}.${root}` : root
}
async function addOrUpdateDomain() {
  const host = buildDomainHost()
  if (!host) {
    domainError.value = 'Fill the subdomain or pick a root domain first.'
    return
  }
  domainError.value = ''
  const port = newDomain.port.trim() || '80'
  try {
    if (editingDomainId.value) {
      await store.updateServiceDomain(projectId.value, envId.value, serviceId.value, editingDomainId.value, { host, port })
    } else {
      await store.addServiceDomain(projectId.value, envId.value, serviceId.value, host, port)
    }
    resetDomainForm()
  } catch (e: any) {
    domainError.value = e?.message || 'Failed to save domain'
  }
}
function editDomain(d: { id: string; host: string; port: string }) {
  editingDomainId.value = d.id
  // Split host into subdomain + root: first label is subdomain, rest is root.
  const parts = d.host.split('.')
  if (parts.length >= 2) {
    newDomain.subdomain = parts[0]
    selectedRootDomain.value = parts.slice(1).join('.')
  } else {
    newDomain.subdomain = ''
    selectedRootDomain.value = d.host
  }
  newDomain.port = d.port
}
function resetDomainForm() {
  editingDomainId.value = null
  newDomain.subdomain = ''
  newDomain.port = '80'
}
async function removeDomain(did: string) {
  try {
    await store.removeServiceDomain(projectId.value, envId.value, serviceId.value, did)
    if (editingDomainId.value === did) resetDomainForm()
  } catch (e: any) {
    domainError.value = e?.message || 'Failed to remove domain'
  }
}

// ─── Service networks (port mappings, service_networks table) ─────────────
const newMapping = reactive({ from: '', to: '' })
const mappingError = ref('')
const editingNetworkId = ref<string | null>(null)
async function addOrUpdateMapping() {
  const from = newMapping.from.trim()
  const to = newMapping.to.trim()
  if (!from) {
    mappingError.value = 'Fill the host port.'
    return
  }
  mappingError.value = ''
  try {
    if (editingNetworkId.value) {
      await store.updateServiceNetwork(projectId.value, envId.value, serviceId.value, editingNetworkId.value, { host_port: from, container_port: to })
    } else {
      await store.addServiceNetwork(projectId.value, envId.value, serviceId.value, from, to)
    }
    resetMappingForm()
  } catch (e: any) {
    mappingError.value = e?.message || 'Failed to save port mapping'
  }
}
function editMapping(n: { id: string; hostPort: string; containerPort: string }) {
  editingNetworkId.value = n.id
  newMapping.from = n.hostPort
  newMapping.to = n.containerPort
}
function resetMappingForm() {
  editingNetworkId.value = null
  newMapping.from = ''
  newMapping.to = ''
}
async function removeMapping(nid: string) {
  try {
    await store.removeServiceNetwork(projectId.value, envId.value, serviceId.value, nid)
    if (editingNetworkId.value === nid) resetMappingForm()
  } catch (e: any) {
    mappingError.value = e?.message || 'Failed to remove port mapping'
  }
}

// Prevent number input value change on scroll (wheel) — CSS hides the spinners.
function preventNumberScroll(e: WheelEvent) {
  e.preventDefault()
}

// Sanitize a replica count: digits only, no leading zero, min 1.
// Rejects "-", "e", "0" at the start, decimals, etc.
function sanitizeReplicas(v: unknown): number {
  const s = String(v ?? '').replace(/[^\d]/g, '') // strip anything non-digit (kills -, e, .)
  if (!s) return 1
  return Math.max(1, parseInt(s.replace(/^0+/, '') || '1', 10))
}

// Hard guard: block invalid keys (-, e, E, ., ,, +) before they ever
// reach the input. Works on desktop + mobile hardware keyboards.
function blockReplicaKeys(e: KeyboardEvent) {
  if (e.key.length === 1 && !/[0-9]/.test(e.key)) {
    e.preventDefault()
  }
}

// Sync the DOM value immediately after sanitizing so the displayed text
// can never show a rejected character (fixes the "-1" visible bug).
function onReplicasInput(e: Event, target: 'replicas' | 'replicasMin' | 'replicasMax') {
  const el = e.target as HTMLInputElement
  const clean = sanitizeReplicas(el.value)
  form[target] = clean
  // Force the DOM to reflect the sanitized value right away — Vue's
  // :value binding won't re-render when the value didn't change.
  el.value = String(clean)
}

// Block paste of non-digit content (e.g. "-1", "3e4" from clipboard).
function blockReplicaPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text') ?? ''
  if (!/[0-9]/.test(text)) {
    e.preventDefault()
    return
  }
  // Allow the paste but strip non-digits — the @input handler cleans up.
  const clean = sanitizeReplicas(text)
  if (clean !== Number(text)) {
    e.preventDefault()
    const el = e.target as HTMLInputElement
    const start = el.selectionStart ?? el.value.length
    const end = el.selectionEnd ?? el.value.length
    el.value = el.value.slice(0, start) + String(clean) + el.value.slice(end)
    el.dispatchEvent(new Event('input', { bubbles: true }))
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

    <!-- Top tab bar: buttons (desktop) -->
    <div class="hidden flex-wrap gap-1 border-b text-sm sm:flex">
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

    <!-- Top tab bar: dropdown (mobile) -->
    <div class="sm:hidden">
      <select
        v-model="activeTab"
        class="h-9 w-full rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <option v-for="t in topTabs" :key="t.id" :value="t.id">{{ t.label }}</option>
      </select>
    </div>

    <!-- Configuration: sidebar LEFT + form content -->
    <div v-if="activeTab === 'configuration'" class="grid gap-4 md:grid-cols-[220px_1fr]">
      <!-- Left sidebar: buttons (desktop) -->
      <nav class="hidden flex-col gap-0.5 rounded-md border bg-card p-2 text-sm md:flex md:h-fit">
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

      <!-- Left sidebar: dropdown (mobile) -->
      <div class="md:hidden">
        <select
          v-model="activeSection"
          class="h-9 w-full rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <option v-for="sec in sections" :key="sec.id" :value="sec.id">{{ sectionIcons[sec.icon] }} {{ sec.label }}</option>
        </select>
      </div>

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
                  <div class="flex gap-2">
                    <Input v-model="newDomain.subdomain" placeholder="subdomain" class="flex-1" />
                    <select
                      v-model="selectedRootDomain"
                      class="h-9 w-40 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <option value="">— domain —</option>
                      <option v-for="d in availableRootDomains()" :key="d" :value="d">{{ d }}</option>
                    </select>
                    <Input v-model="newDomain.port" placeholder="80" class="w-20" />
                  </div>
                  <div class="flex items-center justify-between gap-2">
                    <p class="text-xs text-muted-foreground">
                      Preview: <span class="font-mono">{{ buildDomainHost() || '—' }}</span>
                    </p>
                    <Button
                      size="sm"
                      variant="outline"
                      @click="addOrUpdateDomain"
                      class="shrink-0"
                    >
                      <Plus v-if="!editingDomainId" class="mr-1 size-4" />
                      <Save v-else class="mr-1 size-4" />
                      {{ editingDomainId ? 'Update' : 'Add' }}
                    </Button>
                  </div>
                </div>
                <p v-if="domainError" class="text-xs text-destructive">{{ domainError }}</p>
              </CardHeader>
              <CardContent class="grid gap-1.5">
                <div v-for="d in service.domains ?? []" :key="d.id" class="flex items-center justify-between gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
                  <span class="truncate font-mono">{{ d.host }}</span>
                  <span class="flex items-center gap-2">
                    <Badge variant="secondary">→ :{{ d.port }}</Badge>
                    <button class="text-muted-foreground hover:text-foreground" @click="editDomain(d)"><Pencil class="size-3.5" /></button>
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
                <div class="flex gap-2">
                  <Input v-model="newMapping.from" placeholder="host port (3000)" class="flex-1" />
                  <Input v-model="newMapping.to" placeholder="container port (3000)" class="flex-1" />
                  <Button
                    size="sm"
                    variant="outline"
                    @click="addOrUpdateMapping"
                    class="shrink-0"
                  >
                    <Plus v-if="!editingNetworkId" class="mr-1 size-4" />
                    <Save v-else class="mr-1 size-4" />
                    {{ editingNetworkId ? 'Update' : 'Add' }}
                  </Button>
                </div>
                <p v-if="mappingError" class="text-xs text-destructive">{{ mappingError }}</p>
                <div v-for="n in service.networks ?? []" :key="n.id" class="flex items-center justify-between gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
                  <span class="truncate font-mono">{{ n.containerPort ? `${n.hostPort}:${n.containerPort}` : n.hostPort }}</span>
                  <span class="flex items-center gap-2">
                    <button class="text-muted-foreground hover:text-foreground" @click="editMapping(n)"><Pencil class="size-3.5" /></button>
                    <button class="text-destructive hover:text-destructive/80" @click="removeMapping(n.id)"><Trash2 class="size-3.5" /></button>
                  </span>
                </div>
                <p v-if="!(service.networks ?? []).length" class="text-xs text-muted-foreground">No port mappings yet.</p>
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
                    <input
                      :value="String(form.replicas)"
                      type="text"
                      inputmode="numeric"
                      pattern="[0-9]*"
                      autocomplete="off"
                      class="number-input-no-spin h-9 w-full rounded-md border bg-transparent px-2.5 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 dark:bg-input/30 md:text-sm"
                      @wheel.prevent="preventNumberScroll"
                      @keydown="blockReplicaKeys"
                      @paste="blockReplicaPaste"
                      @input="onReplicasInput($event, 'replicas')"
                    />
                  </div>
                </template>
                <template v-else>
                  <div class="grid grid-cols-2 gap-3">
                    <div class="grid gap-1.5">
                      <Label>Min *</Label>
                      <input
                        :value="String(form.replicasMin)"
                        type="text"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        autocomplete="off"
                        class="number-input-no-spin h-9 w-full rounded-md border bg-transparent px-2.5 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 dark:bg-input/30 md:text-sm"
                        @wheel.prevent="preventNumberScroll"
                        @keydown="blockReplicaKeys"
                        @paste="blockReplicaPaste"
                        @input="onReplicasInput($event, 'replicasMin')"
                      />
                    </div>
                    <div class="grid gap-1.5">
                      <Label>Max *</Label>
                      <input
                        :value="String(form.replicasMax)"
                        type="text"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        autocomplete="off"
                        class="number-input-no-spin h-9 w-full rounded-md border bg-transparent px-2.5 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 dark:bg-input/30 md:text-sm"
                        @wheel.prevent="preventNumberScroll"
                        @keydown="blockReplicaKeys"
                        @paste="blockReplicaPaste"
                        @input="onReplicasInput($event, 'replicasMax')"
                      />
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
  </div>
</template>

<style scoped>
/* Hide number input spinners (Chrome/Safari/Edge + Firefox) */
.number-input-no-spin::-webkit-outer-spin-button,
.number-input-no-spin::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.number-input-no-spin {
  -moz-appearance: textfield;
  appearance: textfield;
}
</style>
