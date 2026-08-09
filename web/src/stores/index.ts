// Domain stores. Backend-backed for Projects; others mock + localStorage.
// Replace each remaining `load()` body with an API call once the BE endpoint lands.

import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { useQuery, useMutation } from '@pinia/colada'
import type {
  ApiKey,
  Environment,
  Key,
  MCPEndpoint,
  Project,
  S3Storage,
  Server,
  Service,
  SharedVariable,
  Source,
  Team,
} from '@/lib/types'
import {
  mockApiKeys,
  mockKeys,
  mockMCP,
  mockProjects,
  mockS3,
  mockServers,
  mockSources,
  mockTeams,
  mockVars,
} from '@/lib/mock'
import { authed } from '@/lib/api'

const KEY = (name: string) => `golify:${name}`

function load<T>(name: string, fallback: T): T {
  if (typeof window === 'undefined') return fallback
  try {
    const raw = localStorage.getItem(KEY(name))
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

function persist<T>(name: string, value: T) {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(KEY(name), JSON.stringify(value))
  } catch {
    /* quota exceeded — silent */
  }
}

// ─── Projects (backend-backed) ─────────────────────────────────────────────

// BE DTOs (snake_case) as returned by the Go API
interface ProjectDTO {
  id: number
  name: string
  description: string
  source_id: string
  environments: EnvironmentDTO[]
  created_at: string
  updated_at: string
}
interface EnvironmentDTO {
  id: number
  project_id: number
  name: string
  is_production: boolean
  domains: { id: number; environment_id: number; host: string; created_at: string }[]
  services: ServiceDTO[]
  created_at: string
  updated_at: string
}
interface ServiceDTO {
  id: number
  environment_id: number
  name: string
  kind: string
  image: string
  compose_path: string
  status: string
  cpu: number
  memory: number
  ports: string[]
  created_at: string
  updated_at: string
}

function mapEnv(e: EnvironmentDTO): Environment {
  return {
    id: String(e.id),
    name: e.name,
    isProduction: e.is_production,
    services: (e.services ?? []).map(mapSvc),
    domains: (e.domains ?? []).map((d) => d.host),
  }
}
function mapSvc(s: ServiceDTO): Service {
  return {
    id: String(s.id),
    name: s.name,
    kind: s.kind === 'compose' ? 'compose' : 'container',
    image: s.image || undefined,
    composePath: s.compose_path || undefined,
    status: (s.status as Service['status']) ?? 'stopped',
    cpu: s.cpu,
    memory: s.memory,
    ports: s.ports ?? [],
  }
}
function mapProject(p: ProjectDTO): Project {
  return {
    id: String(p.id),
    name: p.name,
    description: p.description,
    sourceId: p.source_id || undefined,
    environments: (p.environments ?? []).map(mapEnv),
    createdAt: p.created_at,
  }
}

export const useProjectsStore = defineStore('projects', () => {
  // Colada query — fetches from BE, falls back to mock on failure.
  const q = useQuery<Project[], Error>({
    key: ['projects'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/projects/').json<ProjectDTO[]>()
        return rows.map(mapProject)
      } catch {
        // No backend / no auth yet — use local mock so the UI still works.
        return mockProjects
      }
    },
    initialData: mockProjects,
    staleTime: 30_000,
  })

  const projects = computed(() => q.data.value ?? [])

  function get(id: string): Project | undefined {
    return projects.value.find((p) => p.id === id)
  }
  function getEnv(projectId: string, envId: string): Environment | undefined {
    return get(projectId)?.environments.find((e) => e.id === envId)
  }
  function getService(projectId: string, envId: string, serviceId: string): Service | undefined {
    return getEnv(projectId, envId)?.services.find((s) => s.id === serviceId)
  }
  function start(projectId: string, envId: string, serviceId: string) {
    const s = getService(projectId, envId, serviceId)
    if (s) s.status = 'building'
  }
  function stop(projectId: string, envId: string, serviceId: string) {
    const s = getService(projectId, envId, serviceId)
    if (s) s.status = 'stopped'
  }

  const create = useMutation({
    mutation: (input: { name: string; description?: string; sourceId?: string }) =>
      authed().post('api/v1/projects/', { json: input }).json<ProjectDTO>(),
  })

  return { projects, get, getEnv, getService, start, stop, create, ...q }
})

// ─── Servers ───────────────────────────────────────────────────────────────

export const useServersStore = defineStore('servers', () => {
  const servers = ref<Server[]>(load('servers', mockServers))
  watch(servers, (v) => persist('servers', v), { deep: true })

  function get(id: string) {
    return servers.value.find((s) => s.id === id)
  }
  const onlineCount = computed(() => servers.value.filter((s) => s.status === 'online').length)

  return { servers, get, onlineCount }
})

// ─── Sources ───────────────────────────────────────────────────────────────

export const useSourcesStore = defineStore('sources', () => {
  const sources = ref<Source[]>(load('sources', mockSources))
  watch(sources, (v) => persist('sources', v), { deep: true })

  function add(s: Omit<Source, 'id' | 'createdAt'>) {
    sources.value.push({ ...s, id: `src_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function remove(id: string) {
    sources.value = sources.value.filter((s) => s.id !== id)
  }

  return { sources, add, remove }
})

// ─── S3 storages ───────────────────────────────────────────────────────────

export const useS3Store = defineStore('s3', () => {
  const items = ref<S3Storage[]>(load('s3', mockS3))
  watch(items, (v) => persist('s3', v), { deep: true })

  function add(s: Omit<S3Storage, 'id' | 'createdAt'>) {
    items.value.push({ ...s, id: `s3_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function remove(id: string) {
    items.value = items.value.filter((s) => s.id !== id)
  }
  function setDefault(id: string) {
    items.value = items.value.map((s) => ({ ...s, isDefault: s.id === id }))
  }

  return { items, add, remove, setDefault }
})

// ─── Shared Variables ──────────────────────────────────────────────────────

export const useVarsStore = defineStore('vars', () => {
  const items = ref<SharedVariable[]>(load('vars', mockVars))
  watch(items, (v) => persist('vars', v), { deep: true })

  function add(v: Omit<SharedVariable, 'id' | 'updatedAt'>) {
    items.value.push({ ...v, id: `v_${Date.now()}`, updatedAt: new Date().toISOString() })
  }
  function update(id: string, patch: Partial<SharedVariable>) {
    const i = items.value.findIndex((x) => x.id === id)
    if (i >= 0) {
      items.value[i] = { ...items.value[i], ...patch, updatedAt: new Date().toISOString() }
    }
  }
  function remove(id: string) {
    items.value = items.value.filter((x) => x.id !== id)
  }
  function scopeItems(scope: SharedVariable['scope'], scopeRef?: string) {
    return items.value.filter((v) => v.scope === scope && (!scopeRef || v.scopeRef === scopeRef))
  }

  return { items, add, update, remove, scopeItems }
})

// ─── Keys (SSH) ────────────────────────────────────────────────────────────

export const useKeysStore = defineStore('keys', () => {
  const keys = ref<Key[]>(load('keys', mockKeys))
  watch(keys, (v) => persist('keys', v), { deep: true })

  function add(k: Omit<Key, 'id' | 'createdAt'>) {
    keys.value.push({ ...k, id: `key_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function remove(id: string) {
    keys.value = keys.value.filter((k) => k.id !== id)
  }

  return { keys, add, remove }
})

// ─── API Keys & MCP ────────────────────────────────────────────────────────

export const useApiMcpStore = defineStore('apiMcp', () => {
  const apiKeys = ref<ApiKey[]>(load('apiKeys', mockApiKeys))
  const mcp = ref<MCPEndpoint[]>(load('mcp', mockMCP))
  watch([apiKeys, mcp], ([a, m]) => {
    persist('apiKeys', a)
    persist('mcp', m)
  })

  function addApiKey(k: Omit<ApiKey, 'id' | 'createdAt'>) {
    apiKeys.value.push({ ...k, id: `apik_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function removeApiKey(id: string) {
    apiKeys.value = apiKeys.value.filter((k) => k.id !== id)
  }
  function addMcp(m: Omit<MCPEndpoint, 'id' | 'createdAt'>) {
    mcp.value.push({ ...m, id: `mcp_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function removeMcp(id: string) {
    mcp.value = mcp.value.filter((x) => x.id !== id)
  }

  return { apiKeys, mcp, addApiKey, removeApiKey, addMcp, removeMcp }
})

// ─── Teams ─────────────────────────────────────────────────────────────────

export const useTeamsStore = defineStore('teams', () => {
  const teams = ref<Team[]>(load('teams', mockTeams))
  watch(teams, (v) => persist('teams', v), { deep: true })

  function get(id: string) {
    return teams.value.find((t) => t.id === id)
  }

  return { teams, get }
})