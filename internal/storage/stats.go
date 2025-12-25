// Package storage - daily statistics tracking for rate limiting
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"linkedin-automation/internal/models"
)

// StatsStore handles daily statistics database operations
type StatsStore struct {
	db *Database
}

// NewStatsStore creates a new StatsStore
func NewStatsStore(db *Database) *StatsStore {
	return &StatsStore{db: db}
}

// GetOrCreateToday gets today's stats or creates a new record
func (s *StatsStore) GetOrCreateToday() (*models.DailyStats, error) {
	today := GetTodayDate()
	now := time.Now()

	// Try to get existing
	stats, err := s.GetByDate(today)
	if err != nil {
		return nil, err
	}

	if stats != nil {
		return stats, nil
	}

	// Create new record for today
	result, err := s.db.db.Exec(`
		INSERT INTO daily_stats (date, connections_sent, messages_sent, profiles_searched, last_activity_at, created_at, updated_at)
		VALUES (?, 0, 0, 0, ?, ?, ?)
	`, today, now, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create daily stats: %w", err)
	}

	id, _ := result.LastInsertId()

	return &models.DailyStats{
		ID:               id,
		Date:             today,
		ConnectionsSent:  0,
		MessagesSent:     0,
		ProfilesSearched: 0,
		LastActivityAt:   now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// GetByDate retrieves stats for a specific date
func (s *StatsStore) GetByDate(date string) (*models.DailyStats, error) {
	stats := &models.DailyStats{}

	err := s.db.db.QueryRow(`
		SELECT id, date, connections_sent, messages_sent, profiles_searched, last_activity_at, created_at, updated_at
		FROM daily_stats WHERE date = ?
	`, date).Scan(
		&stats.ID, &stats.Date, &stats.ConnectionsSent, &stats.MessagesSent,
		&stats.ProfilesSearched, &stats.LastActivityAt, &stats.CreatedAt, &stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	return stats, nil
}

// IncrementConnections increments today's connection count
func (s *StatsStore) IncrementConnections() error {
	today := GetTodayDate()
	now := time.Now()

	// Ensure today's record exists
	_, err := s.GetOrCreateToday()
	if err != nil {
		return err
	}

	_, err = s.db.db.Exec(`
		UPDATE daily_stats 
		SET connections_sent = connections_sent + 1, last_activity_at = ?, updated_at = ?
		WHERE date = ?
	`, now, now, today)

	if err != nil {
		return fmt.Errorf("failed to increment connections: %w", err)
	}

	return nil
}

// IncrementMessages increments today's message count
func (s *StatsStore) IncrementMessages() error {
	today := GetTodayDate()
	now := time.Now()

	// Ensure today's record exists
	_, err := s.GetOrCreateToday()
	if err != nil {
		return err
	}

	_, err = s.db.db.Exec(`
		UPDATE daily_stats 
		SET messages_sent = messages_sent + 1, last_activity_at = ?, updated_at = ?
		WHERE date = ?
	`, now, now, today)

	if err != nil {
		return fmt.Errorf("failed to increment messages: %w", err)
	}

	return nil
}

// IncrementSearches increments today's profile search count
func (s *StatsStore) IncrementSearches(count int) error {
	today := GetTodayDate()
	now := time.Now()

	// Ensure today's record exists
	_, err := s.GetOrCreateToday()
	if err != nil {
		return err
	}

	_, err = s.db.db.Exec(`
		UPDATE daily_stats 
		SET profiles_searched = profiles_searched + ?, last_activity_at = ?, updated_at = ?
		WHERE date = ?
	`, count, now, now, today)

	if err != nil {
		return fmt.Errorf("failed to increment searches: %w", err)
	}

	return nil
}

// GetWeeklyStats retrieves stats for the last 7 days
func (s *StatsStore) GetWeeklyStats() ([]*models.DailyStats, error) {
	rows, err := s.db.db.Query(`
		SELECT id, date, connections_sent, messages_sent, profiles_searched, last_activity_at, created_at, updated_at
		FROM daily_stats 
		WHERE date >= DATE('now', '-7 days')
		ORDER BY date DESC
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to get weekly stats: %w", err)
	}
	defer rows.Close()

	var stats []*models.DailyStats
	for rows.Next() {
		s := &models.DailyStats{}
		err := rows.Scan(
			&s.ID, &s.Date, &s.ConnectionsSent, &s.MessagesSent,
			&s.ProfilesSearched, &s.LastActivityAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// CanSendConnection checks if we can send another connection today
func (s *StatsStore) CanSendConnection(limit int) (bool, int, error) {
	stats, err := s.GetOrCreateToday()
	if err != nil {
		return false, 0, err
	}

	remaining := limit - stats.ConnectionsSent
	return remaining > 0, remaining, nil
}

// CanSendMessage checks if we can send another message today
func (s *StatsStore) CanSendMessage(limit int) (bool, int, error) {
	stats, err := s.GetOrCreateToday()
	if err != nil {
		return false, 0, err
	}

	remaining := limit - stats.MessagesSent
	return remaining > 0, remaining, nil
}
