<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useProjectsStore } from '@/stores'
import { getAuth, authed } from '@/lib/api'
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
  Copy,
  Download,
  RefreshCw,
  Maximize,
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

// Load Balancer dropdown is only meaningful with more than one replica.
const lbCount = computed(() => {
  const n = form.replicasMode === 'range' ? Number(form.replicasMin) || 1 : Number(form.replicas) || 1
  return n
})
const lbVisible = computed(() => lbCount.value > 1)

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
// Tab state is driven by the URL query (?tab=...) so a page refresh keeps
// the user on the same tab.
const activeTab = ref<string>(typeof route.query.tab === 'string' && ['configuration', 'deployments', 'logs', 'terminal'].includes(route.query.tab) ? route.query.tab : 'configuration')

// keep the query in sync when the tab changes (replace, no history spam)
watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
})

// also respond to back/forward navigation that changes ?tab=
watch(() => route.query.tab, (q) => {
  if (typeof q === 'string' && q !== activeTab.value && ['configuration', 'deployments', 'logs', 'terminal'].includes(q)) {
    activeTab.value = q
  }
})

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
  loadBalancer: 'round_robin' as 'round_robin' | 'least_conn' | 'k8s',
})
const saving = ref(false)
const saveError = ref('')
const saveOk = ref(false)
const scaleNote = ref('')

// ─── Environment Variables (.env editor) ────────────────────────────────
const envText = ref('')
const savingEnv = ref(false)
const envSaved = ref(false)
const envError = ref('')

// Init env editor from service env_var (raw .env block string)
function initEnvEditor() {
  const s = service.value
  if (!s) return
  envText.value = s.envVar ?? ''
  envError.value = ''
  envSaved.value = false
}
watch(() => service.value?.id, () => { if (service.value) initEnvEditor() }, { immediate: true })

const envVarCount = computed(() => {
  const lines = envText.value.split('\n').filter((l) => l.trim() && !l.trim().startsWith('#'))
  return lines.length
})

// Parsed KEY/VALUE pairs from the env_var string (skips comments: full-line
// and inline "# ..."), mirrors backend parseEnvVarBlock.
const parsedEnvVars = computed(() => {
  const rows: { key: string; value: string }[] = []
  const seen = new Set<string>()
  for (const line of (service.value?.envVar ?? '').split('\n')) {
    let t = line.trim()
    if (!t || t.startsWith('#')) continue
    const ci = t.indexOf(' #')
    if (ci >= 0) t = t.slice(0, ci).trim()
    const eq = t.indexOf('=')
    if (eq <= 0) continue
    const key = t.slice(0, eq).trim()
    let value = t.slice(eq + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key) || seen.has(key)) continue
    seen.add(key)
    rows.push({ key, value })
  }
  return rows
})

// Parse .env text → rows (validated) — mirrors backend parseEnvVarBlock
function parseEnvText(): { key: string; value: string }[] {
  const rows: { key: string; value: string }[] = []
  for (const line of envText.value.split('\n')) {
    let t = line.trim()
    if (!t || t.startsWith('#')) continue
    // strip inline comment
    const ci = t.indexOf(' #')
    if (ci >= 0) t = t.slice(0, ci).trim()
    const eq = t.indexOf('=')
    if (eq <= 0) continue
    const key = t.slice(0, eq).trim()
    let value = t.slice(eq + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
      envError.value = `Invalid key: ${key} (must be A-Za-z0-9_ starting with letter or _)`
      return []
    }
    rows.push({ key, value })
  }
  return rows
}

async function saveEnvText() {
  const s = service.value
  if (!s) return
  envError.value = ''
  if (parseEnvText().length === 0 && envText.value.trim() !== '') {
    if (envError.value) return
  }
  savingEnv.value = true
  try {
    const base = `/api/v1/projects/${projectId.value}/environments/${envId.value}/services/${serviceId.value}`
    // Save the whole .env block as a single string on the service.
    await authed().patch(base, { json: { env_var: envText.value } })
    await store.fetchProjects()
    const fresh = store.getService(projectId.value, envId.value, serviceId.value)
    if (fresh) initEnvEditor()
    envSaved.value = true
    setTimeout(() => (envSaved.value = false), 2000)
  } catch (e: any) {
    envError.value = e?.message || 'Failed to save env vars'
  } finally {
    savingEnv.value = false
  }
}

function addEnvVarRow() {
  envText.value = (envText.value.trimEnd() ? envText.value.trimEnd() + '\n' : '') + 'NEW_VAR='
}

// ─── Monaco .env editor ─────────────────────────────────────────────────
const envEditorEl = ref<HTMLElement | null>(null)
let envEditor: { getValue(): string; setValue(v: string): void; onDidChangeModelContent(cb: () => void): void; dispose(): void } | null = null
let envEditorInitialized = false
let monacoPromise: Promise<typeof import('monaco-editor')> | null = null

// .env language definition — highlight keys cyan, values orange/green,
// comments gray (like the 9Router .env screenshot the user shared).
// Dipanggil sekali setelah monaco di-import (lazy).
function setupMonacoDotenv(m: typeof import('monaco-editor')) {
  m.languages.register({ id: 'dotenv' })
  m.languages.setMonarchTokensProvider('dotenv', {
    tokenizer: {
      root: [
        [/^#.*$/, 'comment'],
        [/^\s*[A-Za-z_][A-Za-z0-9_]*/, 'variable'],
        [/=/, 'operator'],
        [/".*"|'.*'/, 'string.quote'],
        [/\d+/, 'number'],
        [/https?:\/\/\S+/, 'url'],
        [/true|false/, 'boolean'],
        [/.*/, 'string'],
      ],
    },
  })
  m.editor.defineTheme('golify-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: '6b7280', fontStyle: 'italic' },
      { token: 'variable', foreground: '22d3ee' },   // cyan
      { token: 'operator', foreground: 'a1a1aa' },
      { token: 'string', foreground: 'fbbf24' },     // orange
      { token: 'string.quote', foreground: '4ade80' }, // green
      { token: 'number', foreground: 'f472b6' },
      { token: 'url', foreground: '4ade80' },        // green URLs
      { token: 'boolean', foreground: 'fbbf24' },    // orange booleans
    ],
    colors: {
      'editor.background': '#0a0a0a',
      'editor.foreground': '#e4e4e7',
      'editorLineNumber.foreground': '#52525b',
      'editorLineNumber.activeForeground': '#a1a1aa',
      'editorCursor.foreground': '#22d3ee',
      'editor.selectionBackground': '#155e7533',
      'editor.lineHighlightBackground': '#18181b',
      'editorIndentGuide.background1': '#27272a',
    },
  })
}

function initEnvEditorMonaco() {
  if (envEditorInitialized) return
  if (!envEditorEl.value) return // elemen belum render — coba lagi nanti
  envEditorInitialized = true
  // Lazy-load monaco-editor (chunk terpisah ~4MB, hanya dimuat saat
  // section Environment Variables dibuka)
  if (!monacoPromise) monacoPromise = import('monaco-editor')
  monacoPromise.then((m) => {
    if (!envEditorEl.value) return
    setupMonacoDotenv(m)
    envEditor = m.editor.create(envEditorEl.value, {
    value: envText.value,
    language: 'dotenv',
    theme: 'golify-dark',
    fontSize: 13,
    fontFamily: 'JetBrains Mono, Fira Code, Menlo, monospace',
    lineNumbers: 'on',
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    automaticLayout: true,
    padding: { top: 12, bottom: 12 },
    renderWhitespace: 'none',
    wordWrap: 'off',
    tabSize: 2,
    scrollbar: { verticalScrollbarSize: 10, horizontalScrollbarSize: 10 },
  })
    envEditor.onDidChangeModelContent(() => {
      envText.value = envEditor?.getValue() ?? ''
    })
  })
}

// Lazily init when the user opens the Environment Variables section
watch(activeSection, (s) => {
  if (s === 'envvars') {
    // retry init until the editor element is actually mounted (v-if render
    // may lag one tick behind the watcher)
    let tries = 0
    const attempt = () => {
      initEnvEditor() // re-read fresh env_var from store
      initEnvEditorMonaco()
      // editor dibuat async (monaco lazy-loaded) — retry sampai terisi
      if (envEditor) {
        envEditor.setValue(envText.value)
        return
      }
      if (++tries < 15) setTimeout(attempt, 100)
    }
    nextTick(attempt)
  }
})

// Keep the Monaco editor in sync whenever the service env_var changes
// (e.g. after fetchProjects resolves and service.value is populated later
// than the section was opened — the common race that leaves the editor empty).
watch(
  () => service.value?.envVar,
  () => {
    if (envEditor && activeSection.value === 'envvars') {
      envEditor.setValue(service.value?.envVar ?? '')
    }
  },
)

onBeforeUnmount(() => {
  envEditor?.dispose()
  envEditor = null
})

// ─── Persistent Storage ─────────────────────────────────────────────────
const storageForm = reactive({ name: '', mountPath: '', hostPath: '' })
const savingStorage = ref(false)
const storageError = ref('')

async function createStorage() {
  if (!storageForm.name.trim() || !storageForm.mountPath.trim()) {
    storageError.value = 'Name and Mount Path are required'
    return
  }
  savingStorage.value = true
  storageError.value = ''
  try {
    await authed().post(`/api/v1/projects/${projectId.value}/environments/${envId.value}/services/${serviceId.value}/storages`, {
      json: {
        name: storageForm.name.trim(),
        mount_path: storageForm.mountPath.trim(),
        host_path: storageForm.hostPath.trim(),
      },
    })
    storageForm.name = ''
    storageForm.mountPath = ''
    storageForm.hostPath = ''
    await store.fetchProjects()
  } catch (e: any) {
    storageError.value = e?.message || 'Failed to create storage'
  } finally {
    savingStorage.value = false
  }
}

async function removeStorage(id: string) {
  try {
    await authed().delete(`/api/v1/projects/${projectId.value}/environments/${envId.value}/services/${serviceId.value}/storages/${id}`)
    await store.fetchProjects()
  } catch (e: any) {
    storageError.value = e?.message || 'Failed to delete storage'
  }
}

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
  form.loadBalancer = (s.loadBalancer === 'least_conn' || s.loadBalancer === 'k8s' ? s.loadBalancer : 'round_robin')
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
    // Remember the previous replica count BEFORE the save — if it changes we
    // scale the running containers right after persisting (no-op when same).
    const oldReplicas = service.value?.replicas ?? 1
    const newReplicas = form.replicasMode === 'fix'
      ? Number(form.replicas) || 1
      : Number(form.replicasMin) || 1
    const replicasChanged = oldReplicas !== newReplicas

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
      replicas: newReplicas,
      replicas_min: Number(form.replicasMin) || 1,
      replicas_max: Number(form.replicasMax) || 1,
      load_balancer: form.loadBalancer,
    })
    saveOk.value = true
    setTimeout(() => (saveOk.value = false), 2500)

    // Replica count changed → execute the scale so the running containers
    // match the new value. Same value → nothing to do.
    if (replicasChanged) {
      try {
        const res = await store.scaleService(projectId.value, envId.value, serviceId.value, newReplicas)
        const note = res?.scaled
          ? `Replicas scaled to ${newReplicas}`
          : (res?.message || `Replicas saved to ${newReplicas} (applied on start)`)
        saveError.value = ''
        saveOk.value = true
        scaleNote.value = note
        setTimeout(() => (saveOk.value = false), 3000)
      } catch (e: any) {
        saveError.value = `Replicas saved but scale failed: ${e?.message || 'unknown error'}`
      }
    }
  } catch (e: any) {
    saveError.value = e?.message || 'Failed to save configuration'
  } finally {
    saving.value = false
  }
}

// ─── Root domains (dropdown for subdomain picker) ─────────────────────────
const rootDomains = computed(() => store.rootDomains)
const selectedRootDomainId = ref('')
function availableRootDomains(): { id: string; host: string }[] {
  const roots = rootDomains.value
  if (selectedRootDomainId.value && !roots.some((r) => r.id === selectedRootDomainId.value)) {
    const missing = roots.find((r) => r.host === selectedRootDomainId.value)
    if (!missing) return roots
  }
  return roots
}
function selectedRootDomainHost(): string {
  return rootDomains.value.find((r) => r.id === selectedRootDomainId.value)?.host ?? ''
}

// ─── Service domains (subdomain + root domain FK + force-https) ───────────
const newDomain = reactive({ subdomain: '', isForceHTTPS: false })
const domainError = ref('')
const editingDomainId = ref<string | null>(null)
function buildDomainHost(): string {
  const sub = newDomain.subdomain.trim().replace(/^\.+/, '')
  const root = selectedRootDomainHost()
  if (!root) return ''
  return sub ? `${sub}.${root}` : root
}
async function addOrUpdateDomain() {
  const host = buildDomainHost()
  const rootId = selectedRootDomainId.value
  if (!rootId || !host) {
    domainError.value = 'Pick a root domain first (Domains menu), then type a subdomain.'
    return
  }
  // Unique check: same subdomain on the same root domain not allowed twice.
  const svc = service.value
  const exists = (svc?.domains ?? []).some(
    (d) => d.domainId === rootId && d.subdomain === newDomain.subdomain.trim() && d.id !== editingDomainId.value,
  )
  if (exists) {
    domainError.value = `Subdomain already in use for this domain: ${newDomain.subdomain.trim()}`
    return
  }
  domainError.value = ''
  try {
    if (editingDomainId.value) {
      await store.updateServiceDomain(projectId.value, envId.value, serviceId.value, editingDomainId.value, {
        subdomain: newDomain.subdomain.trim(),
        is_force_https: newDomain.isForceHTTPS,
      })
    } else {
      await store.addServiceDomain(projectId.value, envId.value, serviceId.value, {
        domain_id: rootId,
        subdomain: newDomain.subdomain.trim(),
        is_force_https: newDomain.isForceHTTPS,
      })
    }
    resetDomainForm()
  } catch (e: any) {
    domainError.value = e?.message || 'Failed to save domain'
  }
}
function editDomain(d: { id: string; subdomain: string; domainId: string; isForceHTTPS: boolean }) {
  editingDomainId.value = d.id
  newDomain.subdomain = d.subdomain ?? ''
  selectedRootDomainId.value = d.domainId
  newDomain.isForceHTTPS = !!d.isForceHTTPS
}
function resetDomainForm() {
  editingDomainId.value = null
  newDomain.subdomain = ''
  newDomain.isForceHTTPS = false
  selectedRootDomainId.value = ''
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
const showStopConfirm = ref(false)
const stopping = ref(false)
const stopError = ref('')
async function confirmStop() {
  stopError.value = ''
  // open the custom confirmation dialog instead of a native confirm
  showStopConfirm.value = true
}
async function doStop() {
  if (!service.value) return
  stopping.value = true
  stopError.value = ''
  try {
    await store.stop(projectId.value, envId.value, serviceId.value)
    showStopConfirm.value = false
  } catch (e: any) {
    stopError.value = e?.message || 'Gagal menghentikan service'
  } finally {
    stopping.value = false
  }
}

function action(a: 'start' | 'stop' | 'restart') {
  if (!service.value) return
  if (a === 'start') void store.start(projectId.value, envId.value, serviceId.value)
  if (a === 'stop') void store.stop(projectId.value, envId.value, serviceId.value)
  if (a === 'restart') void store.restart(projectId.value, envId.value, serviceId.value)
}

// ─── Terminal (accordion per replica/container, one xterm per accordion) ──
// Each expanded accordion creates its OWN xterm Terminal + WS to
// /api/ws/terminal/:serviceId/:containerName
interface TermSlot {
  term: Terminal | null
  fit: FitAddon | null
  ws: WebSocket | null
  ro: ResizeObserver | null
}
const termSlots = new Map<string, TermSlot>()

function initTerminalFor(c: LogContainer, _attempt = 0) {
  const el = document.getElementById(`term-${c.id}`)
  if (!el) {
    // v-if render may lag behind the toggle — retry briefly
    if (_attempt < 10) {
      setTimeout(() => initTerminalFor(c, _attempt + 1), 50)
    }
    return
  }
  // Idempotence guard: if a live slot already exists (e.g. watch(activeTab)
  // races with toggleTerminal's nextTick init), keep it — never dispose a
  // working terminal here. Collapse is the ONLY path that disposes.
  const live = termSlots.get(c.id)
  if (live?.term && !live.term.isDisposed) {
    live.fit?.fit()
    if (!live.ws && c.running) attachTerminalWS(c, live)
    return
  }
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    theme: {
      background: '#0e1117',
      foreground: '#e6edf3',
      cursor: '#58a6ff',
      selectionBackground: '#21262d',
    },
  })
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(el)
  // ensure the terminal can receive keyboard input — xterm only captures
  // keys once focused; without this, typing does nothing.
  setTimeout(() => { try { term.focus() } catch { /* noop */ } }, 80)
  // fit once the accordion body is actually laid out (v-if just rendered)
  requestAnimationFrame(() => {
    try { fit.fit() } catch { /* noop */ }
  })
  // re-fit whenever the host resizes (window resize, rotate, sidebar toggle)
  // — without this the terminal keeps its initial wide layout and blows the
  // page width on phones. Debounced: fit() resizes the xterm which itself
  // triggers another ResizeObserver callback → without debounce this loops
  // forever ("ResizeObserver loop completed with undelivered notifications").
  let roTimer: ReturnType<typeof setTimeout> | null = null
  let roLastW = 0
  const ro = new ResizeObserver((entries) => {
    const w = entries[0]?.contentRect?.width ?? 0
    if (Math.abs(w - roLastW) < 4) return // no meaningful change — skip
    roLastW = w
    if (roTimer) clearTimeout(roTimer)
    roTimer = setTimeout(() => {
      try { fit.fit() } catch { /* noop */ }
    }, 120)
  })
  ro.observe(el)
  const slot: TermSlot = { term, fit, ws: null }
  slot.ro = ro
  termSlots.set(c.id, slot)
  // always attach the input bridge — even before WS opens. onData buffers
  // into the WS once open; xterm key handling (typing/enter) works locally
  // regardless of connection state.
  term.onData((data) => {
    if (slot.ws?.readyState === WebSocket.OPEN) slot.ws.send(data)
  })
  attachTerminalWS(c, slot)
}

// open the per-replica terminal WS and wire it to the slot's xterm
function attachTerminalWS(c: LogContainer, slot: TermSlot) {
  const { term } = slot
  if (!c.running) {
    term?.writeln('\x1b[33mContainer stopped — start it to open a terminal.\x1b[0m')
    return
  }
  // A WS stuck in CONNECTING/CLOSING/CLOSED is dead — drop it and reconnect.
  if (slot.ws) {
    if (slot.ws.readyState === WebSocket.OPEN) return // already live
    try { slot.ws.close() } catch { /* noop */ }
    slot.ws = null
  }
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  // connect to the SPECIFIC replica container (per-accordion WS)
  const url = `${proto}//${location.host}/api/ws/terminal/${encodeURIComponent(serviceId.value)}/${encodeURIComponent(c.name)}?token=${encodeURIComponent(auth?.token ?? '')}`
  try { slot.ws = new WebSocket(url) } catch { slot.ws = null }
  if (slot.ws) {
    slot.ws.onopen = () => {
      // re-fit — the div may have been 0-sized when first mounted (v-if)
      setTimeout(() => {
        try { slot.fit?.fit() } catch { /* noop */ }
      }, 50)
      term?.writeln('\x1b[36m─── golify terminal ───\x1b[0m')
      term?.writeln(`connected to replica: ${c.replicaId || c.name} via WS`)
      term?.writeln('')
    }
    slot.ws.onmessage = (ev) => term?.write(String(ev.data))
    slot.ws.onclose = () => term?.writeln('\r\n\x1b[31m[connection closed]\x1b[0m')
    slot.ws.onerror = () => term?.writeln('\r\n\x1b[31m[websocket error]\x1b[0m')
  } else {
    term?.writeln('\x1b[31mWS not available\x1b[0m')
  }
}
function closeTermSlot(c: LogContainer) {
  const slot = termSlots.get(c.id)
  if (slot?.ws) { slot.ws.close(); slot.ws = null }
}
function disposeTermSlot(c: LogContainer) {
  const slot = termSlots.get(c.id)
  if (slot) {
    slot.ws?.close()
    slot.ro?.disconnect()
    slot.term?.dispose()
    termSlots.delete(c.id)
  }
}
function closeAllTermSlots() {
  for (const slot of termSlots.values()) {
    slot.ws?.close()
    slot.ro?.disconnect()
    slot.term?.dispose()
  }
  termSlots.clear()
}
// expand/collapse a terminal accordion.
// Closing = close WS + dispose xterm + drop the slot (fresh reconnect next
// open), mirroring the Logs accordion lifecycle.
function toggleTerminal(c: LogContainer) {
  c.expanded = !c.expanded
  if (c.expanded) {
    disposeTermSlot(c) // make sure any stale slot is gone
    // wait for the v-if to render the #term-<id> div before mounting xterm
    nextTick(() => {
      setTimeout(() => initTerminalFor(c), 30)
    })
  } else {
    disposeTermSlot(c)
  }
}
watch(activeTab, (tab) => {
  if (tab === 'terminal') {
    // Reset accordion state when arriving — nothing auto-opens. The user
    // clicks an accordion to open its WS; no connection until then.
    for (const c of containers.value) {
      c.expanded = false
      c.ws?.close()
      c.ws = null
    }
    setTimeout(() => {
      void loadContainers(true)
      // init any expanded terminal accordions (after containers load)
      setTimeout(() => {
        for (const c of containers.value) {
          if (c.expanded) {
            initTerminalFor(c)
            termSlots.get(c.id)?.fit?.fit()
          }
        }
      }, 300)
    }, 0)
  } else {
    closeAllTermSlots()
  }
  if (tab === 'logs') {
    // Reset accordion state on arrival — no auto-open, no accumulated WS.
    for (const c of containers.value) {
      c.expanded = false
      c.ws?.close()
      c.ws = null
    }
    setTimeout(() => connectLogs(), 0)
  }
})
// When the service flips to running and we're on the Logs tab, connect.
watch(() => service.value?.status, (s) => {
  if (s === 'running' && activeTab.value === 'logs') {
    setTimeout(() => connectLogs(), 0)
  }
})
onBeforeUnmount(() => {
  closeAllLogWS()
  closeAllTermSlots()
})

// ─── Logs: accordion per replica/container, each with its OWN websocket ────
interface LogContainer {
  id: string
  name: string
  replicaId: string   // short container id — displayed as the replica id
  status: string
  running: boolean
  ports: string
  expanded: boolean
  lines: string[]      // accumulated log lines for this container
  ws: WebSocket | null
  loading: boolean
  error: string
  linesLimit: number   // Lines selector (100/200/500/1000)
  autoScroll: boolean  // arrow-down toggle: auto-scroll to bottom on new lines
}

const containers = ref<LogContainer[]>([])
const containersLoading = ref(false)
const containersError = ref('')
const logSearch = ref('')
const linesOptions = [100, 200, 500, 1000]

// fetch replica containers for the service (podman ps, filtered by golify-<name>)
async function loadContainers(resetExpanded = false) {
  containersLoading.value = true
  try {
    const auth = getAuth()
    const res = await fetch(`/api/v1/projects/${projectId.value}/environments/${envId.value}/services/${serviceId.value}/containers`, {
      headers: { Authorization: `Bearer ${auth?.token ?? ''}` },
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const rows: any[] = await res.json()
    // merge with existing ws state (keep ws open if already expanded)
    const existing = new Map(containers.value.map(c => [c.id, c]))
    containers.value = rows.map(r => {
      const prev = existing.get(r.id)
      return {
        id: r.id,
        name: r.name,
        replicaId: r.replica_id || '',
        status: r.status || '',
        running: r.running,
        ports: r.ports,
        expanded: resetExpanded ? false : (prev?.expanded ?? false),
        lines: prev?.lines ?? [],
        ws: resetExpanded ? null : (prev?.ws ?? null),
        loading: false,
        error: prev?.error ?? '',
        linesLimit: prev?.linesLimit ?? 100,
        autoScroll: prev?.autoScroll ?? true,
      }
    })
    // close WS for containers that disappeared
    for (const [id, c] of existing) {
      if (!rows.some(r => r.id === id)) c.ws?.close()
    }
  } catch (e: any) {
    containersError.value = e?.message ?? 'Failed to load containers'
  } finally {
    containersLoading.value = false
  }
}

// toggle accordion: expand → connect WS; collapse → close WS
function toggleContainer(c: LogContainer) {
  c.expanded = !c.expanded
  if (c.expanded) {
    connectContainerLogs(c)
  } else {
    c.ws?.close()
    c.ws = null
  }
}

// open ONE websocket per container (replica) streaming podman logs -f
function connectContainerLogs(c: LogContainer) {
  if (!service.value || !c.running) return
  c.ws?.close()
  c.lines = []
  c.loading = true
  c.error = ''
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/api/ws/log/${encodeURIComponent(serviceId.value)}/${encodeURIComponent(c.id)}?token=${encodeURIComponent(auth?.token ?? '')}`
  try { c.ws = new WebSocket(url) } catch { c.ws = null }
  if (!c.ws) {
    c.error = '[WS not available]'
    c.loading = false
    return
  }
  c.ws.onopen = () => { c.loading = false }
  c.ws.onmessage = (ev) => {
    c.lines = [...c.lines, String(ev.data)]
    if (c.lines.length > 500) c.lines = c.lines.slice(-500)
    // auto-scroll to bottom if the toggle is on
    if (c.autoScroll) {
      requestAnimationFrame(() => {
        const el = document.getElementById(`log-pre-${c.id}`)
        if (el) el.scrollTop = el.scrollHeight
      })
    }
  }
  c.ws.onerror = () => {
    c.error = '[connection error]'
    c.loading = false
  }
  c.ws.onclose = () => {
    c.loading = false
    c.ws = null
  }
}

function closeAllLogWS() {
  for (const c of containers.value) {
    c.ws?.close()
    c.ws = null
  }
}

const filteredLines = (c: LogContainer) =>
  logSearch.value ? c.lines.filter(l => l.toLowerCase().includes(logSearch.value.toLowerCase())) : c.lines

// apply the Lines selector: show only the last N lines of the filtered output
const visibleLines = (c: LogContainer) => {
  const all = filteredLines(c)
  return all.slice(-c.linesLimit)
}

function copyContainerLog(c: LogContainer) {
  navigator.clipboard?.writeText(c.lines.join('\n')).catch(() => {})
}
function downloadContainerLog(c: LogContainer) {
  const blob = new Blob([c.lines.join('\n')], { type: 'text/plain' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${c.name}.log`
  a.click()
  URL.revokeObjectURL(a.href)
}
function refreshContainerLog(c: LogContainer) {
  c.lines = []
  connectContainerLogs(c)
}
function fullscreenContainerLog(c: LogContainer) {
  const el = document.getElementById(`log-${c.id}`)
  if (!el) return
  if (document.fullscreenElement) {
    document.exitFullscreen().catch(() => {})
  } else {
    el.requestFullscreen?.().catch(() => {})
  }
}

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

// Open a deployment row → navigate to the dedicated deployment log page.
function openDeployLog(dep: Deployment) {
  router.push(`/project/${projectId.value}/environment/${envId.value}/service/${serviceId.value}/deploy/${dep.id}`)
}

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
  // containers needed for Logs & Terminal accordions (load once on mount)
  void loadContainers()
})

// refresh deployments whenever the tab becomes active (new deploys appear)
watch(activeTab, (tab) => {
  if (tab === 'deployments') loadDeployments()
})

// ─── Logs (live, gated by service status) ─────────────────────────────────
function connectLogs() {
  // now: fetch replica containers; WS opens per-accordion on expand
  void loadContainers()
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
      <RouterLink :to="`/project/${project.id}/environments?envs=1`" class="hover:text-foreground">
        <Layers class="inline size-3" /> {{ env.name }}
      </RouterLink>
      <span>/</span>
      <RouterLink :to="`/project/${project.id}/environment/${env.id}/services`" class="flex items-center gap-1 truncate hover:text-foreground">
        <Box class="inline size-3 shrink-0" /> <span class="truncate">{{ service.name }}</span>
      </RouterLink>
    </div>

    <!-- Header: title + status + actions -->
    <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-2xl font-semibold tracking-tight">{{ service.name }}</h1>
        <p class="mt-0.5 flex items-center gap-2 text-sm">
          <span class="inline-block size-2 rounded-full" :class="service.status === 'running' ? 'bg-green-500' : 'bg-yellow-500'" />
          <span :class="service.status === 'running' ? 'text-green-500' : 'text-yellow-500'">{{ service.status }} (unknown)</span>
          <Badge v-if="service.loadBalancer === 'k8s'" variant="outline" class="border-sky-500/50 text-sky-500">K8s</Badge>
          <Info class="ml-1 size-3 text-muted-foreground" />
          <RotateCw class="size-3 text-yellow-500" />
        </p>
        <p class="truncate font-mono text-xs text-muted-foreground">
          {{ service.kind }} · {{ service.image || service.composePath || '—' }}:{{ service.imageTag || 'latest' }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button v-if="service.status !== 'running'" size="sm" @click="deployNow">
          <Play class="mr-1 size-4" />Start
        </Button>
        <Button v-if="service.status === 'running'" size="sm" variant="destructive" @click="confirmStop">
          <Square class="mr-1 size-4" />Stop
        </Button>
        <Button size="sm" variant="outline" @click="action('restart')">
          <RotateCw class="mr-1 size-4" />Restart
        </Button>
      </div>
    </header>

    <!-- Stop confirmation dialog -->
    <Dialog v-model:open="showStopConfirm">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <Square class="size-4" /> Stop Service
          </DialogTitle>
          <DialogDescription>
            Yakin ingin menghentikan service <span class="font-medium text-foreground">{{ service.name }}</span>?
            <br />Semua request ke service ini akan berhenti berjalan.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-2">
          <Button variant="outline" :disabled="stopping" @click="showStopConfirm = false">Batal</Button>
          <Button variant="destructive" :disabled="stopping" @click="doStop">
            <Loader2 v-if="stopping" class="mr-1 size-4 animate-spin" />
            <Square v-else class="mr-1 size-4" />
            {{ stopping ? 'Menghentikan…' : 'Ya, Stop' }}
          </Button>
        </DialogFooter>
        <p v-if="stopError" class="text-sm text-destructive">{{ stopError }}</p>
      </DialogContent>
    </Dialog>

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
          <p v-if="scaleNote" class="mb-2 text-sm text-emerald-600">{{ scaleNote }}</p>

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

            <!-- Domains (col 6) — subdomain + root domain FK + force-https -->
            <Card class="col-span-12 md:col-span-6">
              <CardHeader class="pb-2">
                <CardDescription class="flex items-center gap-1">
                  Domains <Info class="size-3" />
                </CardDescription>
                <div class="flex flex-col gap-2">
                  <div class="flex gap-2">
                    <Input v-model="newDomain.subdomain" placeholder="subdomain (opsional)" class="flex-1" />
                    <select
                      v-model="selectedRootDomainId"
                      class="h-9 w-44 rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <option value="">— root domain —</option>
                      <option v-for="d in availableRootDomains()" :key="d.id" :value="d.id">.{{ d.host }}</option>
                    </select>
                  </div>
                  <div class="flex items-center justify-between gap-2">
                    <label class="flex items-center gap-2 text-xs text-muted-foreground">
                      <input v-model="newDomain.isForceHTTPS" type="checkbox" class="size-3.5 accent-primary" />
                      Force HTTPS
                    </label>
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
                  <p class="text-xs text-muted-foreground">
                    Preview: <span class="font-mono">{{ buildDomainHost() || '—' }}</span>
                  </p>
                </div>
                <p v-if="domainError" class="text-xs text-destructive">{{ domainError }}</p>
              </CardHeader>
              <CardContent class="grid gap-1.5">
                <div v-for="d in service.domains ?? []" :key="d.id" class="flex items-center justify-between gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
                  <span class="flex items-center gap-2">
                    <a
                      :href="(d.isForceHTTPS ? 'https://' : 'http://') + d.host"
                      target="_blank"
                      rel="noopener"
                      class="truncate font-mono hover:text-primary hover:underline"
                    >
                      {{ d.isForceHTTPS ? 'https://' : 'http://' }}{{ d.host }}
                    </a>
                    <Badge v-if="d.isForceHTTPS" variant="secondary">HTTPS</Badge>
                  </span>
                  <span class="flex items-center gap-2">
                    <button class="text-muted-foreground hover:text-foreground" @click="editDomain(d)"><Pencil class="size-3.5" /></button>
                    <button class="text-destructive hover:text-destructive/80" @click="removeDomain(d.id)"><Trash2 class="size-3.5" /></button>
                  </span>
                </div>
                <p v-if="!(service.domains ?? []).length" class="text-xs text-muted-foreground">No domains yet — pick a root domain (registered in the Domains menu) and a subdomain.</p>
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
                <!-- Load Balancer — only meaningful with more than one replica -->
                <div v-if="lbVisible" class="grid gap-1.5">
                  <Label class="flex items-center gap-1">Load Balancer <Info class="size-3 text-muted-foreground" /></Label>
                  <select
                    v-model="form.loadBalancer"
                    class="h-9 w-full rounded-md border bg-background px-2.5 py-1 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring dark:bg-input/30"
                  >
                    <option value="round_robin">Round Robin</option>
                    <option value="least_conn">Least Connection</option>
                    <option value="k8s">Kubernetes (Ingress)</option>
                  </select>
                  <p class="text-xs text-muted-foreground">Distribution strategy across {{ lbCount }} replicas.</p>
                </div>
              </CardContent>
            </Card>
          </div>
        </template>

            <template v-else-if="activeSection === 'envvars'">
              <div class="mb-3 flex items-center justify-between gap-2">
                <div>
                  <h2 class="text-lg font-semibold">Environment Variables</h2>
                  <p class="text-xs text-muted-foreground">Edit .env style key=value pairs for this service.</p>
                </div>
                <Button size="sm" variant="outline" @click="addEnvVarRow">
                  <Plus class="mr-1 size-4" /> Add Variable
                </Button>
              </div>

              <Card>
                <CardHeader class="pb-2">
                  <CardTitle class="text-base">.env Editor</CardTitle>
                  <CardDescription>One variable per line — KEY=VALUE</CardDescription>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <!-- .env code editor (Monaco) -->
                  <div class="relative overflow-hidden rounded-lg border border-zinc-800">
                    <div ref="envEditorEl" class="h-80 w-full" style="min-height: 320px" />
                  </div>
                  <div class="flex items-center justify-between gap-2">
                    <p class="text-xs text-muted-foreground">{{ envVarCount }} variable(s) · build: 0</p>
                    <div class="flex gap-2">
                      <Button size="sm" variant="ghost" :disabled="!envText" @click="envText = ''">Clear</Button>
                      <Button size="sm" :disabled="savingEnv" @click="saveEnvText">
                        <Loader2 v-if="savingEnv" class="mr-1 size-4 animate-spin" />
                        <Save v-else class="mr-1 size-4" />
                        {{ envSaved ? 'Saved ✓' : 'Save' }}
                      </Button>
                    </div>
                  </div>
                  <p v-if="envError" class="text-xs text-destructive">{{ envError }}</p>

                  <!-- Table of stored variables (parsed from env_var string) -->
                  <div v-if="parsedEnvVars.length" class="mt-2 overflow-x-auto">
                    <table class="w-full text-sm">
                      <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
                        <tr>
                          <th class="px-3 py-2">Key</th>
                          <th class="px-3 py-2">Value</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-border">
                        <tr v-for="v in parsedEnvVars" :key="v.key" class="hover:bg-muted/30">
                          <td class="px-3 py-2 font-mono text-[13px]">{{ v.key }}</td>
                          <td class="max-w-[280px] truncate px-3 py-2 font-mono text-[13px] text-muted-foreground">{{ v.value }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </template>

            <template v-else-if="activeSection === 'storage'">
              <div class="mb-3 flex items-center justify-between gap-2">
                <div>
                  <h2 class="text-lg font-semibold">Persistent Storage</h2>
                  <p class="text-xs text-muted-foreground">Volumes mounted into the service container.</p>
                </div>
                <Button size="sm" variant="outline" @click="addStorageRow">
                  <Plus class="mr-1 size-4" /> Add Storage
                </Button>
              </div>

              <Card>
                <CardHeader class="pb-2">
                  <CardTitle class="text-base">Volumes</CardTitle>
                  <CardDescription>Mount a volume at a container path.</CardDescription>
                </CardHeader>
                <CardContent class="grid gap-3">
                  <div class="grid grid-cols-12 gap-2">
                    <div class="col-span-12 md:col-span-3">
                      <Label>Name</Label>
                      <Input v-model="storageForm.name" placeholder="data" />
                    </div>
                    <div class="col-span-12 md:col-span-4">
                      <Label>Mount Path</Label>
                      <Input v-model="storageForm.mountPath" placeholder="/data" />
                    </div>
                    <div class="col-span-12 md:col-span-3">
                      <Label>Host Path (optional)</Label>
                      <Input v-model="storageForm.hostPath" placeholder="/var/lib/golify/..." />
                    </div>
                    <div class="col-span-12 md:col-span-2 flex items-end">
                      <Button class="w-full" :disabled="savingStorage" @click="createStorage">
                        <Plus class="mr-1 size-4" /> Add
                      </Button>
                    </div>
                  </div>
                  <p v-if="storageError" class="text-xs text-destructive">{{ storageError }}</p>

                  <div v-if="service.storages?.length" class="mt-2 overflow-x-auto">
                    <table class="w-full text-sm">
                      <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
                        <tr>
                          <th class="px-3 py-2">Name</th>
                          <th class="px-3 py-2">Mount Path</th>
                          <th class="px-3 py-2">Host Path</th>
                          <th class="px-3 py-2 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-border">
                        <tr v-for="st in service.storages" :key="st.id" class="hover:bg-muted/30">
                          <td class="px-3 py-2 font-mono text-[13px]">{{ st.name }}</td>
                          <td class="px-3 py-2 font-mono text-[13px]">{{ st.mountPath }}</td>
                          <td class="px-3 py-2 font-mono text-[13px] text-muted-foreground">{{ st.hostPath || '—' }}</td>
                          <td class="px-3 py-2 text-right">
                            <button class="text-muted-foreground hover:text-foreground" @click="removeStorage(st.id)"><Trash2 class="size-3.5" /></button>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
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
          class="cursor-pointer rounded-md border border-l-4 bg-card p-3 transition-colors hover:bg-accent/50"
          :class="statusClass(dep.status)"
          role="button"
          tabindex="0"
          @click="openDeployLog(dep)"
          @keydown.enter="openDeployLog(dep)"
          @keydown.space.prevent="openDeployLog(dep)"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-3">
              <span class="inline-block size-2 rounded-full" :class="dep.status === 'success' ? 'bg-green-500' : dep.status === 'failed' ? 'bg-red-500' : 'bg-yellow-500 animate-pulse'" />
              <span class="text-sm font-medium" :class="dep.status === 'success' ? 'text-green-500' : dep.status === 'failed' ? 'text-red-500' : 'text-yellow-500'">
                {{ dep.status === 'running' ? 'Running' : dep.status === 'success' ? 'Success' : 'Failed' }}
              </span>
            </div>
            <ChevronRight class="size-4 text-muted-foreground" />
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
        </div>
        <p v-if="!deployments.length" class="text-sm text-muted-foreground">No deployments yet.</p>
      </div>
    </div>

    <!-- Logs: accordion per replica/container, one WS per accordion -->
    <div v-else-if="activeTab === 'logs'">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-lg font-semibold">Logs</h2>
        <Button size="sm" variant="outline" @click="loadContainers">
          <RotateCw class="mr-1 size-3" />Refresh
        </Button>
      </div>
      <p v-if="service.status !== 'running'" class="mt-1 text-sm text-muted-foreground">
        Service is <span class="text-yellow-500">{{ service.status }}</span> — start the service to see live logs.
      </p>

      <div v-if="containersLoading" class="mt-3 text-sm text-muted-foreground">Loading containers…</div>
      <p v-else-if="containersError" class="mt-3 text-sm text-destructive">{{ containersError }}</p>

      <div v-else-if="containers.length" class="mt-3 grid gap-2">
        <div v-for="c in containers" :key="c.id" class="overflow-hidden rounded-md border bg-card">
          <!-- accordion header -->
          <button
            type="button"
            class="flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left transition-colors hover:bg-accent/50"
            @click="toggleContainer(c)"
          >
            <span class="flex min-w-0 items-center gap-2">
              <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="c.expanded ? '' : '-rotate-90'" />
              <span class="inline-block size-2 shrink-0 rounded-full" :class="c.running ? 'bg-green-500' : 'bg-yellow-500'" />
              <span class="truncate font-mono text-sm">{{ c.replicaId || c.name }}</span>
            </span>
            <span class="flex shrink-0 items-center gap-2 text-xs text-muted-foreground">
              <span v-if="c.ports" class="hidden sm:inline">{{ c.ports }}</span>
              <span :class="c.running ? 'text-green-500' : 'text-yellow-500'">{{ c.running ? 'running' : 'stopped' }}</span>
            </span>
          </button>

          <!-- accordion body: one WS per container -->
          <div v-if="c.expanded" :id="`log-${c.id}`" class="border-t bg-muted/30">
            <div class="flex flex-wrap items-center gap-2 border-b px-3 py-1.5">
              <!-- Lines selector -->
              <select
                v-model.number="c.linesLimit"
                class="h-7 rounded border bg-background px-1.5 font-mono text-xs outline-none"
                title="Lines"
              >
                <option v-for="n in linesOptions" :key="n" :value="n">Lines: {{ n }}</option>
              </select>

              <!-- Find in logs -->
              <input
                v-model="logSearch"
                placeholder="Find in logs"
                class="ml-auto h-7 w-40 rounded border bg-background px-2 font-mono text-xs outline-none"
              />

              <!-- toolbar icons (Coolify-style) -->
              <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Refresh" @click="refreshContainerLog(c)">
                <RefreshCw class="size-3.5" />
              </button>
              <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Stream (pause/resume)" @click="c.ws ? (c.ws.close(), c.ws = null) : connectContainerLogs(c)">
                <Play class="size-3.5" />
              </button>
              <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Copy" @click="copyContainerLog(c)">
                <Copy class="size-3.5" />
              </button>
              <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Download" @click="downloadContainerLog(c)">
                <Download class="size-3.5" />
              </button>
              <!-- auto-scroll toggle: arrow-down (active = follow new logs) -->
              <button
                class="rounded p-1 transition-colors"
                :class="c.autoScroll ? 'bg-primary/20 text-primary' : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                :title="c.autoScroll ? 'Auto-scroll: on' : 'Auto-scroll: off'"
                @click="c.autoScroll = !c.autoScroll; if (c.autoScroll) requestAnimationFrame(() => { const el = document.getElementById(`log-pre-${c.id}`); if (el) el.scrollTop = el.scrollHeight })"
              >
                <ChevronDown class="size-3.5" />
              </button>
              <button class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Fullscreen" @click="fullscreenContainerLog(c)">
                <Maximize class="size-3.5" />
              </button>
            </div>
            <div v-if="c.loading" class="px-3 py-2 text-xs text-muted-foreground">Connecting…</div>
            <pre :id="`log-pre-${c.id}`" v-else class="max-h-80 overflow-auto p-3 font-mono text-xs leading-relaxed"><code>{{ visibleLines(c).join('\n') || c.error || 'No log output yet.' }}</code></pre>
          </div>
        </div>
      </div>

      <div v-else-if="service.status === 'running'" class="mt-3 flex flex-col items-center gap-2 rounded bg-muted p-6 text-center">
        <ScrollText class="size-6 text-muted-foreground" />
        <p class="text-sm text-muted-foreground">No containers found — deploy the service first.</p>
      </div>
    </div>

    <!-- Terminal: accordion per replica/container, one xterm per accordion -->
    <div v-else-if="activeTab === 'terminal'">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-lg font-semibold">Terminal</h2>
        <Button variant="outline" size="sm" class="gap-1.5" @click="loadContainers()">
          <RefreshCw class="size-3.5" /> Refresh
        </Button>
      </div>
      <div class="mt-3 space-y-2">
        <div v-for="c in containers" :key="c.id" class="overflow-hidden rounded-md border">
          <button
            class="flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left transition-colors hover:bg-accent/50"
            @click="toggleTerminal(c)"
          >
            <span class="flex min-w-0 items-center gap-2">
              <ChevronDown class="size-4 shrink-0 text-muted-foreground transition-transform" :class="c.expanded ? '' : '-rotate-90'" />
              <span class="inline-block size-2 shrink-0 rounded-full" :class="c.running ? 'bg-green-500' : 'bg-yellow-500'" />
              <span class="truncate font-mono text-sm">{{ c.replicaId || c.name }}</span>
            </span>
            <span class="flex shrink-0 items-center gap-2 text-xs text-muted-foreground">
              <span v-if="c.ports" class="hidden sm:inline">{{ c.ports }}</span>
              <span :class="c.running ? 'text-green-500' : 'text-yellow-500'">{{ c.running ? 'running' : 'stopped' }}</span>
            </span>
          </button>
          <div v-if="c.expanded" class="border-t bg-muted/30 p-2">
            <div :id="`term-${c.id}`" class="term-host h-80 w-full max-w-full overflow-hidden rounded bg-[#0e1117]" />
          </div>
        </div>
        <div v-if="!containers.length" class="mt-3 flex flex-col items-center gap-2 rounded bg-muted p-6 text-center">
          <Box class="size-6 text-muted-foreground" />
          <p class="text-sm text-muted-foreground">No containers found — deploy the service first.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* xterm.js — constrain the terminal to its container so it never blows the
   page width on small screens (responsive bug: accordion expanded → page
   overflowed horizontally). */
.term-host :deep(.xterm) {
  width: 100% !important;
  max-width: 100%;
  overflow: hidden;
}
.term-host :deep(.xterm .xterm-viewport) {
  overflow-y: auto;
  overflow-x: hidden;
}
.term-host :deep(.xterm-screen) {
  width: 100% !important;
}
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
/* .env editor — dark terminal style, good for KEY=VALUE files */
.env-editor {
  tab-size: 4;
  line-height: 1.6;
  font-variant-ligatures: none;
}
.env-editor::placeholder {
  color: #52525b;
  opacity: 1;
}
.env-editor:focus {
  outline: none;
}
</style>
