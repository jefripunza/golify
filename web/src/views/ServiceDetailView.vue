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
import type { Deployment, Service } from '@/lib/types'
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
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  X,
  Search,
  Copy,
  Download,
  RefreshCw,
  Maximize2,
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
  replicas: '1',
  replicasMin: '1',
  replicasMax: '1',
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
  form.replicas = String(s.replicas ?? 1)
  form.replicasMin = String(s.replicasMin ?? 1)
  form.replicasMax = String(s.replicasMax ?? 1)
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
  // Unique check: same host (subdomain+domain, incl. bare root) not allowed twice.
  const svc = service.value
  const exists = (svc?.domains ?? []).some(
    (d) => d.host === host && d.id !== editingDomainId.value,
  )
  if (exists) {
    domainError.value = `Domain already exists: ${host}`
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

// Port inputs: digits only (same guard style as replicas).
function sanitizePort(v: unknown): string {
  return String(v ?? '').replace(/[^\d]/g, '')
}
function onPortKeyup(e: Event, target: 'from' | 'to') {
  const el = e.target as HTMLInputElement
  const clean = sanitizePort(el.value)
  newMapping[target] = clean
  if (el.value !== clean) el.value = clean
}

// Prevent number input value change on scroll (wheel) — CSS hides the spinners.
function preventNumberScroll(e: WheelEvent) {
  e.preventDefault()
}

// Sanitize a replica count string: digits only — minus and letters are
// stripped out entirely (replaced/hilang). A lone "0" typed from empty is
// rejected (min 1), leading zeros stripped ('05' -> '5'), empty stays empty
// so the user can clear and retype. Clamp to 1 happens at save time.
function sanitizeReplicas(v: unknown): string {
  const s = String(v ?? '').replace(/[^\d]/g, '') // strip anything non-digit (kills -, e, letters)
  return s.replace(/^0+/, '') // '0' -> '', '05' -> '5', '10' -> '10'
}

// Hard guard: block invalid keys (-, e, E, ., ,, +) before they ever
// reach the input. Works on desktop + mobile hardware keyboards.
function blockReplicaKeys(e: KeyboardEvent) {
  if (e.key.length === 1 && !/[0-9]/.test(e.key)) {
    e.preventDefault()
  }
}

// Runs on keyup: sanitize after every keystroke so minus/letters never
// survive, and a lone "0" from empty is rejected (stays empty).
function onReplicasKeyup(e: Event, target: 'replicas' | 'replicasMin' | 'replicasMax') {
  const el = e.target as HTMLInputElement
  const clean = sanitizeReplicas(el.value)
  form[target] = clean as never
  // Force the DOM to reflect the sanitized value right away — Vue's
  // :value binding won't re-render when the value didn't change.
  if (el.value !== clean) el.value = clean
}

// Runs on blur: if the field was left empty (or sanitized to empty),
// default it to 1 so the saved value is never empty/0.
function onReplicasBlur(e: Event, target: 'replicas' | 'replicasMin' | 'replicasMax') {
  const el = e.target as HTMLInputElement
  const clean = sanitizeReplicas(el.value)
  const v = clean === '' ? '1' : clean
  form[target] = v as never
  el.value = v
}

// Block paste of non-digit content (e.g. "-1", "3e4" from clipboard).
function blockReplicaPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text') ?? ''
  if (!/[0-9]/.test(text)) {
    e.preventDefault()
    return
  }
  // Allow the paste but strip non-digits — the @keyup handler cleans up.
  const clean = sanitizeReplicas(text)
  if (clean !== text) {
    e.preventDefault()
    const el = e.target as HTMLInputElement
    const start = el.selectionStart ?? el.value.length
    const end = el.selectionEnd ?? el.value.length
    el.value = el.value.slice(0, start) + clean + el.value.slice(end)
    el.dispatchEvent(new KeyboardEvent('keyup', { bubbles: true }))
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
  const url = `${proto}//${location.host}/api/ws/terminal/${encodeURIComponent(serviceId.value)}?token=${encodeURIComponent(auth?.token ?? '')}`
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
  if (tab === 'logs') {
    setTimeout(() => connectLogs(), 0)
  }
})
watch(termEl, () => setTimeout(() => fit?.fit(), 0))
// When the service flips to running and we're on the Logs tab, connect.
watch(() => service.value?.status, (s) => {
  if (s === 'running' && activeTab.value === 'logs') {
    setTimeout(() => connectLogs(), 0)
  }
})
onBeforeUnmount(() => {
  closeDeployLog()
  logWS?.close()
  logWS = null
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

// ─── Danger Zone: delete service (cascade) ────────────────────────────────
const deletingService = ref(false)
const deleteServiceError = ref('')
async function deleteServiceNow() {
  if (deletingService.value) return
  if (!confirm(`Delete service "${service.value?.name}" permanently? This removes domains, networks, deployment history and the container. This cannot be undone!`)) return
  deletingService.value = true
  deleteServiceError.value = ''
  try {
    await store.removeService(projectId.value, envId.value, serviceId.value)
    router.push(`/project/${projectId.value}/environment/${envId.value}/services`)
  } catch (e: any) {
    deleteServiceError.value = e?.message ?? 'Failed to delete service'
  } finally {
    deletingService.value = false
  }
}

// ─── Deployments ──────────────────────────────────────────────────────────

// Dummy deployment history matching the reference screenshot
// (3 entries: 2 success + 1 failed, 2026-08-08, HEAD / Manual / API).
const dummyDeployments: Deployment[] = [
  {
    id: 'dep-dummy-1',
    serviceId: serviceId.value,
    status: 'success',
    commit: 'HEAD',
    source: 'Manual',
    startedAt: '2026-08-08T04:30:33Z',
    endedAt: '2026-08-08T04:31:18Z',
    createdAt: '2026-08-08T04:30:33Z',
  },
  {
    id: 'dep-dummy-2',
    serviceId: serviceId.value,
    status: 'success',
    commit: 'HEAD',
    source: 'API',
    startedAt: '2026-08-08T03:06:49Z',
    endedAt: '2026-08-08T03:09:14Z',
    createdAt: '2026-08-08T03:06:49Z',
  },
  {
    id: 'dep-dummy-3',
    serviceId: serviceId.value,
    status: 'failed',
    commit: 'HEAD',
    source: 'API',
    startedAt: '2026-08-08T03:01:02Z',
    endedAt: '2026-08-08T03:03:07Z',
    createdAt: '2026-08-08T03:01:02Z',
  },
]

const deployments = ref<Deployment[]>([])
const deploymentsLoading = ref(false)
const deploymentsError = ref('')
const activeDeployId = ref<string | null>(null) // row with live log open
const deployLogLines = ref<string[]>([])
let deployWS: WebSocket | null = null

async function loadDeployments() {
  deploymentsLoading.value = true
  deploymentsError.value = ''
  try {
    const rows = await store.fetchDeployments(projectId.value, envId.value, serviceId.value)
    deployments.value = rows.length ? rows : [...dummyDeployments]
  } catch (e: any) {
    deploymentsError.value = e?.message || 'Failed to load deployments'
    deployments.value = [...dummyDeployments]
  } finally {
    deploymentsLoading.value = false
  }
}

// Open a deployment row → Coolify-style deployment log modal.
// - running  → WS live stream (auto-closes when the deploy finishes)
// - finished → fetch persisted log from the new detail endpoint
const deployModal = ref(false)
const deployModalDep = ref<Deployment | null>(null)
const deployModalLoading = ref(false)
const deploySearch = ref('')
const deployFullscreen = ref(false)

function openDeployLog(dep: Deployment) {
  if (activeDeployId.value === dep.id) {
    closeDeployLog()
    return
  }
  closeDeployLog()
  activeDeployId.value = dep.id
  deployLogLines.value = []
  deployModal.value = true
  deployModalDep.value = dep

  if (dep.status === 'running') {
    // live stream over WS; close when finished
    const auth = getAuth()
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/api/ws/deploy/${encodeURIComponent(dep.id)}?token=${encodeURIComponent(auth?.token ?? '')}`
    try { deployWS = new WebSocket(url) } catch { deployWS = null }
    if (!deployWS) {
      deployLogLines.value = ['[WS not available]']
      return
    }
    deployWS.onmessage = (ev) => {
      deployLogLines.value = [...deployLogLines.value, String(ev.data)]
    }
    deployWS.onclose = () => {
      // Deploy finished — refresh deployments list AND the store's service
      // status (stopped → running) so Logs/Terminal gating unlocks.
      setTimeout(async () => {
        await store.refresh()
        await loadDeployments()
      }, 300)
    }
    deployWS.onerror = () => {
      deployLogLines.value = [...deployLogLines.value, '[connection error]']
    }
  } else {
    // finished — fetch the persisted log from the DB-backed endpoint
    deployModalLoading.value = true
    store.fetchDeployment(projectId.value, envId.value, serviceId.value, dep.id)
      .then((full) => {
        deployModalDep.value = full
        deployLogLines.value = full.log ? full.log.split('\n') : []
      })
      .catch(() => { deployLogLines.value = ['[failed to load log]'] })
      .finally(() => { deployModalLoading.value = false })
  }
}

function closeDeployLog() {
  deployWS?.close()
  deployWS = null
  activeDeployId.value = null
  deployModal.value = false
  deployModalDep.value = null
  deploySearch.value = ''
  deployFullscreen.value = false
}

// ─── deployment log modal helpers ─────────────────────────────────────────
const deployLogBodyEl = ref<HTMLElement | null>(null)

const filteredDeployLog = computed(() => {
  const q = deploySearch.value.trim().toLowerCase()
  if (!q) return deployLogLines.value
  return deployLogLines.value.filter((l) => l.toLowerCase().includes(q))
})

function copyDeployLog() {
  const text = deployLogLines.value.join('\n')
  navigator.clipboard?.writeText(text).catch(() => {})
}

function downloadDeployLog() {
  const text = deployLogLines.value.join('\n')
  const blob = new Blob([text], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `deploy-${deployModalDep.value?.id ?? 'log'}.log`
  a.click()
  URL.revokeObjectURL(url)
}

function refreshDeployLog() {
  const dep = deployModalDep.value
  if (!dep) return
  if (dep.status === 'running') return // WS already live
  deployModalLoading.value = true
  store.fetchDeployment(projectId.value, envId.value, serviceId.value, dep.id)
    .then((full) => {
      deployModalDep.value = full
      deployLogLines.value = full.log ? full.log.split('\n') : []
    })
    .catch(() => { deployLogLines.value = [...deployLogLines.value, '[failed to reload log]'] })
    .finally(() => { deployModalLoading.value = false })
}

// auto-scroll the log body to the bottom as new lines arrive
watch(deployLogLines, () => {
  const el = deployLogBodyEl.value
  if (el) el.scrollTop = el.scrollHeight
}, { flush: 'post' })

function fmtDateTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${p(d.getUTCMonth() + 1)}-${p(d.getUTCDate())} ${p(d.getUTCHours())}:${p(d.getUTCMinutes())}:${p(d.getUTCSeconds())}`
}

function fmtDuration(startIso?: string, endIso?: string | null): string {
  if (!startIso) return '—'
  const s = new Date(startIso).getTime()
  const e = endIso ? new Date(endIso).getTime() : Date.now()
  if (Number.isNaN(s) || Number.isNaN(e)) return '—'
  const secs = Math.max(0, Math.round((e - s) / 1000))
  const m = Math.floor(secs / 60)
  const sec = secs % 60
  return `${String(m).padStart(2, '0')}m ${String(sec).padStart(2, '0')}s`
}

function fmtAgo(iso?: string): string {
  if (!iso) return '—'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return '—'
  const days = Math.floor((Date.now() - t) / 86_400_000)
  if (days <= 0) return 'today'
  return `${days} day${days === 1 ? '' : 's'} ago`
}

function statusClass(s: string): string {
  return s === 'success' ? 'border-l-green-500 text-green-500' : s === 'failed' ? 'border-l-red-500 text-red-500' : 'border-l-yellow-500 text-yellow-500'
}

// Start a deploy (from the Start button): POST → auto-switch to Deployments
// tab → open the new deployment's live log.
async function deployNow() {
  if (!service.value) return
  try {
    const dep = await store.createDeployment(projectId.value, envId.value, serviceId.value, { commit: 'HEAD', source: 'Manual' })
    activeTab.value = 'deployments'
    await loadDeployments()
    // find the new row (by id) and open its live log
    openDeployLog(dep)
  } catch (e: any) {
    deploymentsError.value = e?.message || 'Failed to start deployment'
  }
}

onMounted(() => {
  store.fetchRootDomains()
  loadDeployments()
})

// refresh deployments whenever the tab becomes active (new deploys appear)
watch(activeTab, (tab) => {
  if (tab === 'deployments') loadDeployments()
})

// ─── Logs (live, gated by service status) ─────────────────────────────────
let logWS: WebSocket | null = null
function connectLogs() {
  logWS?.close()
  logWS = null
  logs.value = []
  if (service.value?.status !== 'running') return
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/api/ws/log/${encodeURIComponent(serviceId.value)}?token=${encodeURIComponent(auth?.token ?? '')}`
  try { logWS = new WebSocket(url) } catch { logWS = null }
  if (!logWS) {
    logs.value = ['[WS not available]']
    return
  }
  logWS.onmessage = (ev) => {
    logs.value = [...logs.value, String(ev.data)]
    if (logs.value.length > 200) logs.value = logs.value.slice(-200)
  }
  logWS.onclose = () => { /* keep */ }
  logWS.onerror = () => logs.value = [...logs.value, '[connection error]']
}

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
        <Button size="sm" :disabled="service.status === 'running'" @click="deployNow">
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
                  <input
                    :value="newMapping.from"
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    autocomplete="off"
                    placeholder="host port (3000)"
                    class="number-input-no-spin h-9 w-full flex-1 rounded-md border bg-transparent px-2.5 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 dark:bg-input/30 md:text-sm"
                    @wheel.prevent="preventNumberScroll"
                    @keydown="blockReplicaKeys"
                    @keyup="onPortKeyup($event, 'from')"
                  />
                  <input
                    :value="newMapping.to"
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    autocomplete="off"
                    placeholder="container port (3000)"
                    class="number-input-no-spin h-9 w-full flex-1 rounded-md border bg-transparent px-2.5 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-3 dark:bg-input/30 md:text-sm"
                    @wheel.prevent="preventNumberScroll"
                    @keydown="blockReplicaKeys"
                    @keyup="onPortKeyup($event, 'to')"
                  />
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
                      @keyup="onReplicasKeyup($event, 'replicas')"
                      @paste="blockReplicaPaste"
                      @blur="onReplicasBlur($event, 'replicas')"
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
                        @keyup="onReplicasKeyup($event, 'replicasMin')"
                        @paste="blockReplicaPaste"
                        @blur="onReplicasBlur($event, 'replicasMin')"
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
                        @keyup="onReplicasKeyup($event, 'replicasMax')"
                        @paste="blockReplicaPaste"
                        @blur="onReplicasBlur($event, 'replicasMax')"
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
                <CardContent class="space-y-4">
                  <div class="rounded-lg border border-destructive/30 bg-destructive/5 p-4">
                    <p class="text-sm font-medium">Delete Resource</p>
                    <p class="mt-1 text-xs text-muted-foreground">
                      This will stop the container, remove the service and all related data — domains, networks, deployment history. There is no coming back!
                    </p>
                    <Button variant="destructive" class="mt-3" :disabled="deletingService" @click="deleteServiceNow">
                      <Loader2 v-if="deletingService" class="mr-1 size-4 animate-spin" />
                      <Trash2 v-else class="mr-1 size-4" />
                      {{ deletingService ? 'Deleting…' : 'Delete' }}
                    </Button>
                    <p v-if="deleteServiceError" class="mt-2 text-xs text-destructive">{{ deleteServiceError }}</p>
                  </div>
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
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-lg font-semibold">Deployments</h2>
          <p class="text-xs text-muted-foreground">
            {{ deployments.length }} total {{ deployments.length === 1 ? 'deployment' : 'deployments' }}
          </p>
        </div>
        <div class="flex items-center gap-1 text-sm text-muted-foreground">
          <Button size="icon" variant="ghost" disabled class="size-7"><ChevronLeft class="size-4" /></Button>
          <span class="text-xs">Page 1 of 1</span>
          <Button size="icon" variant="ghost" disabled class="size-7"><ChevronRight class="size-4" /></Button>
        </div>
      </div>

      <!-- Filter row -->
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <Label class="text-sm">Pull Request id</Label>
        <Input placeholder="e.g. 123" class="h-8 w-40" />
        <Button size="sm" variant="outline" class="h-8">Filter</Button>
      </div>

      <p v-if="deploymentsError" class="mb-2 text-sm text-destructive">{{ deploymentsError }}</p>

      <div v-if="deploymentsLoading" class="text-sm text-muted-foreground">Loading…</div>

      <div v-else class="grid gap-2">
        <div
          v-for="dep in deployments"
          :key="dep.id"
          class="rounded-md border border-l-4 bg-card p-3 transition-colors hover:bg-accent/50"
          :class="statusClass(dep.status)"
        >
          <div class="flex cursor-pointer items-center justify-between gap-3" @click="openDeployLog(dep)">
            <div class="flex items-center gap-3">
              <span class="inline-block size-2 rounded-full" :class="dep.status === 'success' ? 'bg-green-500' : dep.status === 'failed' ? 'bg-red-500' : 'bg-yellow-500 animate-pulse'" />
              <span class="text-sm font-medium" :class="dep.status === 'success' ? 'text-green-500' : dep.status === 'failed' ? 'text-red-500' : 'text-yellow-500'">
                {{ dep.status === 'running' ? 'Running' : dep.status === 'success' ? 'Success' : 'Failed' }}
              </span>
            </div>
            <ChevronDown v-if="activeDeployId === dep.id" class="size-4 text-muted-foreground" />
            <ChevronRight v-else class="size-4 text-muted-foreground" />
          </div>
          <div class="mt-2 grid gap-1 text-xs text-muted-foreground">
            <p><span class="font-medium text-foreground">Started:</span> {{ fmtDateTime(dep.startedAt) }} UTC</p>
            <p><span class="font-medium text-foreground">Ended:</span> {{ dep.endedAt ? fmtDateTime(dep.endedAt) + ' UTC' : '—' }}</p>
            <p><span class="font-medium text-foreground">Duration:</span> {{ fmtDuration(dep.startedAt, dep.endedAt) }}</p>
            <p class="flex items-center gap-1"><span class="font-medium text-foreground">Finished</span> {{ fmtAgo(dep.endedAt ?? dep.startedAt) }}</p>
            <p class="flex items-center gap-1">
              <span class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">{{ dep.commit }}</span>
              <span class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">{{ dep.source }}</span>
            </p>
          </div>

          <!-- Live log row (expanded) -->
          <div v-if="activeDeployId === dep.id" class="mt-2 rounded bg-[#0e1117] p-2">
            <pre class="max-h-64 overflow-auto font-mono text-[11px] leading-relaxed text-green-400"><code>{{ deployLogLines.join('\n') || 'Waiting for log output…' }}</code></pre>
          </div>
        </div>
        <p v-if="!deployments.length" class="text-sm text-muted-foreground">No deployments yet.</p>
      </div>
    </div>

    <!-- Deployment log modal (Coolify-style) -->
    <div
      v-if="deployModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-2 md:p-6"
      @click.self="closeDeployLog"
    >
      <div
        class="flex w-full flex-col overflow-hidden rounded-lg border bg-card shadow-2xl"
        :class="deployFullscreen ? 'h-[96vh] max-w-none' : 'h-[80vh] max-w-3xl'"
      >
        <!-- Header -->
        <div class="flex items-center justify-between border-b px-4 py-3">
          <div>
            <h3 class="text-sm font-semibold">Deployment Log</h3>
            <p class="text-xs text-muted-foreground">
              {{ deployModalDep?.id }}
            </p>
          </div>
          <Button size="sm" variant="ghost" class="size-8" @click="closeDeployLog">
            <X class="size-4" />
          </Button>
        </div>

        <!-- Sub-header: status + toolbar -->
        <div class="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2">
          <p class="text-xs">
            Deployment is
            <span
              class="font-medium"
              :class="deployModalDep?.status === 'success' ? 'text-green-500' : deployModalDep?.status === 'failed' ? 'text-red-500' : 'text-yellow-500'"
            >
              {{ deployModalDep?.status === 'running' ? 'Running' : deployModalDep?.status === 'success' ? 'Finished' : 'Failed' }}
            </span>
            <span v-if="deployModalDep?.status !== 'running'" class="ml-2 text-muted-foreground">· {{ fmtDuration(deployModalDep?.startedAt, deployModalDep?.endedAt) }}</span>
          </p>
          <div class="flex items-center gap-1">
            <div class="relative">
              <Search class="absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input v-model="deploySearch" placeholder="Find in logs" class="h-7 w-44 pl-7 text-xs" />
            </div>
            <Button size="icon" variant="ghost" class="size-7" title="Copy" @click="copyDeployLog">
              <Copy class="size-3.5" />
            </Button>
            <Button size="icon" variant="ghost" class="size-7" title="Download" @click="downloadDeployLog">
              <Download class="size-3.5" />
            </Button>
            <Button size="icon" variant="ghost" class="size-7" title="Refresh" @click="refreshDeployLog">
              <RefreshCw class="size-3.5" />
            </Button>
            <Button size="icon" variant="ghost" class="size-7" :title="deployFullscreen ? 'Exit fullscreen' : 'Fullscreen'" @click="deployFullscreen = !deployFullscreen">
              <Maximize2 class="size-3.5" />
            </Button>
          </div>
        </div>

        <!-- Log body -->
        <div class="flex-1 overflow-auto bg-[#0e1117] p-3 font-mono text-[11px] leading-relaxed text-gray-300" ref="deployLogBodyEl">
          <div v-if="deployModalLoading" class="text-gray-500">Loading log…</div>
          <template v-else>
            <p v-for="(line, i) in filteredDeployLog" :key="i" class="whitespace-pre-wrap">
              <span v-if="line.match(/^20\d\d-/)" class="text-gray-500">{{ line.slice(0, 26) }}</span>
              <span v-if="line.match(/^20\d\d-/)" class="text-gray-300">{{ line.slice(26) }}</span>
              <span v-else class="text-gray-300">{{ line }}</span>
            </p>
            <p v-if="!deployLogLines.length" class="text-gray-500">Waiting for log output…</p>
          </template>
        </div>
      </div>
    </div>

    <!-- Logs (gated: only when service is running) -->
    <div v-else-if="activeTab === 'logs'">
      <Card>
        <CardHeader>
          <CardTitle>Logs</CardTitle>
          <CardDescription v-if="service.status !== 'running'">
            Service is <span class="text-yellow-500">{{ service.status }}</span> — start the service to see live logs.
          </CardDescription>
          <CardDescription v-else>Streaming logs from the container…</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="service.status === 'running'">
            <div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
              <span class="inline-block size-2 rounded-full bg-green-500" /> Live
            </div>
            <pre class="max-h-80 overflow-auto rounded bg-muted p-3 font-mono text-xs leading-relaxed"><code>{{ logs.join('\n') || 'Connecting…' }}</code></pre>
          </div>
          <div v-else class="flex flex-col items-center gap-2 rounded bg-muted p-6 text-center">
            <ScrollText class="size-6 text-muted-foreground" />
            <p class="text-sm text-muted-foreground">No logs yet — the service hasn't been started.</p>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Terminal (gated: only when service is running) -->
    <div v-else-if="activeTab === 'terminal'">
      <Card>
        <CardHeader>
          <CardTitle>Terminal</CardTitle>
          <CardDescription v-if="service.status !== 'running'">
            Service is <span class="text-yellow-500">{{ service.status }}</span> — start the service to open a terminal.
          </CardDescription>
        </CardHeader>
        <CardContent class="p-2">
          <div v-if="service.status === 'running'">
            <div ref="termEl" class="h-80 rounded bg-[#0e1117]" />
          </div>
          <div v-else class="flex flex-col items-center gap-2 rounded bg-muted p-6 text-center">
            <Box class="size-6 text-muted-foreground" />
            <p class="text-sm text-muted-foreground">Terminal is unavailable while the service is stopped.</p>
          </div>
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
