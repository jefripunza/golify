// Domain stores. Backend-backed via manual fetch + watchEffect for all 9 menus.
// local/mock data remains as fallback when BE is unreachable (no auth yet).

import { defineStore } from 'pinia'
import { ref, computed, watchEffect } from 'vue'
import type {
  ApiKey,
  Environment,
  Key,
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
  mockProjects,
  mockS3,
  mockServers,
  mockSources,
  mockTeams,
  mockVars,
} from '@/lib/mock'
import { authed, getAuth } from '@/lib/api'

// ─── generic list resource helper ─────────────────────────────────────────
// Returns { items, refresh, pending, error } with mock fallback on any failure.
// Runs fetch() immediately in setup scope and re-runs whenever the auth token
// changes (so login / logout refetches).
function useResourceList<T>(path: string, fallback: T[], map: (raw: any) => T) {
  const items = ref<T[]>([])
  const pending = ref(false)
  const error = ref<unknown>(null)

  async function fetchOnce() {
    pending.value = true
    error.value = null
    try {
      const rows = await authed().get(path).json<any[]>()
      const mapped: T[] = []
      for (const r of rows) mapped.push(map(r))
      items.value = mapped
    } catch (e) {
      error.value = e
      items.value = fallback
    } finally {
      pending.value = false
    }
  }

  // Auto-fetch on mount + re-fetch whenever localStorage auth changes (login/logout).
  watchEffect(() => {
    // touch the auth key so reactivity re-runs on login/logout
    void authed()
    fetchOnce()
  })

  return { items, pending, error, refresh: fetchOnce }
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
function safeParse(x: string) { try { return JSON.parse(x) } catch { return {} } }
function mapTeam(d: any): Team {
  return {
    id: String(d.id), name: d.name, description: d.description ?? '', createdAt: d.created_at,
    members: (d.members ?? []).map((m: any) => ({ id: String(m.id), email: m.email, role: m.role, joinedAt: m.joined_at })),
    permissions: typeof d.permissions === 'string' ? safeParse(d.permissions) : (d.permissions ?? {}),
  }
}

// ─── Projects ──────────────────────────────────────────────────────────────

function mapEnv(e: any): Environment {
  return {
    id: String(e.id),
    name: e.name,
    description: e.description ?? '',
    ipInternal: e.ip_internal ?? '',
    isProduction: e.is_production,
    clusterStatus: e.cluster_status || 'Unknown',
    services: (e.services ?? []).map(mapSvc),
    domains: (e.domains ?? []).map((d: any) => d.host),
  }
}
function mapSvc(s: any): Service {
  return {
    id: String(s.id),
    name: s.name,
    kind: s.kind === 'compose' ? 'compose' : 'container',
    type: s.type === 'database' || s.type === 'tool' ? s.type : 'application',
    catalog: s.catalog || undefined,
    image: s.image || undefined,
    imageTag: s.image_tag || undefined,
    composePath: s.compose_path || undefined,
    description: s.description || undefined,
    dockerOptions: s.docker_options || undefined,
    portsExposes: s.ports_exposes || undefined,
    portMappings: s.port_mappings ?? [],
    networkAliases: s.network_aliases ?? [],
    basicAuthEnable: s.basic_auth_enable ?? false,
    basicAuthUser: s.basic_auth_user || undefined,
    basicAuthPass: s.basic_auth_pass || undefined,
    replicasMode: s.replicas_mode === 'range' ? 'range' : 'fix',
    replicas: s.replicas ?? 1,
    replicasMin: s.replicas_min ?? 1,
    replicasMax: s.replicas_max ?? 1,
    status: s.status ?? 'stopped',
    cpu: s.cpu,
    memory: s.memory,
    ports: s.ports ?? [],
    domains: (s.domains ?? []).map((d: any) => ({ id: String(d.id), host: d.host, port: d.port })),
    networks: (s.networks ?? []).map((n: any) => ({ id: String(n.id), hostPort: n.host_port, containerPort: n.container_port })),
    updatedAt: s.updated_at,
  }
}
function mapProject(p: any): Project {
  return {
    id: String(p.id),
    name: p.name,
    description: p.description,
    sourceId: p.source_id || undefined,
    envCount: p.env_count ?? (p.environments ?? []).length,
    environments: (p.environments ?? []).map(mapEnv),
    createdAt: p.created_at,
  }
}

export const useProjectsStore = defineStore('projects', () => {
  const raw = ref<any[]>([])
  const pending = ref(false)
  const error = ref<unknown>(null)
  async function fetchOnce() {
    pending.value = true
    error.value = null
    try {
      raw.value = await authed().get('api/v1/projects/').json<any[]>()
    } catch (e) { error.value = e; raw.value = [] }
    finally { pending.value = false }
  }
  watchEffect(() => { void authed(); fetchOnce() })

  // Token-reactive refetch: a 5s poll re-reads auth (storage/cookie may
  // have changed since mount) and refetches when a token appears or the
  // raw list is empty while auth exists. SPA navigation never re-mounts
  // the store, so this poll is the safety net that keeps lists alive.
  let lastToken = ''
  setInterval(async () => {
    const auth = getAuth()
    const tok = auth?.token ?? ''
    const emptyButAuthed = raw.value.length === 0 && tok !== ''
    if (tok !== lastToken || emptyButAuthed) {
      lastToken = tok
      await fetchOnce()
    }
  }, 5_000)

  const projects = computed<Project[]>(() => raw.value.map(mapProject))

  async function create(name: string, description: string) {
    const created = await authed().post('api/v1/projects/', { json: { name, description } }).json<any>()
    raw.value = [created, ...raw.value]
    return created
  }

  async function remove(id: string) {
    await authed().delete(`api/v1/projects/${id}`).json<any>()
    raw.value = raw.value.filter((p) => p.id !== id)
  }

  async function removeEnv(projectId: string, envId: string) {
    await authed().delete(`api/v1/projects/${projectId}/environments/${envId}`).json<any>()
    const p = raw.value.find((p) => p.id === projectId)
    if (p) p.environments = p.environments.filter((e: any) => e.id !== envId)
  }

  async function createEnv(projectId: string, input: { name: string; description?: string }) {
    const created = await authed().post(`api/v1/projects/${projectId}/environments`, { json: input }).json<any>()
    const p = raw.value.find((p) => p.id === projectId)
    if (p) p.environments = [created, ...(p.environments ?? [])]
    return created
  }

  async function updateEnv(projectId: string, envId: string, input: { name?: string; description?: string }) {
    const updated = await authed().patch(`api/v1/projects/${projectId}/environments/${envId}`, { json: input }).json<any>()
    const p = raw.value.find((p) => p.id === projectId)
    const e = p?.environments?.find((e: any) => e.id === envId)
    if (e) Object.assign(e, updated)
    return updated
  }

  async function createService(projectId: string, envId: string, input: { name: string; type: string; catalog?: string; image?: string; ports?: string[] }) {
    const created = await authed().post(`api/v1/projects/${projectId}/environments/${envId}/services`, {
      json: { kind: 'container', ...input },
    }).json<any>()
    const p = raw.value.find((p) => p.id === projectId)
    const e = p?.environments?.find((e: any) => e.id === envId)
    if (e) e.services = [...(e.services ?? []), created]
    return created
  }

  async function removeService(projectId: string, envId: string, serviceId: string) {
    // Dummy services (svc-dummy-*) are FE-only placeholders until the
    // backend service CRUD is wired up — removing them is local-only.
    if (serviceId.startsWith('svc-dummy-')) {
      const p = raw.value.find((p) => p.id === projectId)
      const e = p?.environments?.find((e: any) => e.id === envId)
      if (e) e.services = e.services.filter((s: any) => s.id !== serviceId)
      return
    }
    await authed().delete(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}`).json<any>()
    const p = raw.value.find((p) => p.id === projectId)
    const e = p?.environments?.find((e: any) => e.id === envId)
    if (e) e.services = e.services.filter((s: any) => s.id !== serviceId)
  }

  async function updateService(projectId: string, envId: string, serviceId: string, patch: Record<string, unknown>) {
    const updated = await authed().patch(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}`, {
      json: patch,
    }).json<any>()
    const s = getService(projectId, envId, serviceId)
    if (s) Object.assign(s, updated)
    return updated
  }

  async function addServiceDomain(projectId: string, envId: string, serviceId: string, host: string, port: string) {
    const created = await authed().post(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/domains`, {
      json: { host, port },
    }).json<any>()
    // re-fetch so the list reflects exactly what the DB has (fast + consistent)
    await fetchOnce()
    return created
  }

  async function removeServiceDomain(projectId: string, envId: string, serviceId: string, domainId: string) {
    await authed().delete(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/domains/${domainId}`).json<any>()
    await fetchOnce()
  }

  async function updateServiceDomain(projectId: string, envId: string, serviceId: string, domainId: string, patch: { host: string; port: string }) {
    const updated = await authed().patch(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/domains/${domainId}`, {
      json: patch,
    }).json<any>()
    await fetchOnce()
    return updated
  }

  // ─── Service networks (port mappings) ───────────────────────────────────
  async function addServiceNetwork(projectId: string, envId: string, serviceId: string, hostPort: string, containerPort: string) {
    const created = await authed().post(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/networks`, {
      json: { host_port: hostPort, container_port: containerPort },
    }).json<any>()
    await fetchOnce()
    return created
  }

  async function updateServiceNetwork(projectId: string, envId: string, serviceId: string, networkId: string, patch: { host_port: string; container_port: string }) {
    const updated = await authed().patch(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/networks/${networkId}`, {
      json: patch,
    }).json<any>()
    await fetchOnce()
    return updated
  }

  async function removeServiceNetwork(projectId: string, envId: string, serviceId: string, networkId: string) {
    await authed().delete(`api/v1/projects/${projectId}/environments/${envId}/services/${serviceId}/networks/${networkId}`).json<any>()
    await fetchOnce()
  }

  // ─── Root domains (for the subdomain dropdown) ──────────────────────────
  const rootDomains = ref<string[]>([])
  async function fetchRootDomains() {
    try {
      const rows = await authed().get('api/v1/domains').json<any[]>()
      rootDomains.value = (rows ?? []).map((d: any) => String(d.host))
    } catch {
      rootDomains.value = []
    }
  }

  async function update(id: string, name: string, description: string) {
    const updated = await authed().patch(`api/v1/projects/${id}`, { json: { name, description } }).json<any>()
    const i = raw.value.findIndex((p) => p.id === id)
    if (i >= 0) raw.value[i] = updated
    return updated
  }

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

  return { projects, pending, error, get, getEnv, getService, start, stop, create, update, remove, removeEnv, createEnv, updateEnv, createService, removeService, updateService, addServiceDomain, updateServiceDomain, removeServiceDomain, addServiceNetwork, updateServiceNetwork, removeServiceNetwork, rootDomains, fetchRootDomains, refresh: fetchOnce }
})

// ─── Servers ───────────────────────────────────────────────────────────────
export const useServersStore = defineStore('servers', () => {
  const { items: servers, pending, error, refresh } = useResourceList<Server>('api/v1/servers', mockServers, mapServer)
  const onlineCount = computed(() => servers.value.filter((s) => s.status === 'online').length)
  function get(id: string) { return servers.value.find((s) => s.id === id) }
  return { servers, onlineCount, pending, error, get, refresh }
})

// ─── Sources ───────────────────────────────────────────────────────────────
export const useSourcesStore = defineStore('sources', () => {
  const { items: sources, pending, error, refresh } = useResourceList<Source>('api/v1/sources', mockSources, mapSource)
  return { sources, pending, error, refresh }
})

// ─── S3 ─────────────────────────────────────────────────────────────────────
export const useS3Store = defineStore('s3', () => {
  const { items, pending, error, refresh } = useResourceList<S3Storage>('api/v1/s3', mockS3, mapS3)
  return { items, pending, error, refresh }
})

// ─── Variables ──────────────────────────────────────────────────────────────
export const useVarsStore = defineStore('vars', () => {
  const { items, pending, error, refresh } = useResourceList<SharedVariable>('api/v1/variables', mockVars, mapVar)
  function scopeItems(scope: SharedVariable['scope'], scopeRef?: string) {
    return items.value.filter((v) => v.scope === scope && (!scopeRef || v.scopeRef === scopeRef))
  }
  return { items, scopeItems, pending, error, refresh }
})

// ─── Keys ───────────────────────────────────────────────────────────────────
export const useKeysStore = defineStore('keys', () => {
  const { items: keys, pending, error, refresh } = useResourceList<Key>('api/v1/keys', mockKeys, mapKey)
  return { keys, pending, error, refresh }
})

// ─── API Keys ──────────────────────────────────────────────────────────────
export const useApiKeysStore = defineStore('apiKeys', () => {
  const ak = useResourceList<ApiKey>('api/v1/api-keys', mockApiKeys, mapApiKey)

  async function revokeKey(id: string) {
    await authed().delete(`api/v1/api-keys/${id}`).json<any>()
    await ak.refresh()
  }

  async function addKey(input: { name: string; prefix: string; scopes: string[]; lastUsedAt: null; expiresAt: null }) {
    const created = await authed().post('api/v1/api-keys', { json: { name: input.name, scopes: input.scopes } }).json<any>()
    await ak.refresh()
    return created
  }

  return {
    apiKeys: ak.items,
    pending: computed(() => ak.pending.value),
    error: computed(() => ak.error.value ?? null),
    refresh: async () => { await ak.refresh() },
    revokeKey, addKey,
  }
})

// ─── Teams ──────────────────────────────────────────────────────────────────
export const useTeamsStore = defineStore('teams', () => {
  const { items: teams, pending, error, refresh } = useResourceList<Team>('api/v1/teams', mockTeams, mapTeam)
  function get(id: string) { return teams.value.find((t) => t.id === id) }
  return { teams, pending, error, get, refresh }
})