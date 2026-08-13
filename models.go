package main

import (
	"time"

	"gorm.io/gorm"
)

// UUID is a string primary key holding a UUID v7 (time-ordered).
type UUID = string

// BeforeCreate auto-fills a UUID v7 primary key when empty. Every model with
// `ID UUID` benefits; callers never need to set IDs manually.
func (m *User) BeforeCreate(tx *gorm.DB) error           { return fillID(&m.ID) }
func (m *Project) BeforeCreate(tx *gorm.DB) error        { return fillID(&m.ID) }
func (m *Environment) BeforeCreate(tx *gorm.DB) error    { return fillID(&m.ID) }
func (m *Service) BeforeCreate(tx *gorm.DB) error        { return fillID(&m.ID) }
func (m *ServiceDomain) BeforeCreate(tx *gorm.DB) error  { return fillID(&m.ID) }
func (m *Domain) BeforeCreate(tx *gorm.DB) error         { return fillID(&m.ID) }
func (m *Server) BeforeCreate(tx *gorm.DB) error         { return fillID(&m.ID) }
func (m *Source) BeforeCreate(tx *gorm.DB) error         { return fillID(&m.ID) }
func (m *S3Storage) BeforeCreate(tx *gorm.DB) error      { return fillID(&m.ID) }
func (m *SharedVariable) BeforeCreate(tx *gorm.DB) error { return fillID(&m.ID) }
func (m *Key) BeforeCreate(tx *gorm.DB) error            { return fillID(&m.ID) }
func (m *ApiKey) BeforeCreate(tx *gorm.DB) error         { return fillID(&m.ID) }
func (m *Team) BeforeCreate(tx *gorm.DB) error           { return fillID(&m.ID) }
func (m *TeamMember) BeforeCreate(tx *gorm.DB) error     { return fillID(&m.ID) }

// fillID assigns a UUID v7 when the pointer target is empty.
func fillID(id *UUID) error {
	if *id == "" {
		*id = newID()
	}
	return nil
}

// User is a dashboard user (admin or read-only).
type User struct {
	ID        UUID      `gorm:"primaryKey;size:36" json:"id"`
	Username  string    `gorm:"size:255;not null;uniqueIndex" json:"username"`
	Email     string    `gorm:"size:255;not null;default:'';uniqueIndex" json:"email"`
	Passhash  string    `gorm:"size:255;not null" json:"-"`
	Admin     bool      `gorm:"not null;default:false" json:"admin"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── PaaS-style dashboard models ──────────────────────────────────────────

// Project groups environments (e.g. a software project).
type Project struct {
	ID          UUID          `gorm:"primaryKey;size:36" json:"id"`
	Name        string        `gorm:"size:255;not null" json:"name"`
	Description string        `gorm:"size:1024;default:''" json:"description"`
	SourceID    string        `gorm:"size:255;default:''" json:"source_id"`
	Envs        []Environment `gorm:"constraint:OnDelete:CASCADE" json:"environments,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Environment groups services of a Project (production, staging, ...).
type Environment struct {
	ID           UUID      `gorm:"primaryKey;size:36" json:"id"`
	ProjectID    UUID      `gorm:"not null;index;size:36" json:"project_id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	Description  string    `gorm:"size:512;default:''" json:"description"`
	IsProduction bool      `gorm:"not null;default:false" json:"is_production"`
	IPInternal   string    `gorm:"size:64;default:''" json:"ip_internal"`
	Domains      []Domain  `gorm:"constraint:OnDelete:CASCADE" json:"domains,omitempty"`
	Services     []Service `gorm:"constraint:OnDelete:CASCADE" json:"services,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Service is a container or compose app inside an Environment.
type Service struct {
	ID            UUID   `gorm:"primaryKey;size:36" json:"id"`
	EnvironmentID UUID   `gorm:"not null;index;size:36" json:"environment_id"`
	Name          string `gorm:"size:255;not null" json:"name"`
	Kind          string `gorm:"size:32;not null;default:'container'" json:"kind"`   // container | compose
	Type          string `gorm:"size:32;not null;default:'application'" json:"type"` // application | database | tool
	Catalog       string `gorm:"size:64;default:''" json:"catalog"`                  // e.g. docker-image | version-control | postgres | qdrant
	Image         string `gorm:"size:512;default:''" json:"image"`
	ImageTag      string `gorm:"size:128;default:'latest'" json:"image_tag"` // docker image tag or hash
	ComposePath   string `gorm:"size:512;default:''" json:"compose_path"`
	Description   string `gorm:"size:512;default:''" json:"description"`
	// Coolify-style configuration
	DockerOptions   string          `gorm:"size:512;default:''" json:"docker_options"` // custom docker options e.g. --privileged
	PortsExposes    string          `gorm:"size:255;default:''" json:"ports_exposes"`  // e.g. 8080
	PortMappings    []string        `gorm:"serializer:json" json:"port_mappings"`      // e.g. ["3000:3000"]
	NetworkAliases  []string        `gorm:"serializer:json" json:"network_aliases"`
	BasicAuthEnable bool            `gorm:"default:false" json:"basic_auth_enable"`
	BasicAuthUser   string          `gorm:"size:255;default:''" json:"basic_auth_user"`
	BasicAuthPass   string          `gorm:"size:255;default:''" json:"basic_auth_pass"`
	Status          string          `gorm:"size:32;not null;default:'stopped'" json:"status"`
	CPU             float64         `gorm:"default:0" json:"cpu"`
	Memory          int64           `gorm:"default:0" json:"memory"` // MB
	Ports           []string        `gorm:"serializer:json" json:"ports"`
	Domains         []ServiceDomain `gorm:"constraint:OnDelete:CASCADE" json:"domains,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ServiceDomain is a domain/subdomain attached to a single Service,
// mapped to a specific port of that service (Coolify-style).
type ServiceDomain struct {
	ID        UUID      `gorm:"primaryKey;size:36" json:"id"`
	ServiceID UUID      `gorm:"not null;index;size:36" json:"service_id"`
	Host      string    `gorm:"size:255;not null" json:"host"`    // e.g. app.example.com
	Port      string    `gorm:"size:16;default:'80'" json:"port"` // target port on the service
	CreatedAt time.Time `json:"created_at"`
}

// Domain is a hostname attached to an Environment. It is also the single
// place where domains are registered (menu "Domains") — the standalone
// domain_entries table was removed. A domain may be registered without an
// environment (environment_id empty) and later linked to one; the proxy
// gate only serves the SPA when the environment has a service.
type Domain struct {
	ID            UUID      `gorm:"primaryKey;size:36" json:"id"`
	EnvironmentID *UUID     `gorm:"index;size:36" json:"environment_id"`
	Host          string    `gorm:"size:255;not null;uniqueIndex" json:"host"`
	CreatedAt     time.Time `json:"created_at"`
}

// ─── Infrastructure / Security models (menus Servers..Teams) ──────────────

// Server is a deploy target registered in the dashboard.
type Server struct {
	ID          UUID      `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Host        string    `gorm:"size:255;not null" json:"host"`
	IP          string    `gorm:"size:64;default:''" json:"ip"`
	Region      string    `gorm:"size:64;default:''" json:"region"`
	Provider    string    `gorm:"size:32;default:'self-hosted'" json:"provider"`
	Status      string    `gorm:"size:16;default:'unknown'" json:"status"`
	CPU         float64   `gorm:"default:0" json:"cpu"`
	Memory      int64     `gorm:"default:0" json:"memory"`       // MB used
	MemoryTotal int64     `gorm:"default:0" json:"memory_total"` // MB total
	Disk        float64   `gorm:"default:0" json:"disk"`         // % used
	Containers  int       `gorm:"default:0" json:"containers"`
	KeyID       UUID      `gorm:"default:'';size:36" json:"key_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Source is a VCS repo / git provider connection.
type Source struct {
	ID        UUID      `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Provider  string    `gorm:"size:32;not null" json:"provider"`
	URL       string    `gorm:"size:512;not null" json:"url"`
	IsGlobal  bool      `gorm:"not null;default:false" json:"is_global"`
	CreatedAt time.Time `json:"created_at"`
}

// S3Storage is an object-storage backup target.
type S3Storage struct {
	ID          UUID      `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Endpoint    string    `gorm:"size:512;not null" json:"endpoint"`
	Region      string    `gorm:"size:64;default:''" json:"region"`
	Bucket      string    `gorm:"size:255;not null" json:"bucket"`
	AccessKeyID string    `gorm:"size:255;not null" json:"access_key_id"`
	IsDefault   bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}

// SharedVariable is a global/project/env/service scoped key-value.
type SharedVariable struct {
	ID        UUID      `gorm:"primaryKey;size:36" json:"id"`
	Key       string    `gorm:"size:255;not null" json:"key"`
	Value     string    `gorm:"size:4096;default:''" json:"value"`
	IsSecret  bool      `gorm:"not null;default:false" json:"is_secret"`
	Scope     string    `gorm:"size:32;not null;default:'global'" json:"scope"`
	ScopeRef  UUID      `gorm:"default:'';size:36" json:"scope_ref"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Key is an SSH public key (private key never stored in DB).
type Key struct {
	ID          UUID      `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	PublicKey   string    `gorm:"size:4096;not null" json:"public_key"`
	Fingerprint string    `gorm:"size:255;default:''" json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

// ApiKey is a token for CI / external automation.
type ApiKey struct {
	ID         UUID       `gorm:"primaryKey;size:36" json:"id"`
	Name       string     `gorm:"size:255;not null" json:"name"`
	Prefix     string     `gorm:"size:64;not null" json:"prefix"`
	Scopes     []string   `gorm:"serializer:json" json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Team + members (RBAC).
type Team struct {
	ID          UUID         `gorm:"primaryKey;size:36" json:"id"`
	Name        string       `gorm:"size:255;not null" json:"name"`
	Description string       `gorm:"size:1024;default:''" json:"description"`
	Permissions string       `gorm:"type:text;default:'{}'" json:"permissions"` // JSON map
	Members     []TeamMember `gorm:"constraint:OnDelete:CASCADE" json:"members,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type TeamMember struct {
	ID       UUID      `gorm:"primaryKey;size:36" json:"id"`
	TeamID   UUID      `gorm:"not null;index;size:36" json:"team_id"`
	Email    string    `gorm:"size:255;not null" json:"email"`
	Role     string    `gorm:"size:32;not null;default:'viewer'" json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
