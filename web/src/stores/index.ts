// Domain stores. Backend-backed via Colada for all 9 menus.
// local/mock data remains as fallback when BE is unreachable (no auth yet).

import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useQuery } from '@pinia/colada'
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

// ─── generic list query helper ────────────────────────────────────────────
// Returns { data (computed), status } with mock fallback on any failure.
function useList<T>(path: string, fallback: T[], map: (raw: unknown) => T) {
  const q = useQuery<T[], Error>({
    key: [path],
    query: async () => {
      try {
        const rows = await authed().get(path).json<unknown[]>()
        return rows.map(map)
      } catch {
        return fallback
      }
    },
    staleTime: 30_000,
  })
  const data = computed(() => q.data.value ?? fallback)
  return { ...q, data }
}

// ─── BE DTO → FE type mappers (snake_case → camelCase) ──────────────────

function mapServer(r: any): Server {
  return {
    id: String(r.id),
    name: r.name,
    host: r.host,
    ip: r.ip ?? '',
    region: r.region ?? '',
    provider: r.provider ?? 'self-hosted',
    status: r.status ?? 'unknown',
    cpu: r.cpu ?? 0,
    memory: r.memory ?? 0,
    memoryTotal: r.memory_total ?? 0,
    disk: r.disk ?? 0,
    containers: r.containers ?? 0,
    keyId: r.key_id ? String(r.key_id) : undefined,
  }
}
function mapSource(d: any): Source {
  return { id: String(d.id), name: d.name, provider: d.provider, url: d.url, isGlobal: d.is_global ?? false, createdAt: d.created_at }
}
function mapS3(d: any): S3Storage {
  return { id: String(d.id), name: d.name, endpoint: d.endpoint, region: d.region ?? '', bucket: d.bucket, accessKeyId: d.access_key_id, isDefault: d.is_default ?? false, createdAt: d.created_at }
}
function mapVar(d: any): SharedVariable {
  return { id: String(d.id), key: d.key, value: d.value ?? '', isSecret: d.is_secret ?? false, scope: d.scope ?? 'global', scopeRef: d.scope_ref ? String(d.scope_ref) : undefined, updatedAt: d.updated_at }
}
function mapKey(d: any): Key {
  return { id: String(d.id), name: d.name, publicKey: d.public_key, fingerprint: d.fingerprint ?? '', createdAt: d.created_at }
}
function mapApiKey(d: any): ApiKey {
  return { id: String(d.id), name: d.name, prefix: d.prefix, scopes: d.scopes ?? [], createdAt: d.created_at, lastUsedAt: d.last_used_at ?? null, expiresAt: d.expires_at ?? null }
}
function mapMcp(d: any): MCPEndpoint {
  return { id: String(d.id), name: d.name, url: d.url, transport: d.transport ?? 'http', apiKeyId: String(d.api_key_id ?? ''), enabled: d.enabled ?? true, createdAt: d.created_at }
}
function mapTeam(d: any): Team {
  return {
    id: String(d.id), name: d.name, description: d.description ?? '', createdAt: d.created_at,
    members: (d.members ?? []).map((m: any) => ({ id: String(m.id), email: m.email, role: m.role, joinedAt: m.joined_at })),
    permissions: typeof d.permissions === 'string' ? safeParse(d.permissions) : (d.permissions ?? {}),
  }
}
function safeParse(x: string) { try { return JSON.parse(x) } catch { return {} } }

// ─── Projects ──────────────────────────────────────────────────────────────

function mapEnv(e: any): Environment {
  return {
    id: String(e.id),
    name: e.name,
    isProduction: e.is_production,
    services: (e.services ?? []).map(mapSvc),
    domains: (e.domains ?? []).map((d: any) => d.host),
  }
}
function mapSvc(s: any): Service {
  return {
    id: String(s.id),
    name: s.name,
    kind: s.kind === 'compose' ? 'compose' : 'container',
    image: s.image || undefined,
    composePath: s.compose_path || undefined,
    status: s.status ?? 'stopped',
    cpu: s.cpu,
    memory: s.memory,
    ports: s.ports ?? [],
    updatedAt: s.updated_at,
  }
}
function mapProject(p: any): Project {
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
  const q = useQuery<any[], Error>({
    key: ['projects'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/projects/').json<any[]>()
        return rows
      } catch {
        return []
      }
    },
    staleTime: 30_000,
  })
  const projects = computed<Project[]>(() => (q.data.value ?? []).map(mapProject))

  function get(id: string): Project | undefined { return projects.value.find((p) => p.id === id) }
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

  return { projects, get, getEnv, getService, start, stop, ...q }
})

// ─── Servers ────────────────────────────────────────────────────────────────
export const useServersStore = defineStore('servers', () => {
  const q = useQuery<Server[], any>({
    key: ['servers'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/servers').json<any[]>()
        return rows.map(mapServer)
      } catch { return mockServers }
    },
    staleTime: 30_000,
  })
  const servers = computed(() => q.data.value ?? [])
  const onlineCount = computed(() => servers.value.filter((s) => s.status === 'online').length)
  function get(id: string) { return servers.value.find((s) => s.id === id) }
  return { servers, get, onlineCount, ...q }
})

// ─── Sources ────────────────────────────────────────────────────────────────
export const useSourcesStore = defineStore('sources', () => {
  const q = useQuery<Source[], any>({
    key: ['sources'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/sources').json<any[]>()
        return rows.map(mapSource)
      } catch { return mockSources }
    },
    staleTime: 30_000,
  })
  const sources = computed(() => q.data.value ?? [])
  return { sources, ...q }
})

// ─── S3 ─────────────────────────────────────────────────────────────────────
export const useS3Store = defineStore('s3', () => {
  const q = useQuery<S3Storage[], any>({
    key: ['s3'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/s3').json<any[]>()
        return rows.map(mapS3)
      } catch { return mockS3 }
    },
    staleTime: 30_000,
  })
  const items = computed(() => q.data.value ?? [])
  return { items, ...q }
})

// ─── Variables ──────────────────────────────────────────────────────────────
export const useVarsStore = defineStore('vars', () => {
  const q = useQuery<SharedVariable[], any>({
    key: ['variables'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/variables').json<any[]>()
        return rows.map(mapVar)
      } catch { return mockVars }
    },
    staleTime: 30_000,
  })
  const items = computed(() => q.data.value ?? [])
  function scopeItems(scope: SharedVariable['scope'], scopeRef?: string) {
    return items.value.filter((v) => v.scope === scope && (!scopeRef || v.scopeRef === scopeRef))
  }
  return { items, scopeItems, ...q }
})

// ─── Keys ───────────────────────────────────────────────────────────────────
export const useKeysStore = defineStore('keys', () => {
  const q = useQuery<Key[], any>({
    key: ['keys'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/keys').json<any[]>()
        return rows.map(mapKey)
      } catch { return mockKeys }
    },
    staleTime: 30_000,
  })
  const keys = computed(() => q.data.value ?? [])
  return { keys, ...q }
})

// ─── API Keys & MCP ─────────────────────────────────────────────────────────
export const useApiMcpStore = defineStore('apiMcp', () => {
  const ak = useQuery<ApiKey[], any>({
    key: ['api-keys'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/api-keys').json<any[]>()
        return rows.map(mapApiKey)
      } catch { return mockApiKeys }
    },
    staleTime: 30_000,
  })
  const mcpQ = useQuery<MCPEndpoint[], any>({
    key: ['mcp'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/mcp').json<any[]>()
        return rows.map(mapMcp)
      } catch { return mockMCP }
    },
    staleTime: 30_000,
  })
  const apiKeys = computed(() => ak.data.value ?? [])
  const mcp = computed(() => mcpQ.data.value ?? [])
  return { apiKeys, mcp, ...ak, ...mcpQ }
})

// ─── Teams ──────────────────────────────────────────────────────────────────
export const useTeamsStore = defineStore('teams', () => {
  const q = useQuery<Team[], any>({
    key: ['teams'],
    query: async () => {
      try {
        const rows = await authed().get('api/v1/teams').json<any[]>()
        return rows.map(mapTeam)
      } catch { return mockTeams }
    },
    staleTime: 30_000,
  })
  const teams = computed(() => q.data.value ?? [])
  function get(id: string) { return teams.value.find((t) => t.id === id) }
  return { teams, get, ...q }
})