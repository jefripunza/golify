package main

import "time"

// Message represents a notification message stored in the golify database.
type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:255;not null;default:''" json:"title"`
	Message   string    `gorm:"not null" json:"message"`
	Priority  int       `gorm:"not null;default:0" json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// Application represents an application that can send messages via a token.
type Application struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null;uniqueIndex" json:"name"`
	Token     string    `gorm:"size:255;not null;uniqueIndex" json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

// Client represents a connected device/web client receiving messages.
type Client struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"size:255;not null" json:"name"`
	Token     string     `gorm:"size:255;not null;uniqueIndex" json:"token"`
	LastSeen  *time.Time `json:"last_seen"`
	CreatedAt time.Time  `json:"created_at"`
}

// User is a dashboard user (admin or read-only).
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:255;not null;uniqueIndex" json:"username"`
	Passhash  string    `gorm:"size:255;not null" json:"-"`
	Admin     bool      `gorm:"not null;default:false" json:"admin"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── PaaS-style dashboard models ──────────────────────────────────────────

// Project groups environments (e.g. a software project).
type Project struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"size:1024;default:''" json:"description"`
	SourceID    string         `gorm:"size:255;default:''" json:"source_id"`
	Envs        []Environment  `gorm:"constraint:OnDelete:CASCADE" json:"environments,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Environment groups services of a Project (production, staging, ...).
type Environment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ProjectID    uint      `gorm:"not null;index" json:"project_id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	IsProduction bool      `gorm:"not null;default:false" json:"is_production"`
	Domains      []Domain  `gorm:"constraint:OnDelete:CASCADE" json:"domains,omitempty"`
	Services     []Service `gorm:"constraint:OnDelete:CASCADE" json:"services,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Service is a container or compose app inside an Environment.
type Service struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	EnvironmentID uint      `gorm:"not null;index" json:"environment_id"`
	Name          string    `gorm:"size:255;not null" json:"name"`
	Kind          string    `gorm:"size:32;not null;default:'container'" json:"kind"` // container | compose
	Image         string    `gorm:"size:512;default:''" json:"image"`
	ComposePath   string    `gorm:"size:512;default:''" json:"compose_path"`
	Status        string    `gorm:"size:32;not null;default:'stopped'" json:"status"`
	CPU           float64   `gorm:"default:0" json:"cpu"`
	Memory        int64     `gorm:"default:0" json:"memory"` // MB
	Ports         []string  `gorm:"serializer:json" json:"ports"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Domain is one hostname attached to an Environment.
type Domain struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	EnvironmentID uint      `gorm:"not null;index" json:"environment_id"`
	Host          string    `gorm:"size:255;not null" json:"host"`
	CreatedAt     time.Time `json:"created_at"`
}
