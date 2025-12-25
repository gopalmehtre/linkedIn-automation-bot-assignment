// Package storage provides SQLite database operations for persistence.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Database wraps the SQLite database connection
type Database struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at the given path
func Open(dbPath string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	database := &Database{db: db}

	// Run migrations
	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// DB returns the underlying sql.DB for advanced operations
func (d *Database) DB() *sql.DB {
	return d.db
}

// Migrate creates all necessary tables
func (d *Database) Migrate() error {
	migrations := []string{
		// Profiles table
		`CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			first_name TEXT DEFAULT '',
			last_name TEXT DEFAULT '',
			full_name TEXT DEFAULT '',
			title TEXT DEFAULT '',
			company TEXT DEFAULT '',
			location TEXT DEFAULT '',
			status TEXT DEFAULT 'found',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Connection requests table
		`CREATE TABLE IF NOT EXISTS connection_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER NOT NULL,
			note_text TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES profiles(id)
		)`,

		// Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id INTEGER NOT NULL,
			message_text TEXT NOT NULL,
			message_type TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES profiles(id)
		)`,

		// Daily stats table
		`CREATE TABLE IF NOT EXISTS daily_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT UNIQUE NOT NULL,
			connections_sent INTEGER DEFAULT 0,
			messages_sent INTEGER DEFAULT 0,
			profiles_searched INTEGER DEFAULT 0,
			last_activity_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cookies TEXT NOT NULL,
			user_agent TEXT DEFAULT '',
			is_valid INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_profiles_status ON profiles(status)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_url ON profiles(url)`,
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_status ON connection_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_profile ON connection_requests(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_profile ON messages(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date)`,
	}

	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, migration)
		}
	}

	return nil
}

// GetTodayDate returns today's date in YYYY-MM-DD format
func GetTodayDate() string {
	return time.Now().Format("2006-01-02")
}

// Transaction helper for running operations in a transaction
func (d *Database) Transaction(fn func(*sql.Tx) error) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
