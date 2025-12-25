// Package storage - profile CRUD operations
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"linkedin-automation/internal/models"
)

// ProfileStore handles profile database operations
type ProfileStore struct {
	db *Database
}

// NewProfileStore creates a new ProfileStore
func NewProfileStore(db *Database) *ProfileStore {
	return &ProfileStore{db: db}
}

// Save inserts or updates a profile
func (s *ProfileStore) Save(profile *models.Profile) error {
	now := time.Now()

	result, err := s.db.db.Exec(`
		INSERT INTO profiles (url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			full_name = excluded.full_name,
			title = excluded.title,
			company = excluded.company,
			location = excluded.location,
			updated_at = excluded.updated_at
	`, profile.URL, profile.FirstName, profile.LastName, profile.FullName,
		profile.Title, profile.Company, profile.Location, profile.Status, now, now)

	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// Get the ID if it was an insert
	if profile.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			profile.ID = id
		}
	}

	return nil
}

// GetByURL retrieves a profile by URL
func (s *ProfileStore) GetByURL(url string) (*models.Profile, error) {
	profile := &models.Profile{}

	err := s.db.db.QueryRow(`
		SELECT id, url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at
		FROM profiles WHERE url = ?
	`, url).Scan(
		&profile.ID, &profile.URL, &profile.FirstName, &profile.LastName,
		&profile.FullName, &profile.Title, &profile.Company, &profile.Location,
		&profile.Status, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// GetByID retrieves a profile by ID
func (s *ProfileStore) GetByID(id int64) (*models.Profile, error) {
	profile := &models.Profile{}

	err := s.db.db.QueryRow(`
		SELECT id, url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at
		FROM profiles WHERE id = ?
	`, id).Scan(
		&profile.ID, &profile.URL, &profile.FirstName, &profile.LastName,
		&profile.FullName, &profile.Title, &profile.Company, &profile.Location,
		&profile.Status, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// GetPending retrieves profiles with 'found' status (not yet requested)
func (s *ProfileStore) GetPending(limit int) ([]*models.Profile, error) {
	rows, err := s.db.db.Query(`
		SELECT id, url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at
		FROM profiles 
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT ?
	`, models.ProfileStatusFound, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending profiles: %w", err)
	}
	defer rows.Close()

	return s.scanProfiles(rows)
}

// GetRequested retrieves profiles with 'requested' status (awaiting acceptance)
func (s *ProfileStore) GetRequested() ([]*models.Profile, error) {
	rows, err := s.db.db.Query(`
		SELECT id, url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at
		FROM profiles 
		WHERE status = ?
		ORDER BY updated_at ASC
	`, models.ProfileStatusRequested)

	if err != nil {
		return nil, fmt.Errorf("failed to get requested profiles: %w", err)
	}
	defer rows.Close()

	return s.scanProfiles(rows)
}

// GetConnected retrieves profiles with 'connected' status
func (s *ProfileStore) GetConnected() ([]*models.Profile, error) {
	rows, err := s.db.db.Query(`
		SELECT id, url, first_name, last_name, full_name, title, company, location, status, created_at, updated_at
		FROM profiles 
		WHERE status = ?
		ORDER BY updated_at DESC
	`, models.ProfileStatusConnected)

	if err != nil {
		return nil, fmt.Errorf("failed to get connected profiles: %w", err)
	}
	defer rows.Close()

	return s.scanProfiles(rows)
}

// UpdateStatus updates the status of a profile
func (s *ProfileStore) UpdateStatus(id int64, status models.ProfileStatus) error {
	_, err := s.db.db.Exec(`
		UPDATE profiles SET status = ?, updated_at = ? WHERE id = ?
	`, status, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update profile status: %w", err)
	}

	return nil
}

// Exists checks if a profile URL already exists
func (s *ProfileStore) Exists(url string) (bool, error) {
	var count int
	err := s.db.db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE url = ?`, url).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check profile existence: %w", err)
	}
	return count > 0, nil
}

// Count returns the total number of profiles
func (s *ProfileStore) Count() (int, error) {
	var count int
	err := s.db.db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count profiles: %w", err)
	}
	return count, nil
}

// CountByStatus returns the count of profiles with a specific status
func (s *ProfileStore) CountByStatus(status models.ProfileStatus) (int, error) {
	var count int
	err := s.db.db.QueryRow(`SELECT COUNT(*) FROM profiles WHERE status = ?`, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count profiles by status: %w", err)
	}
	return count, nil
}

// GetConnectedWithoutFollowup retrieves connected profiles that haven't received a follow-up message
func (s *ProfileStore) GetConnectedWithoutFollowup(limit int) ([]*models.Profile, error) {
	rows, err := s.db.db.Query(`
		SELECT p.id, p.url, p.first_name, p.last_name, p.full_name, p.title, p.company, p.location, p.status, p.created_at, p.updated_at
		FROM profiles p
		WHERE p.status = ?
		AND NOT EXISTS (
			SELECT 1 FROM messages m 
			WHERE m.profile_id = p.id 
			AND m.message_type = 'followup'
		)
		ORDER BY p.updated_at ASC
		LIMIT ?
	`, models.ProfileStatusConnected, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to get unmessaged connections: %w", err)
	}
	defer rows.Close()

	return s.scanProfiles(rows)
}

// scanProfiles scans rows into profile slice
func (s *ProfileStore) scanProfiles(rows *sql.Rows) ([]*models.Profile, error) {
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
