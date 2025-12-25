// Package models contains shared data structures for the LinkedIn automation tool.
package models

import (
	"time"
)

// ProfileStatus represents the current state of a LinkedIn profile in the system
type ProfileStatus string

const (
	ProfileStatusFound     ProfileStatus = "found"     // Profile discovered via search
	ProfileStatusRequested ProfileStatus = "requested" // Connection request sent
	ProfileStatusConnected ProfileStatus = "connected" // Connection accepted
	ProfileStatusSkipped   ProfileStatus = "skipped"   // Skipped (error or manual skip)
)

// ConnectionStatus represents the state of a connection request
type ConnectionStatus string

const (
	ConnectionStatusPending  ConnectionStatus = "pending"  // Request sent, awaiting response
	ConnectionStatusAccepted ConnectionStatus = "accepted" // Request accepted
	ConnectionStatusIgnored  ConnectionStatus = "ignored"  // Request ignored/declined
)

// MessageType represents the type of message sent
type MessageType string

const (
	MessageTypeConnectionNote MessageType = "connection_note" // Note sent with connection request
	MessageTypeFollowup       MessageType = "followup"        // Follow-up message after connection
	MessageTypeDirect         MessageType = "direct"          // Direct message
)

// Profile represents a LinkedIn profile
type Profile struct {
	ID        int64         `json:"id"`
	URL       string        `json:"url"`
	FirstName string        `json:"first_name"`
	LastName  string        `json:"last_name"`
	FullName  string        `json:"full_name"`
	Title     string        `json:"title"`
	Company   string        `json:"company"`
	Location  string        `json:"location"`
	Status    ProfileStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ID        int64            `json:"id"`
	ProfileID int64            `json:"profile_id"`
	NoteText  string           `json:"note_text"`
	Status    ConnectionStatus `json:"status"`
	SentAt    time.Time        `json:"sent_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// Message represents a message sent to a connection
type Message struct {
	ID          int64       `json:"id"`
	ProfileID   int64       `json:"profile_id"`
	MessageText string      `json:"message_text"`
	MessageType MessageType `json:"message_type"`
	SentAt      time.Time   `json:"sent_at"`
}

// DailyStats tracks daily activity for rate limiting
type DailyStats struct {
	ID                int64     `json:"id"`
	Date              string    `json:"date"` // YYYY-MM-DD format
	ConnectionsSent   int       `json:"connections_sent"`
	MessagesSent      int       `json:"messages_sent"`
	ProfilesSearched  int       `json:"profiles_searched"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Session stores browser session data
type Session struct {
	ID        int64     `json:"id"`
	Cookies   string    `json:"cookies"` // JSON-encoded cookies
	UserAgent string    `json:"user_agent"`
	IsValid   bool      `json:"is_valid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CheckpointType represents types of security checkpoints
type CheckpointType string

const (
	CheckpointNone            CheckpointType = "none"
	CheckpointTwoFactor       CheckpointType = "two_factor"
	CheckpointCaptcha         CheckpointType = "captcha"
	CheckpointPhoneVerify     CheckpointType = "phone_verify"
	CheckpointEmailVerify     CheckpointType = "email_verify"
	CheckpointUnusualActivity CheckpointType = "unusual_activity"
	CheckpointUnknown         CheckpointType = "unknown"
)

// LoginResult represents the outcome of a login attempt
type LoginResult struct {
	Success        bool           `json:"success"`
	CheckpointType CheckpointType `json:"checkpoint_type,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	SessionSaved   bool           `json:"session_saved"`
}

// SearchParams holds parameters for LinkedIn search
type SearchParams struct {
	JobTitle string   `json:"job_title"`
	Company  string   `json:"company"`
	Location string   `json:"location"`
	Keywords []string `json:"keywords"`
}

// TemplateData holds data for message template rendering
type TemplateData struct {
	FirstName string
	LastName  string
	FullName  string
	Company   string
	Title     string
	Location  string
}

// NewTemplateData creates TemplateData from a Profile
func NewTemplateData(p *Profile) TemplateData {
	firstName := p.FirstName
	if firstName == "" {
		firstName = "there" // Fallback
	}
	
	return TemplateData{
		FirstName: firstName,
		LastName:  p.LastName,
		FullName:  p.FullName,
		Company:   p.Company,
		Title:     p.Title,
		Location:  p.Location,
	}
}

// ActionType represents types of tracked actions for rate limiting
type ActionType string

const (
	ActionTypeConnection ActionType = "connection"
	ActionTypeMessage    ActionType = "message"
	ActionTypeSearch     ActionType = "search"
	ActionTypeProfileView ActionType = "profile_view"
)
