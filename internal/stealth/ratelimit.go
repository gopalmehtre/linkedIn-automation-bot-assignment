// Package stealth - rate limiting and throttling
// This is additional stealth technique #5
package stealth

import (
	"sync"
	"time"

	"github.com/rs/zerolog"

	"linkedin-automation/internal/models"
)

// RateLimiter handles rate limiting for different action types
type RateLimiter struct {
	logger       zerolog.Logger
	mu           sync.Mutex
	actionCounts map[models.ActionType][]time.Time
	cooldowns    map[models.ActionType]time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger zerolog.Logger) *RateLimiter {
	return &RateLimiter{
		logger:       logger.With().Str("module", "ratelimit").Logger(),
		actionCounts: make(map[models.ActionType][]time.Time),
		cooldowns:    make(map[models.ActionType]time.Time),
	}
}

// Limits defines rate limits for different actions
type Limits struct {
	DailyLimit      int
	HourlyLimit     int
	MinDelaySeconds int
	MaxDelaySeconds int
}

// DefaultLimits returns default rate limits for each action type
func DefaultLimits(actionType models.ActionType) Limits {
	switch actionType {
	case models.ActionTypeConnection:
		return Limits{
			DailyLimit:      50,
			HourlyLimit:     10,
			MinDelaySeconds: 30,
			MaxDelaySeconds: 90,
		}
	case models.ActionTypeMessage:
		return Limits{
			DailyLimit:      100,
			HourlyLimit:     20,
			MinDelaySeconds: 20,
			MaxDelaySeconds: 60,
		}
	case models.ActionTypeSearch:
		return Limits{
			DailyLimit:      500,
			HourlyLimit:     50,
			MinDelaySeconds: 5,
			MaxDelaySeconds: 15,
		}
	case models.ActionTypeProfileView:
		return Limits{
			DailyLimit:      100,
			HourlyLimit:     20,
			MinDelaySeconds: 10,
			MaxDelaySeconds: 30,
		}
	default:
		return Limits{
			DailyLimit:      100,
			HourlyLimit:     20,
			MinDelaySeconds: 10,
			MaxDelaySeconds: 30,
		}
	}
}

// CanPerform checks if an action can be performed based on rate limits
func (r *RateLimiter) CanPerform(actionType models.ActionType, limits Limits) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cooldown
	if cooldownEnd, ok := r.cooldowns[actionType]; ok {
		if time.Now().Before(cooldownEnd) {
			remaining := time.Until(cooldownEnd)
			r.logger.Debug().
				Str("action", string(actionType)).
				Dur("remaining", remaining).
				Msg("Action in cooldown")
			return false, "in cooldown"
		}
	}

	// Clean old entries
	r.cleanOldEntries(actionType)

	counts := r.actionCounts[actionType]

	// Check hourly limit
	hourlyCount := r.countInWindow(counts, time.Hour)
	if hourlyCount >= limits.HourlyLimit {
		r.logger.Debug().
			Str("action", string(actionType)).
			Int("hourlyCount", hourlyCount).
			Int("limit", limits.HourlyLimit).
			Msg("Hourly limit reached")
		return false, "hourly limit reached"
	}

	// Check daily limit
	dailyCount := r.countInWindow(counts, 24*time.Hour)
	if dailyCount >= limits.DailyLimit {
		r.logger.Debug().
			Str("action", string(actionType)).
			Int("dailyCount", dailyCount).
			Int("limit", limits.DailyLimit).
			Msg("Daily limit reached")
		return false, "daily limit reached"
	}

	return true, ""
}

// RecordAction records that an action was performed
func (r *RateLimiter) RecordAction(actionType models.ActionType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.actionCounts[actionType] == nil {
		r.actionCounts[actionType] = make([]time.Time, 0)
	}

	r.actionCounts[actionType] = append(r.actionCounts[actionType], time.Now())

	r.logger.Debug().
		Str("action", string(actionType)).
		Int("count", len(r.actionCounts[actionType])).
		Msg("Recorded action")
}

// SetCooldown sets a cooldown period for an action type
func (r *RateLimiter) SetCooldown(actionType models.ActionType, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cooldowns[actionType] = time.Now().Add(duration)

	r.logger.Debug().
		Str("action", string(actionType)).
		Dur("duration", duration).
		Msg("Set cooldown")
}

// GetRemainingQuota returns remaining actions for daily and hourly limits
func (r *RateLimiter) GetRemainingQuota(actionType models.ActionType, limits Limits) (daily, hourly int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanOldEntries(actionType)

	counts := r.actionCounts[actionType]

	hourlyCount := r.countInWindow(counts, time.Hour)
	dailyCount := r.countInWindow(counts, 24*time.Hour)

	return limits.DailyLimit - dailyCount, limits.HourlyLimit - hourlyCount
}

// GetLastActionTime returns the time of the last action of a given type
func (r *RateLimiter) GetLastActionTime(actionType models.ActionType) *time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	counts := r.actionCounts[actionType]
	if len(counts) == 0 {
		return nil
	}

	lastTime := counts[len(counts)-1]
	return &lastTime
}

// WaitForNextAction waits until the next action can be performed
func (r *RateLimiter) WaitForNextAction(actionType models.ActionType, limits Limits) {
	for {
		canPerform, reason := r.CanPerform(actionType, limits)
		if canPerform {
			return
		}

		var waitTime time.Duration

		switch reason {
		case "in cooldown":
			r.mu.Lock()
			if cooldownEnd, ok := r.cooldowns[actionType]; ok {
				waitTime = time.Until(cooldownEnd)
			}
			r.mu.Unlock()

		case "hourly limit reached":
			// Wait until oldest action in window expires
			waitTime = time.Minute * 5 // Check every 5 minutes

		case "daily limit reached":
			// Wait until tomorrow
			now := time.Now()
			tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			waitTime = time.Until(tomorrow)
		}

		if waitTime <= 0 {
			waitTime = time.Minute
		}

		r.logger.Info().
			Str("action", string(actionType)).
			Str("reason", reason).
			Dur("waitTime", waitTime).
			Msg("Waiting for rate limit")

		time.Sleep(waitTime)
	}
}

// cleanOldEntries removes entries older than 24 hours
func (r *RateLimiter) cleanOldEntries(actionType models.ActionType) {
	cutoff := time.Now().Add(-24 * time.Hour)

	counts := r.actionCounts[actionType]
	filtered := make([]time.Time, 0, len(counts))

	for _, t := range counts {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	r.actionCounts[actionType] = filtered
}

// countInWindow counts actions within a time window
func (r *RateLimiter) countInWindow(times []time.Time, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0

	for _, t := range times {
		if t.After(cutoff) {
			count++
		}
	}

	return count
}

// ResetLimits clears all rate limit tracking (use with caution)
func (r *RateLimiter) ResetLimits() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.actionCounts = make(map[models.ActionType][]time.Time)
	r.cooldowns = make(map[models.ActionType]time.Time)

	r.logger.Info().Msg("Rate limits reset")
}
