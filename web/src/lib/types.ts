// Domain types for the 9-menu dashboard.
// BE is still a notif server; these types are mock-only and will swap to API DTOs later.

export type ID = string

export type ServiceKind = 'container' | 'compose'
export type ServiceStatus = 'running' | 'stopped' | 'building' | 'error' | 'deploying'

export interface Service {
  id: ID
  name: string
  kind: ServiceKind
  image?: string
  composePath?: string
  status: ServiceStatus
  cpu: number // %
  memory: number // MB
  ports: string[]
  updatedAt: string // ISO
}

export interface Environment {
  id: ID
  name: string // e.g. production, staging, preview-123
  description?: string
  ipInternal?: string
  services: Service[]
  domains: string[]
  isProduction: boolean
  clusterStatus?: string
}

export interface Project {
  id: ID
  name: string
  description: string
  sourceId?: ID
  envCount?: number
  environments: Environment[]
  createdAt: string
}

export interface Server {
  id: ID
  name: string
  host: string
  ip: string
  region: string
  provider: 'self-hosted' | 'aws' | 'gcp' | 'azure' | 'digitalocean' | 'hetzner' | 'other'
  status: 'online' | 'offline' | 'unknown'
  cpu: number
  memory: number // MB used / total
  memoryTotal: number
  disk: number // % used
  containers: number
  keyId?: ID
}

export type SourceProvider =
  | 'github'
  | 'gitlab'
  | 'bitbucket'
  | 'gitea'
  | 'codeberg'
  | 'other'

export interface Source {
  id: ID
  name: string
  provider: SourceProvider
  url: string
  isGlobal: boolean
  createdAt: string
}

export interface S3Storage {
  id: ID
  name: string
  endpoint: string
  region: string
  bucket: string
  accessKeyId: string // we never store the secret in mock; just the AKID prefix
  isDefault: boolean
  createdAt: string
}

export interface SharedVariable {
  id: ID
  key: string
  value: string
  isSecret: boolean
  scope: 'global' | 'project' | 'environment' | 'service'
  scopeRef?: ID
  updatedAt: string
}

export interface Key {
  id: ID
  name: string
  publicKey: string // ssh-ed25519 ...
  fingerprint: string
  createdAt: string
  // privateKey intentionally NOT stored in mock; in real BE it would be encrypted at rest
}

export interface ApiKey {
  id: ID
  name: string
  prefix: string // first chars of token, like ghk_xxxx
  scopes: string[]
  createdAt: string
  lastUsedAt: string | null
  expiresAt: string | null
}

export type ResourceScope = 'projects' | 'servers' | 'sources' | 's3' | 'variables' | 'keys' | 'api-keys' | 'teams'

export interface TeamMember {
  id: ID
  email: string
  role: 'owner' | 'admin' | 'developer' | 'viewer'
  joinedAt: string
}

export interface Team {
  id: ID
  name: string
  description: string
  members: TeamMember[]
  // permissions per scope: '*' = full, 'read' = view-only, or array of resource IDs
  permissions: Partial<Record<ResourceScope, '*' | 'read' | ID[]>>
  createdAt: string
}
