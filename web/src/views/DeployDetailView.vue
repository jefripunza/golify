<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useProjectsStore } from '@/stores'
import { getAuth } from '@/lib/api'
import type { Deployment } from '@/lib/types'
import {
  ArrowLeft,
  Copy,
  Download,
  RefreshCw,
  Maximize2,
  Minimize2,
  Search,
} from '@lucide/vue'

const route = useRoute()
const router = useRouter()
const store = useProjectsStore()

const projectId = computed(() => String(route.params.projectId))
const envId = computed(() => String(route.params.envId))
const serviceId = computed(() => String(route.params.serviceId))
const deployId = computed(() => String(route.params.deployId))

const project = computed(() => store.get(projectId.value))
const env = computed(() => store.getEnv(projectId.value, envId.value))
const service = computed(() => store.getService(projectId.value, envId.value, serviceId.value))

const dep = ref<Deployment | null>(null)
const logLines = ref<string[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const fullscreen = ref(false)
let ws: WebSocket | null = null

const logBodyEl = ref<HTMLElement | null>(null)

const filteredLog = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return logLines.value
  return logLines.value.filter((l) => l.toLowerCase().includes(q))
})

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    const d = await store.fetchDeployment(projectId.value, envId.value, serviceId.value, deployId.value)
    dep.value = d
    logLines.value = d.log ? d.log.split('\n') : []
    if (d.status === 'running') connectWS()
  } catch (e: any) {
    error.value = e?.message || 'Failed to load deployment'
  } finally {
    loading.value = false
  }
}

function connectWS() {
  ws?.close()
  const auth = getAuth()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${location.host}/api/ws/deploy/${encodeURIComponent(deployId.value)}?token=${encodeURIComponent(auth?.token ?? '')}`
  try { ws = new WebSocket(url) } catch { ws = null }
  if (!ws) return
  ws.onmessage = (ev) => {
    logLines.value = [...logLines.value, String(ev.data)]
  }
  ws.onclose = () => {
    // deploy finished → reload detail (status + persisted log)
    setTimeout(async () => {
      try {
        const d = await store.fetchDeployment(projectId.value, envId.value, serviceId.value, deployId.value)
        dep.value = d
        if (d.status !== 'running') logLines.value = d.log ? d.log.split('\n') : logLines.value
        await store.fetchOnce()
      } catch { /* keep */ }
    }, 300)
  }
  ws.onerror = () => { logLines.value = [...logLines.value, '[connection error]'] }
}

function copyLog() {
  navigator.clipboard?.writeText(logLines.value.join('\n')).catch(() => {})
}

function downloadLog() {
  const blob = new Blob([logLines.value.join('\n')], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `deploy-${deployId.value}.log`
  a.click()
  URL.revokeObjectURL(url)
}

function refreshLog() {
  loadDetail()
}

function goBack() {
  router.push(`/project/${projectId.value}/environment/${envId.value}/service/${serviceId.value}`)
}

function fmtDateTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${p(d.getUTCMonth() + 1)}-${p(d.getUTCDate())} ${p(d.getUTCHours())}:${p(d.getUTCMinutes())}:${p(d.getUTCSeconds())}`
}

// auto-scroll to bottom on new lines
watch(logLines, () => {
  const el = logBodyEl.value
  if (el) el.scrollTop = el.scrollHeight
}, { flush: 'post' })

onMounted(loadDetail)
onBeforeUnmount(() => ws?.close())
</script>

<template>
  <div class="flex h-full min-h-[calc(100vh-57px)] flex-col">
    <!-- Breadcrumb + back -->
    <div class="mb-3 flex items-center gap-2">
      <Button size="sm" variant="ghost" class="size-8" @click="goBack">
        <ArrowLeft class="size-4" />
      </Button>
      <div class="flex flex-wrap items-center gap-1 text-sm text-muted-foreground">
        <RouterLink :to="`/projects`" class="hover:text-foreground">{{ project?.name ?? 'project' }}</RouterLink>
        <span>/</span>
        <RouterLink :to="`/project/${projectId}/environment/${envId}/services`" class="hover:text-foreground">{{ env?.name ?? 'env' }}</RouterLink>
        <span>/</span>
        <RouterLink :to="`/project/${projectId}/environment/${envId}/service/${serviceId}`" class="hover:text-foreground">{{ service?.name ?? 'service' }}</RouterLink>
        <span>/</span>
        <span class="text-foreground">deploy {{ deployId.slice(0, 8) }}</span>
      </div>
    </div>

    <!-- Header -->
    <div class="flex items-center justify-between border-b px-4 py-3">
      <div>
        <h3 class="text-sm font-semibold">Deployment Log</h3>
        <p class="text-xs text-muted-foreground font-mono">{{ deployId }}</p>
      </div>
    </div>

    <!-- Sub-header: status + toolbar -->
    <div class="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2">
      <p class="text-xs">
        Deployment is
        <span
          class="font-medium"
          :class="dep?.status === 'success' ? 'text-green-500' : dep?.status === 'failed' ? 'text-red-500' : 'text-yellow-500'"
        >
          {{ dep?.status === 'running' ? 'Running' : dep?.status === 'success' ? 'Finished' : 'Failed' }}
        </span>
        <span v-if="dep && dep.status !== 'running'" class="ml-2 text-muted-foreground">
          · started {{ fmtDateTime(dep.startedAt) }} UTC
        </span>
      </p>
      <div class="flex items-center gap-1">
        <div class="relative">
          <Search class="absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="search" placeholder="Find in logs" class="h-7 w-44 pl-7 text-xs" />
        </div>
        <Button size="icon" variant="ghost" class="size-7" title="Copy" @click="copyLog">
          <Copy class="size-3.5" />
        </Button>
        <Button size="icon" variant="ghost" class="size-7" title="Download" @click="downloadLog">
          <Download class="size-3.5" />
        </Button>
        <Button size="icon" variant="ghost" class="size-7" title="Refresh" @click="refreshLog">
          <RefreshCw class="size-3.5" />
        </Button>
        <Button size="icon" variant="ghost" class="size-7" :title="fullscreen ? 'Exit fullscreen' : 'Fullscreen'" @click="fullscreen = !fullscreen">
          <Maximize2 v-if="!fullscreen" class="size-3.5" />
          <Minimize2 v-else class="size-3.5" />
        </Button>
      </div>
    </div>

    <!-- Log body -->
    <div
      ref="logBodyEl"
      class="flex-1 overflow-auto bg-[#0e1117] p-3 font-mono text-[11px] leading-relaxed text-gray-300"
      :class="fullscreen ? 'fixed inset-0 z-50 p-4' : ''"
    >
      <div v-if="loading" class="text-gray-500">Loading log…</div>
      <p v-else-if="error" class="text-red-400">{{ error }}</p>
      <template v-else>
        <p v-for="(line, i) in filteredLog" :key="i" class="whitespace-pre-wrap">
          <span v-if="line.match(/^20\d\d-/)" class="text-gray-500">{{ line.slice(0, 26) }}</span>
          <span v-if="line.match(/^20\d\d-/)" class="text-gray-300">{{ line.slice(26) }}</span>
          <span v-else class="text-gray-300">{{ line }}</span>
        </p>
        <p v-if="!logLines.length" class="text-gray-500">Waiting for log output…</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
/* reuse fmtDateTime from the pattern used elsewhere */
</style>
