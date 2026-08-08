package main

import "time"

// Message represents a notification message stored in the gotify database.
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
