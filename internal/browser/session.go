// Package browser - cookie/session persistence
package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog"
)

// SessionManager handles browser session persistence
type SessionManager struct {
	cookiesPath string
	logger      zerolog.Logger
}

// CookieData represents a serializable cookie
type CookieData struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(cookiesPath string, logger zerolog.Logger) *SessionManager {
	return &SessionManager{
		cookiesPath: cookiesPath,
		logger:      logger.With().Str("component", "session").Logger(),
	}
}

// SaveCookies saves browser cookies to a file
func (s *SessionManager) SaveCookies(browser *rod.Browser) error {
	s.logger.Debug().Msg("Saving cookies")

	// Get all cookies
	cookies, err := browser.GetCookies()
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Convert to serializable format
	cookieData := make([]CookieData, len(cookies))
	for i, c := range cookies {
		sameSite := "Lax"
		if c.SameSite == proto.NetworkCookieSameSiteStrict {
			sameSite = "Strict"
		} else if c.SameSite == proto.NetworkCookieSameSiteNone {
			sameSite = "None"
		}

		cookieData[i] = CookieData{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: sameSite,
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(s.cookiesPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cookies directory: %w", err)
	}

	// Save to file
	data, err := json.MarshalIndent(cookieData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	if err := os.WriteFile(s.cookiesPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	s.logger.Info().
		Int("count", len(cookies)).
		Str("path", s.cookiesPath).
		Msg("Cookies saved")

	return nil
}

// LoadCookies loads cookies from file into browser
func (s *SessionManager) LoadCookies(browser *rod.Browser) error {
	s.logger.Debug().Msg("Loading cookies")

	// Check if file exists
	if _, err := os.Stat(s.cookiesPath); os.IsNotExist(err) {
		s.logger.Debug().Msg("No saved cookies found")
		return nil
	}

	// Read file
	data, err := os.ReadFile(s.cookiesPath)
	if err != nil {
		return fmt.Errorf("failed to read cookies file: %w", err)
	}

	// Parse cookies
	var cookieData []CookieData
	if err := json.Unmarshal(data, &cookieData); err != nil {
		return fmt.Errorf("failed to parse cookies: %w", err)
	}

	// Filter out expired cookies
	now := time.Now()
	validCookies := make([]*proto.NetworkCookieParam, 0, len(cookieData))

	for _, c := range cookieData {
		// Check expiration
		if c.Expires > 0 && c.Expires < float64(now.Unix()) {
			continue // Skip expired
		}

		sameSite := proto.NetworkCookieSameSiteLax
		switch c.SameSite {
		case "Strict":
			sameSite = proto.NetworkCookieSameSiteStrict
		case "None":
			sameSite = proto.NetworkCookieSameSiteNone
		}

		validCookies = append(validCookies, &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  proto.TimeSinceEpoch(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: sameSite,
		})
	}

	// Set cookies
	if len(validCookies) > 0 {
		if err := browser.SetCookies(validCookies); err != nil {
			return fmt.Errorf("failed to set cookies: %w", err)
		}
	}

	s.logger.Info().
		Int("loaded", len(validCookies)).
		Int("total", len(cookieData)).
		Msg("Cookies loaded")

	return nil
}

// ClearCookies deletes saved cookies
func (s *SessionManager) ClearCookies() error {
	if _, err := os.Stat(s.cookiesPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(s.cookiesPath); err != nil {
		return fmt.Errorf("failed to delete cookies file: %w", err)
	}

	s.logger.Info().Msg("Cookies cleared")
	return nil
}

// HasSavedSession checks if a saved session exists
func (s *SessionManager) HasSavedSession() bool {
	_, err := os.Stat(s.cookiesPath)
	return err == nil
}

// GetSessionAge returns how old the saved session is
func (s *SessionManager) GetSessionAge() (time.Duration, error) {
	info, err := os.Stat(s.cookiesPath)
	if err != nil {
		return 0, err
	}

	return time.Since(info.ModTime()), nil
}

// IsSessionValid checks if saved session is likely still valid
// LinkedIn sessions typically last ~24 hours
func (s *SessionManager) IsSessionValid() bool {
	age, err := s.GetSessionAge()
	if err != nil {
		return false
	}

	// Consider session invalid if older than 20 hours
	return age < 20*time.Hour
}
