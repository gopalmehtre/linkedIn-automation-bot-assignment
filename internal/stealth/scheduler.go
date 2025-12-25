// Package stealth - activity scheduling
// This is additional stealth technique #4
package stealth

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// Scheduler handles activity timing and business hours compliance
type Scheduler struct {
	config          *config.StealthConfig
	logger          zerolog.Logger
	lastActivityAt  time.Time
	sessionStartAt  time.Time
	actionsThisHour int
}

// NewScheduler creates a new activity scheduler
func NewScheduler(cfg *config.StealthConfig, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		config:         cfg,
		logger:         logger.With().Str("module", "scheduler").Logger(),
		sessionStartAt: time.Now(),
	}
}

// IsWithinSchedule checks if current time is within allowed activity window
func (s *Scheduler) IsWithinSchedule() bool {
	if !s.config.BusinessHoursOnly {
		return true
	}

	now := time.Now()
	hour := now.Hour()

	// Check if within business hours
	if hour < s.config.StartHour || hour >= s.config.EndHour {
		return false
	}

	// Check if it's a weekday (Monday = 1, Sunday = 0)
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		// 20% chance to work on weekends (some people do)
		return rand.Float64() < 0.2
	}

	return true
}

// WaitForSchedule blocks until we're within the allowed activity window
func (s *Scheduler) WaitForSchedule() {
	if !s.config.BusinessHoursOnly {
		return
	}

	for !s.IsWithinSchedule() {
		now := time.Now()
		hour := now.Hour()

		// Calculate time until next valid window
		var waitUntil time.Time

		if hour >= s.config.EndHour {
			// Wait until tomorrow's start hour
			tomorrow := now.AddDate(0, 0, 1)
			waitUntil = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
				s.config.StartHour, 0, 0, 0, now.Location())
		} else {
			// Wait until today's start hour
			waitUntil = time.Date(now.Year(), now.Month(), now.Day(),
				s.config.StartHour, 0, 0, 0, now.Location())
		}

		// Add some randomization to start time (±30 minutes)
		waitUntil = waitUntil.Add(time.Duration(rand.Intn(60)-30) * time.Minute)

		waitDuration := time.Until(waitUntil)

		s.logger.Info().
			Time("waitUntil", waitUntil).
			Dur("duration", waitDuration).
			Msg("Outside business hours, waiting for schedule")

		// Wait in chunks to allow for interruption
		for time.Now().Before(waitUntil) {
			// Sleep for 1 minute at a time
			sleepTime := time.Minute
			remaining := time.Until(waitUntil)
			if remaining < sleepTime {
				sleepTime = remaining
			}
			time.Sleep(sleepTime)
		}
	}

	s.logger.Info().Msg("Within scheduled hours, proceeding")
}

// ShouldTakeBreak suggests if a break should be taken based on activity patterns
func (s *Scheduler) ShouldTakeBreak(lastActivity time.Time, actionsCount int) bool {
	// Take a break every 20-40 actions
	if actionsCount > 0 && actionsCount%(20+rand.Intn(20)) == 0 {
		s.logger.Info().
			Int("actionsCount", actionsCount).
			Msg("Suggesting break based on action count")
		return true
	}

	// Take a break if working for more than 45-90 minutes straight
	sessionDuration := time.Since(s.sessionStartAt)
	breakThreshold := time.Duration(45+rand.Intn(45)) * time.Minute

	if sessionDuration > breakThreshold {
		s.logger.Info().
			Dur("sessionDuration", sessionDuration).
			Msg("Suggesting break based on session duration")
		return true
	}

	// Random chance for spontaneous break (simulates human unpredictability)
	if rand.Float64() < 0.02 { // 2% chance
		s.logger.Info().Msg("Suggesting spontaneous break")
		return true
	}

	return false
}

// TakeBreak simulates a human-like break
func (s *Scheduler) TakeBreak() {
	// Break duration: 2-10 minutes
	breakMinutes := 2 + rand.Intn(8)
	breakDuration := time.Duration(breakMinutes) * time.Minute

	s.logger.Info().
		Dur("duration", breakDuration).
		Msg("Taking a break")

	time.Sleep(breakDuration)

	// Reset session start time after break
	s.sessionStartAt = time.Now()

	s.logger.Info().Msg("Break finished, resuming activity")
}

// RecordActivity records that an action was performed
func (s *Scheduler) RecordActivity() {
	s.lastActivityAt = time.Now()
	s.actionsThisHour++
}

// GetSessionDuration returns how long the current session has been active
func (s *Scheduler) GetSessionDuration() time.Duration {
	return time.Since(s.sessionStartAt)
}

// GetTimeSinceLastActivity returns time since last recorded activity
func (s *Scheduler) GetTimeSinceLastActivity() time.Duration {
	if s.lastActivityAt.IsZero() {
		return 0
	}
	return time.Since(s.lastActivityAt)
}

// ResetSession resets session tracking (call after a break)
func (s *Scheduler) ResetSession() {
	s.sessionStartAt = time.Now()
	s.actionsThisHour = 0
}

// GetOptimalActivityWindow returns a suggested time range for activity today
func (s *Scheduler) GetOptimalActivityWindow() (start, end time.Time) {
	now := time.Now()

	// Randomize start within first 2 hours of business hours
	startHour := s.config.StartHour + rand.Intn(2)
	startMinute := rand.Intn(60)

	// Randomize end within last 2 hours of business hours
	endHour := s.config.EndHour - rand.Intn(2)
	endMinute := rand.Intn(60)

	start = time.Date(now.Year(), now.Month(), now.Day(), startHour, startMinute, 0, 0, now.Location())
	end = time.Date(now.Year(), now.Month(), now.Day(), endHour, endMinute, 0, 0, now.Location())

	return start, end
}
