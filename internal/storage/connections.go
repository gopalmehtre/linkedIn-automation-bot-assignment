// Package storage - connection request CRUD operations
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"linkedin-automation/internal/models"
)

// ConnectionStore handles connection request database operations
type ConnectionStore struct {
	db *Database
}

// NewConnectionStore creates a new ConnectionStore
func NewConnectionStore(db *Database) *ConnectionStore {
	return &ConnectionStore{db: db}
}

// RecordRequest inserts a new connection request
func (s *ConnectionStore) RecordRequest(profileID int64, noteText string) (*models.ConnectionRequest, error) {
	now := time.Now()

	result, err := s.db.db.Exec(`
		INSERT INTO connection_requests (profile_id, note_text, status, sent_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, profileID, noteText, models.ConnectionStatusPending, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to record connection request: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.ConnectionRequest{
		ID:        id,
		ProfileID: profileID,
		NoteText:  noteText,
		Status:    models.ConnectionStatusPending,
		SentAt:    now,
		UpdatedAt: now,
	}, nil
}

// GetByProfileID retrieves a connection request by profile ID
func (s *ConnectionStore) GetByProfileID(profileID int64) (*models.ConnectionRequest, error) {
	req := &models.ConnectionRequest{}

	err := s.db.db.QueryRow(`
		SELECT id, profile_id, note_text, status, sent_at, updated_at
		FROM connection_requests WHERE profile_id = ?
		ORDER BY sent_at DESC LIMIT 1
	`, profileID).Scan(
		&req.ID, &req.ProfileID, &req.NoteText, &req.Status, &req.SentAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connection request: %w", err)
	}

	return req, nil
}

// GetPending retrieves all pending connection requests
func (s *ConnectionStore) GetPending() ([]*models.ConnectionRequest, error) {
	rows, err := s.db.db.Query(`
		SELECT id, profile_id, note_text, status, sent_at, updated_at
		FROM connection_requests 
		WHERE status = ?
		ORDER BY sent_at ASC
	`, models.ConnectionStatusPending)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}
	defer rows.Close()

	return s.scanRequests(rows)
}

// MarkAccepted updates a connection request to accepted status
func (s *ConnectionStore) MarkAccepted(profileID int64) error {
	_, err := s.db.db.Exec(`
		UPDATE connection_requests 
		SET status = ?, updated_at = ? 
		WHERE profile_id = ? AND status = ?
	`, models.ConnectionStatusAccepted, time.Now(), profileID, models.ConnectionStatusPending)

	if err != nil {
		return fmt.Errorf("failed to mark request accepted: %w", err)
	}

	return nil
}

// MarkIgnored updates a connection request to ignored status
func (s *ConnectionStore) MarkIgnored(profileID int64) error {
	_, err := s.db.db.Exec(`
		UPDATE connection_requests 
		SET status = ?, updated_at = ? 
		WHERE profile_id = ? AND status = ?
	`, models.ConnectionStatusIgnored, time.Now(), profileID, models.ConnectionStatusPending)

	if err != nil {
		return fmt.Errorf("failed to mark request ignored: %w", err)
	}

	return nil
}

// GetTodayCount returns the number of connection requests sent today
func (s *ConnectionStore) GetTodayCount() (int, error) {
	var count int
	today := GetTodayDate()

	err := s.db.db.QueryRow(`
		SELECT COUNT(*) FROM connection_requests 
		WHERE DATE(sent_at) = ?
	`, today).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get today's request count: %w", err)
	}

	return count, nil
}

// GetHourCount returns the number of connection requests sent in the last hour
func (s *ConnectionStore) GetHourCount() (int, error) {
	var count int
	oneHourAgo := time.Now().Add(-time.Hour)

	err := s.db.db.QueryRow(`
		SELECT COUNT(*) FROM connection_requests 
		WHERE sent_at >= ?
	`, oneHourAgo).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get hourly request count: %w", err)
	}

	return count, nil
}

// GetLastRequestTime returns the time of the last connection request
func (s *ConnectionStore) GetLastRequestTime() (*time.Time, error) {
	var sentAt time.Time

	err := s.db.db.QueryRow(`
		SELECT sent_at FROM connection_requests 
		ORDER BY sent_at DESC LIMIT 1
	`).Scan(&sentAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last request time: %w", err)
	}

	return &sentAt, nil
}

// scanRequests scans rows into connection request slice
func (s *ConnectionStore) scanRequests(rows *sql.Rows) ([]*models.ConnectionRequest, error) {
	var requests []*models.ConnectionRequest

	for rows.Next() {
		req := &models.ConnectionRequest{}
		err := rows.Scan(
			&req.ID, &req.ProfileID, &req.NoteText, &req.Status, &req.SentAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}
		requests = append(requests, req)
	}

	return requests, rows.Err()
}
