// Package stealth - randomized timing patterns
// This is MANDATORY stealth technique #2
package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// TimingController handles randomized delays to simulate human behavior
type TimingController struct {
	config *config.StealthConfig
	logger zerolog.Logger
}

// NewTimingController creates a new timing controller
func NewTimingController(cfg *config.StealthConfig, logger zerolog.Logger) *TimingController {
	return &TimingController{
		config: cfg,
		logger: logger.With().Str("module", "timing").Logger(),
	}
}

// RandomDelay sleeps for a random duration between min and max seconds
func (t *TimingController) RandomDelay(minSeconds, maxSeconds int) {
	if minSeconds >= maxSeconds {
		minSeconds = 1
		maxSeconds = 3
	}

	// Use normal distribution for more natural variance
	mean := float64(minSeconds+maxSeconds) / 2
	stdDev := float64(maxSeconds-minSeconds) / 4

	delay := t.normalRandom(mean, stdDev)

	// Clamp to bounds
	if delay < float64(minSeconds) {
		delay = float64(minSeconds)
	}
	if delay > float64(maxSeconds) {
		delay = float64(maxSeconds)
	}

	duration := time.Duration(delay * float64(time.Second))

	t.logger.Debug().
		Dur("delay", duration).
		Msg("Random delay")

	time.Sleep(duration)
}

// ThinkDelay simulates time spent reading/thinking (2-5 seconds)
func (t *TimingController) ThinkDelay() {
	delay := 2.0 + rand.Float64()*3.0
	duration := time.Duration(delay * float64(time.Second))

	t.logger.Debug().
		Dur("delay", duration).
		Msg("Think delay")

	time.Sleep(duration)
}

// ActionDelay adds delay between user actions (0.5-2 seconds)
func (t *TimingController) ActionDelay() {
	delay := 0.5 + rand.Float64()*1.5
	duration := time.Duration(delay * float64(time.Second))

	t.logger.Debug().
		Dur("delay", duration).
		Msg("Action delay")

	time.Sleep(duration)
}

// PageLoadDelay waits after page navigation (1-3 seconds)
func (t *TimingController) PageLoadDelay() {
	delay := 1.0 + rand.Float64()*2.0
	duration := time.Duration(delay * float64(time.Second))

	t.logger.Debug().
		Dur("delay", duration).
		Msg("Page load delay")

	time.Sleep(duration)
}

// ShortDelay adds a brief pause (100-500ms)
func (t *TimingController) ShortDelay() {
	delay := 100 + rand.Intn(400)
	duration := time.Duration(delay) * time.Millisecond

	time.Sleep(duration)
}

// MicroDelay adds a very brief pause (10-50ms)
func (t *TimingController) MicroDelay() {
	delay := 10 + rand.Intn(40)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// BetweenActionsDelay uses config min/max delay settings
func (t *TimingController) BetweenActionsDelay() {
	minDelay := t.config.MinTypingDelayMs
	maxDelay := t.config.MaxTypingDelayMs

	if minDelay <= 0 {
		minDelay = 30
	}
	if maxDelay <= minDelay {
		maxDelay = minDelay + 100
	}

	delay := minDelay + rand.Intn(maxDelay-minDelay)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// LongDelay simulates a longer pause (5-15 seconds)
// Used between major actions like viewing profiles
func (t *TimingController) LongDelay() {
	delay := 5.0 + rand.Float64()*10.0
	duration := time.Duration(delay * float64(time.Second))

	t.logger.Debug().
		Dur("delay", duration).
		Msg("Long delay")

	time.Sleep(duration)
}

// VeryLongDelay simulates an extended break (30-90 seconds)
// Used to simulate taking breaks
func (t *TimingController) VeryLongDelay() {
	delay := 30.0 + rand.Float64()*60.0
	duration := time.Duration(delay * float64(time.Second))

	t.logger.Info().
		Dur("delay", duration).
		Msg("Taking a break (very long delay)")

	time.Sleep(duration)
}

// ConfiguredDelay uses the min/max delay from config
func (t *TimingController) ConfiguredDelay(minSec, maxSec int) {
	t.RandomDelay(minSec, maxSec)
}

// normalRandom generates a random number from normal distribution
func (t *TimingController) normalRandom(mean, stdDev float64) float64 {
	// Box-Muller transform for normal distribution
	u1 := rand.Float64()
	u2 := rand.Float64()

	// Avoid log(0)
	for u1 == 0 {
		u1 = rand.Float64()
	}

	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + z*stdDev
}

// JitteredDelay adds a delay with random jitter around a base value
func (t *TimingController) JitteredDelay(baseMs int, jitterPercent float64) {
	jitter := float64(baseMs) * jitterPercent
	delay := float64(baseMs) + (rand.Float64()*2-1)*jitter

	if delay < 0 {
		delay = float64(baseMs)
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// WaitWithProgress shows progress while waiting (for CLI feedback)
func (t *TimingController) WaitWithProgress(seconds int, message string) {
	t.logger.Info().
		Int("seconds", seconds).
		Str("reason", message).
		Msg("Waiting")

	time.Sleep(time.Duration(seconds) * time.Second)
}

// ExponentialBackoff implements exponential backoff with jitter
func (t *TimingController) ExponentialBackoff(attempt int, baseMs int, maxMs int) {
	// Calculate delay: baseMs * 2^attempt
	delay := float64(baseMs) * math.Pow(2, float64(attempt))

	// Add jitter (±25%)
	jitter := delay * 0.25 * (rand.Float64()*2 - 1)
	delay += jitter

	// Cap at max
	if delay > float64(maxMs) {
		delay = float64(maxMs)
	}

	t.logger.Debug().
		Int("attempt", attempt).
		Float64("delayMs", delay).
		Msg("Exponential backoff")

	time.Sleep(time.Duration(delay) * time.Millisecond)
}
