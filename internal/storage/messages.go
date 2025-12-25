// Package storage - message CRUD operations
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"linkedin-automation/internal/models"
)

// MessageStore handles message database operations
type MessageStore struct {
	db *Database
}

// NewMessageStore creates a new MessageStore
func NewMessageStore(db *Database) *MessageStore {
	return &MessageStore{db: db}
}

// RecordMessage inserts a new message
func (s *MessageStore) RecordMessage(profileID int64, messageText string, messageType models.MessageType) (*models.Message, error) {
	now := time.Now()

	result, err := s.db.db.Exec(`
		INSERT INTO messages (profile_id, message_text, message_type, sent_at)
		VALUES (?, ?, ?, ?)
	`, profileID, messageText, messageType, now)

	if err != nil {
		return nil, fmt.Errorf("failed to record message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Message{
		ID:          id,
		ProfileID:   profileID,
		MessageText: messageText,
		MessageType: messageType,
		SentAt:      now,
	}, nil
}

// GetByProfileID retrieves all messages for a profile
func (s *MessageStore) GetByProfileID(profileID int64) ([]*models.Message, error) {
	rows, err := s.db.db.Query(`
		SELECT id, profile_id, message_text, message_type, sent_at
		FROM messages WHERE profile_id = ?
		ORDER BY sent_at ASC
	`, profileID)

	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	return s.scanMessages(rows)
}

// HasFollowup checks if a follow-up message has been sent to a profile
func (s *MessageStore) HasFollowup(profileID int64) (bool, error) {
	var count int
	err := s.db.db.QueryRow(`
		SELECT COUNT(*) FROM messages 
		WHERE profile_id = ? AND message_type = ?
	`, profileID, models.MessageTypeFollowup).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check followup: %w", err)
	}

	return count > 0, nil
}

// GetTodayCount returns the number of messages sent today
func (s *MessageStore) GetTodayCount() (int, error) {
	var count int
	today := GetTodayDate()

	err := s.db.db.QueryRow(`
		SELECT COUNT(*) FROM messages 
		WHERE DATE(sent_at) = ?
	`, today).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get today's message count: %w", err)
	}

	return count, nil
}

// GetHourCount returns the number of messages sent in the last hour
func (s *MessageStore) GetHourCount() (int, error) {
	var count int
	oneHourAgo := time.Now().Add(-time.Hour)

	err := s.db.db.QueryRow(`
		SELECT COUNT(*) FROM messages 
		WHERE sent_at >= ?
	`, oneHourAgo).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get hourly message count: %w", err)
	}

	return count, nil
}

// GetUnmessagedConnections retrieves connected profiles without a follow-up message
func (s *MessageStore) GetUnmessagedConnections(db *Database) ([]*models.Profile, error) {
	rows, err := db.db.Query(`
		SELECT p.id, p.url, p.first_name, p.last_name, p.full_name, p.title, p.company, p.location, p.status, p.created_at, p.updated_at
		FROM profiles p
		WHERE p.status = ?
		AND NOT EXISTS (
			SELECT 1 FROM messages m 
			WHERE m.profile_id = p.id AND m.message_type = ?
		)
		ORDER BY p.updated_at ASC
	`, models.ProfileStatusConnected, models.MessageTypeFollowup)

	if err != nil {
		return nil, fmt.Errorf("failed to get unmessaged connections: %w", err)
	}
	defer rows.Close()

	var profiles []*models.Profile
	for rows.Next() {
		profile := &models.Profile{}
		err := rows.Scan(
			&profile.ID, &profile.URL, &profile.FirstName, &profile.LastName,
			&profile.FullName, &profile.Title, &profile.Company, &profile.Location,
			&profile.Status, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// GetLastMessageTime returns the time of the last message sent
func (s *MessageStore) GetLastMessageTime() (*time.Time, error) {
	var sentAt time.Time

	err := s.db.db.QueryRow(`
		SELECT sent_at FROM messages 
		ORDER BY sent_at DESC LIMIT 1
	`).Scan(&sentAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last message time: %w", err)
	}

	return &sentAt, nil
}

// scanMessages scans rows into message slice
func (s *MessageStore) scanMessages(rows *sql.Rows) ([]*models.Message, error) {
	var messages []*models.Message

	for rows.Next() {
		msg := &models.Message{}
		err := rows.Scan(
			&msg.ID, &msg.ProfileID, &msg.MessageText, &msg.MessageType, &msg.SentAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
