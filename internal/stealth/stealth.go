// Package stealth provides anti-detection techniques for browser automation.
// It implements human-like behavior patterns to avoid bot detection.
package stealth

import (
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// Controller orchestrates all stealth techniques
type Controller struct {
	config     *config.StealthConfig
	logger     zerolog.Logger
	mouse      *MouseController
	typing     *TypingController
	scroll     *ScrollController
	timing     *TimingController
	scheduler  *Scheduler
	rateLimit  *RateLimiter
}

// NewController creates a new stealth controller with all sub-modules
func NewController(cfg *config.StealthConfig, logger zerolog.Logger) *Controller {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	return &Controller{
		config:    cfg,
		logger:    logger.With().Str("component", "stealth").Logger(),
		mouse:     NewMouseController(cfg, logger),
		typing:    NewTypingController(cfg, logger),
		scroll:    NewScrollController(cfg, logger),
		timing:    NewTimingController(cfg, logger),
		scheduler: NewScheduler(cfg, logger),
		rateLimit: NewRateLimiter(logger),
	}
}

// Mouse returns the mouse controller for human-like mouse movements
func (c *Controller) Mouse() *MouseController {
	return c.mouse
}

// Typing returns the typing controller for human-like text input
func (c *Controller) Typing() *TypingController {
	return c.typing
}

// Scroll returns the scroll controller for natural scrolling
func (c *Controller) Scroll() *ScrollController {
	return c.scroll
}

// Timing returns the timing controller for randomized delays
func (c *Controller) Timing() *TimingController {
	return c.timing
}

// Scheduler returns the scheduler for activity timing
func (c *Controller) Scheduler() *Scheduler {
	return c.scheduler
}

// RateLimit returns the rate limiter
func (c *Controller) RateLimit() *RateLimiter {
	return c.rateLimit
}

// ApplyToPage applies stealth settings to a page via JavaScript injection
func (c *Controller) ApplyToPage(page *rod.Page) error {
	c.logger.Debug().Msg("Applying stealth settings to page")

	// Inject fingerprint masking JavaScript
	if err := ApplyFingerprint(page, c.logger); err != nil {
		return err
	}

	return nil
}

// RandomHover performs a random hover on visible elements (if enabled)
func (c *Controller) RandomHover(page *rod.Page) error {
	if !c.config.EnableRandomHovers {
		return nil
	}

	// Random chance based on config
	if rand.Float64() > c.config.HoverProbability {
		return nil
	}

	return HoverRandomElement(page, c.mouse, c.timing, c.logger)
}

// SimulateReading simulates a user reading content on the page
func (c *Controller) SimulateReading(page *rod.Page) error {
	c.logger.Debug().Msg("Simulating reading behavior")

	// Random scroll to simulate reading
	if err := c.scroll.ScrollRandom(page); err != nil {
		c.logger.Warn().Err(err).Msg("Failed to scroll during reading simulation")
	}

	// Think time
	c.timing.ThinkDelay()

	// Maybe hover some elements
	if err := c.RandomHover(page); err != nil {
		c.logger.Warn().Err(err).Msg("Failed to hover during reading simulation")
	}

	return nil
}

// WaitForSchedule blocks until we're within the allowed activity window
func (c *Controller) WaitForSchedule() {
	c.scheduler.WaitForSchedule()
}

// IsWithinSchedule checks if current time is within allowed window
func (c *Controller) IsWithinSchedule() bool {
	return c.scheduler.IsWithinSchedule()
}

// ShouldTakeBreak suggests if a break should be taken
func (c *Controller) ShouldTakeBreak(lastActivity time.Time, actionsCount int) bool {
	return c.scheduler.ShouldTakeBreak(lastActivity, actionsCount)
}
