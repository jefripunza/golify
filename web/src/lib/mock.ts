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

const now = () => new Date().toISOString()

// ─── Services ──────────────────────────────────────────────────────────────

const svc = (
  id: string,
  name: string,
  kind: 'container' | 'compose',
  status: Service['status'],
  extras: Partial<Service> = {},
): Service => ({
  id,
  name,
  kind,
  status,
  cpu: Math.round(Math.random() * 80),
  memory: 100 + Math.round(Math.random() * 800),
  ports: [],
  updatedAt: now(),
  ...extras,
})

const envProd = (id: string, name: string, services: Service[], domains: string[]): Environment => ({
  id,
  name,
  services,
  domains,
  isProduction: true,
})

const envStage = (id: string, name: string, services: Service[], domains: string[] = [`stage-${name}.example.com`]): Environment => ({
  id,
  name,
  services,
  domains,
  isProduction: false,
})

// ─── Projects ──────────────────────────────────────────────────────────────

export const mockProjects: Project[] = [
  {
    id: 'proj_website',
    name: 'sawang.tech-website',
    description: 'NuxtJS StarterKit landing page deployed via Coolify.',
    sourceId: 'src_github_main',
    createdAt: now(),
    environments: [
      envProd(
        'env_prod_website',
        'production',
        [
          svc('svc_web', 'web', 'compose', 'running', {
            image: 'node:22-alpine',
            ports: ['3000:3000'],
          }),
          svc('svc_db', 'postgres', 'container', 'running', {
            image: 'postgres:16-alpine',
            ports: ['5432:5432'],
          }),
          svc('svc_cache', 'redis', 'container', 'stopped', {
            image: 'redis:7-alpine',
          }),
        ],
        ['sawang.tech', 'www.sawang.tech'],
      ),
      envStage('env_stage_website', 'staging', [
        svc('svc_web_stage', 'web', 'compose', 'running', {
          image: 'node:22-alpine',
          ports: ['3001:3000'],
        }),
      ]),
    ],
  },
  {
    id: 'proj_gotify',
    name: 'gotify',
    description: 'Self-hosted push notification server (this project).',
    sourceId: 'src_github_main',
    createdAt: now(),
    environments: [
      envProd(
        'env_prod_gotify',
        'production',
        [
          svc('svc_gotify_run', 'gotify', 'compose', 'running', {
            image: 'jefripunza/gotify:latest',
            ports: ['80:80', '443:443', '3000:3000'],
          }),
        ],
        ['gotify.sawang.tech'],
      ),
    ],
  },
  {
    id: 'proj_hindsight',
    name: 'hindsight-agent-memory',
    description: 'Self-hosted agent memory system (db + tei + app).',
    sourceId: 'src_gitea_main',
    createdAt: now(),
    environments: [
      envProd('env_prod_hindsight', 'production', [
        svc('svc_hindsight_db', 'hindsight-db', 'container', 'running', { image: 'postgres:16-alpine' }),
        svc('svc_hindsight_tei', 'hindsight-tei', 'container', 'running', { image: 'ghcr.io/hindsight/tei:latest' }),
        svc('svc_hindsight_app', 'hindsight-app', 'container', 'building', { image: 'ghcr.io/hindsight/app:latest' }),
      ]),
    ],
  },
]

// ─── Servers ───────────────────────────────────────────────────────────────

export const mockServers: Server[] = [
  {
    id: 'srv_local',
    name: 'laptop-jefri',
    host: 'laptop.local',
    ip: '192.168.69.2',
    region: 'home',
    provider: 'self-hosted',
    status: 'online',
    cpu: 35,
    memory: 6200,
    memoryTotal: 16000,
    disk: 48,
    containers: 11,
    keyId: 'key_main',
  },
  {
    id: 'srv_vps_jak1',
    name: 'vps-jakarta-1',
    host: 'vps1.sawang.cloud',
    ip: '103.21.x.x',
    region: 'ap-southeast-3',
    provider: 'hetzner',
    status: 'online',
    cpu: 18,
    memory: 2400,
    memoryTotal: 8000,
    disk: 22,
    containers: 6,
    keyId: 'key_main',
  },
  {
    id: 'srv_offline',
    name: 'legacy-build',
    host: 'build.legacy.local',
    ip: '10.0.0.5',
    region: 'home',
    provider: 'self-hosted',
    status: 'offline',
    cpu: 0,
    memory: 0,
    memoryTotal: 8000,
    disk: 0,
    containers: 0,
  },
]

// ─── Sources ───────────────────────────────────────────────────────────────

export const mockSources: Source[] = [
  {
    id: 'src_github_main',
    name: 'jefripunza',
    provider: 'github',
    url: 'https://github.com/jefripunza',
    isGlobal: true,
    createdAt: now(),
  },
  {
    id: 'src_gitlab_work',
    name: 'gitlab-work',
    provider: 'gitlab',
    url: 'https://gitlab.com/jefripunza',
    isGlobal: false,
    createdAt: now(),
  },
  {
    id: 'src_gitea_main',
    name: 'gitea-self-hosted',
    provider: 'gitea',
    url: 'https://git.sawang.tech',
    isGlobal: true,
    createdAt: now(),
  },
]

// ─── S3 storages ───────────────────────────────────────────────────────────

export const mockS3: S3Storage[] = [
  {
    id: 's3_main',
    name: 'sawang-backups',
    endpoint: 'https://s3.jkt1.sawang.cloud',
    region: 'jkt1',
    bucket: 'gotify-backups',
    accessKeyId: 'AKIAJEXAMPLExxxx',
    isDefault: true,
    createdAt: now(),
  },
  {
    id: 's3_cold',
    name: 'cold-archive',
    endpoint: 'https://s3.eu-central-1.wasabisys.com',
    region: 'eu-central-1',
    bucket: 'cold-archive',
    accessKeyId: 'AKIAWASABIxxxxx',
    isDefault: false,
    createdAt: now(),
  },
]

// ─── Shared variables ──────────────────────────────────────────────────────

export const mockVars: SharedVariable[] = [
  { id: 'v1', key: 'NODE_ENV', value: 'production', isSecret: false, scope: 'global', updatedAt: now() },
  { id: 'v2', key: 'LOG_LEVEL', value: 'info', isSecret: false, scope: 'global', updatedAt: now() },
  { id: 'v3', key: 'DB_PASSWORD', value: '••••••••••', isSecret: true, scope: 'global', updatedAt: now() },
  { id: 'v4', key: 'DOMAIN', value: 'sawang.tech', isSecret: false, scope: 'project', scopeRef: 'proj_website', updatedAt: now() },
  { id: 'v5', key: 'JWT_SIGNING_KEY', value: '••••••••••', isSecret: true, scope: 'global', updatedAt: now() },
]

// ─── Keys ──────────────────────────────────────────────────────────────────

export const mockKeys: Key[] = [
  {
    id: 'key_main',
    name: 'deploy-key-main',
    publicKey: 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJx7...bWhpQ== jefri@laptop',
    fingerprint: 'SHA256:abcd1234efgh5678ijkl9012mnop3456qrst7890uvwx',
    createdAt: now(),
  },
  {
    id: 'key_legacy',
    name: 'legacy-build-server',
    publicKey: 'ssh-rsa AAAAB3NzaC1yc2EAAAA...',
    fingerprint: 'SHA256:0000ffff1111aaaa2222bbbb3333cccc4444dddd',
    createdAt: now(),
  },
]

// ─── API Keys & MCP ────────────────────────────────────────────────────────

export const mockApiKeys: ApiKey[] = [
  {
    id: 'ak_main',
    name: 'cli-default',
    prefix: 'ghk_4f7c1d2e',
    scopes: ['projects:read', 'projects:write', 'servers:read'],
    createdAt: now(),
    lastUsedAt: now(),
    expiresAt: null,
  },
  {
    id: 'ak_ci',
    name: 'ci-deploy',
    prefix: 'ghk_a1b2c3d4',
    scopes: ['projects:write'],
    createdAt: now(),
    lastUsedAt: now(),
    expiresAt: new Date(Date.now() + 30 * 86400 * 1000).toISOString(),
  },
]

export const mockMCP: MCPEndpoint[] = [
  {
    id: 'mcp_main',
    name: 'main-mcp',
    url: 'http://localhost:3000/mcp',
    transport: 'http',
    apiKeyId: 'ak_main',
    enabled: true,
    createdAt: now(),
  },
  {
    id: 'mcp_agent',
    name: 'agent-mcp',
    url: 'http://localhost:3001/mcp',
    transport: 'sse',
    apiKeyId: 'ak_main',
    enabled: false,
    createdAt: now(),
  },
]

// ─── Teams ─────────────────────────────────────────────────────────────────

export const mockTeams: Team[] = [
  {
    id: 'team_dev',
    name: 'developers',
    description: 'Core dev team — full access.',
    members: [
      { id: 'm1', email: 'jefri@sawang.tech', role: 'owner', joinedAt: now() },
      { id: 'm2', email: 'andi@sawang.tech', role: 'admin', joinedAt: now() },
      { id: 'm3', email: 'dev@sawang.tech', role: 'developer', joinedAt: now() },
    ],
    permissions: {
      projects: '*',
      servers: '*',
      sources: '*',
      s3: '*',
      variables: '*',
      keys: 'read',
      'api-mcp': '*',
      teams: '*',
    },
    createdAt: now(),
  },
  {
    id: 'team_viewer',
    name: 'viewers',
    description: 'Read-only access for stakeholders.',
    members: [{ id: 'm4', email: 'client@example.com', role: 'viewer', joinedAt: now() }],
    permissions: {
      projects: 'read',
      servers: 'read',
      sources: 'read',
      s3: 'read',
      variables: 'read',
      keys: 'read',
      'api-mcp': 'read',
      teams: 'read',
    },
    createdAt: now(),
  },
]
