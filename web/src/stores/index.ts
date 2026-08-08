// Domain stores. All backed by mock data + localStorage persistence.
// Replace each `load()` body with an API call once the BE endpoints exist.

import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
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

const KEY = (name: string) => `gotify:${name}`

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

// ─── Projects ──────────────────────────────────────────────────────────────

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref<Project[]>(load('projects', mockProjects))

  watch(projects, (v) => persist('projects', v), { deep: true })

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

  return { projects, get, getEnv, getService, start, stop }
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
    items.value = items.value.filter((v) => v.id !== id)
  }

  return { items, add, update, remove }
})

// ─── Keys ──────────────────────────────────────────────────────────────────

export const useKeysStore = defineStore('keys', () => {
  const items = ref<Key[]>(load('keys', mockKeys))
  watch(items, (v) => persist('keys', v), { deep: true })

  function add(k: Omit<Key, 'id' | 'createdAt'>) {
    items.value.push({ ...k, id: `key_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function remove(id: string) {
    items.value = items.value.filter((k) => k.id !== id)
  }

  return { items, add, remove }
})

// ─── API Keys & MCP ────────────────────────────────────────────────────────

export const useApiMcpStore = defineStore('api-mcp', () => {
  const apiKeys = ref<ApiKey[]>(load('apiKeys', mockApiKeys))
  const mcp = ref<MCPEndpoint[]>(load('mcp', mockMCP))

  watch(apiKeys, (v) => persist('apiKeys', v), { deep: true })
  watch(mcp, (v) => persist('mcp', v), { deep: true })

  function addKey(k: Omit<ApiKey, 'id' | 'createdAt'>) {
    apiKeys.value.push({ ...k, id: `ak_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function revokeKey(id: string) {
    apiKeys.value = apiKeys.value.filter((k) => k.id !== id)
  }
  function addMcp(m: Omit<MCPEndpoint, 'id' | 'createdAt'>) {
    mcp.value.push({ ...m, id: `mcp_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function toggleMcp(id: string) {
    const i = mcp.value.findIndex((x) => x.id === id)
    if (i >= 0) mcp.value[i] = { ...mcp.value[i], enabled: !mcp.value[i].enabled }
  }
  function removeMcp(id: string) {
    mcp.value = mcp.value.filter((m) => m.id !== id)
  }

  return { apiKeys, mcp, addKey, revokeKey, addMcp, toggleMcp, removeMcp }
})

// ─── Teams ─────────────────────────────────────────────────────────────────

export const useTeamsStore = defineStore('teams', () => {
  const items = ref<Team[]>(load('teams', mockTeams))
  watch(items, (v) => persist('teams', v), { deep: true })

  function get(id: string) {
    return items.value.find((t) => t.id === id)
  }
  function add(t: Omit<Team, 'id' | 'createdAt'>) {
    items.value.push({ ...t, id: `team_${Date.now()}`, createdAt: new Date().toISOString() })
  }
  function remove(id: string) {
    items.value = items.value.filter((t) => t.id !== id)
  }

  return { items, get, add, remove }
})
