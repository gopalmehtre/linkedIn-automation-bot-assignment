// Package stealth - natural scrolling behavior
// This is additional stealth technique #1
package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// ScrollController handles human-like scrolling behavior
type ScrollController struct {
	config *config.StealthConfig
	logger zerolog.Logger
}

// NewScrollController creates a new scroll controller
func NewScrollController(cfg *config.StealthConfig, logger zerolog.Logger) *ScrollController {
	return &ScrollController{
		config: cfg,
		logger: logger.With().Str("module", "scroll").Logger(),
	}
}

// ScrollTo scrolls to a specific Y position with human-like motion
func (s *ScrollController) ScrollTo(page *rod.Page, targetY int) error {
	s.logger.Debug().Int("targetY", targetY).Msg("Scrolling to position")

	// Get current scroll position
	currentY, err := s.getScrollPosition(page)
	if err != nil {
		return err
	}

	// Calculate distance
	distance := targetY - currentY
	if math.Abs(float64(distance)) < 10 {
		return nil // Already there
	}

	// Break into steps with acceleration/deceleration
	steps := s.calculateScrollSteps(currentY, targetY)

	for _, step := range steps {
		// Execute scroll
		_, err := page.Eval(`window.scrollTo(0, ` + itoa(step.position) + `)`)
		if err != nil {
			return err
		}

		// Wait based on step duration
		time.Sleep(time.Duration(step.delayMs) * time.Millisecond)
	}

	// Occasionally scroll back a bit (natural overshoot)
	if s.config.EnableScrollBack && rand.Float64() < 0.2 {
		s.applyScrollCorrection(page, targetY)
	}

	return nil
}

// ScrollIntoView scrolls an element into view with natural motion
func (s *ScrollController) ScrollIntoView(page *rod.Page, element *rod.Element) error {
	// Get element position
	box, err := element.Shape()
	if err != nil {
		return err
	}

	if len(box.Quads) == 0 {
		// Fallback to default scroll
		return element.ScrollIntoView()
	}

	// Get element Y position
	elementY := int(box.Quads[0][1])

	// Get viewport height
	viewportHeight, err := s.getViewportHeight(page)
	if err != nil {
		return err
	}

	// Calculate target scroll position (element in middle of viewport)
	targetY := elementY - viewportHeight/2

	if targetY < 0 {
		targetY = 0
	}

	return s.ScrollTo(page, targetY)
}

// ScrollDown scrolls down by a random amount
func (s *ScrollController) ScrollDown(page *rod.Page) error {
	scrollAmount := s.config.ScrollSpeedMin + rand.Intn(s.config.ScrollSpeedMax-s.config.ScrollSpeedMin)

	currentY, err := s.getScrollPosition(page)
	if err != nil {
		return err
	}

	return s.ScrollTo(page, currentY+scrollAmount)
}

// ScrollUp scrolls up by a random amount
func (s *ScrollController) ScrollUp(page *rod.Page) error {
	scrollAmount := s.config.ScrollSpeedMin + rand.Intn(s.config.ScrollSpeedMax-s.config.ScrollSpeedMin)

	currentY, err := s.getScrollPosition(page)
	if err != nil {
		return err
	}

	targetY := currentY - scrollAmount
	if targetY < 0 {
		targetY = 0
	}

	return s.ScrollTo(page, targetY)
}

// ScrollRandom performs a random scroll action
func (s *ScrollController) ScrollRandom(page *rod.Page) error {
	// 70% chance to scroll down, 30% chance to scroll up
	if rand.Float64() < 0.7 {
		return s.ScrollDown(page)
	}
	return s.ScrollUp(page)
}

// ScrollToTop scrolls to the top of the page
func (s *ScrollController) ScrollToTop(page *rod.Page) error {
	return s.ScrollTo(page, 0)
}

// ScrollToBottom scrolls to the bottom of the page
func (s *ScrollController) ScrollToBottom(page *rod.Page) error {
	height, err := s.getPageHeight(page)
	if err != nil {
		return err
	}

	viewportHeight, err := s.getViewportHeight(page)
	if err != nil {
		return err
	}

	return s.ScrollTo(page, height-viewportHeight)
}

// scrollStep represents a single scroll step
type scrollStep struct {
	position int
	delayMs  int
}

// calculateScrollSteps generates scroll steps with acceleration/deceleration
func (s *ScrollController) calculateScrollSteps(startY, endY int) []scrollStep {
	distance := endY - startY
	absDistance := int(math.Abs(float64(distance)))

	// Number of steps based on distance
	numSteps := int(math.Max(5, float64(absDistance)/50))
	if numSteps > 30 {
		numSteps = 30
	}

	steps := make([]scrollStep, numSteps)

	for i := 0; i < numSteps; i++ {
		// Progress through scroll (0 to 1)
		t := float64(i+1) / float64(numSteps)

		// Ease-out cubic for natural deceleration
		eased := 1 - math.Pow(1-t, 3)

		// Calculate position
		position := startY + int(float64(distance)*eased)

		// Calculate delay (faster in middle, slower at start/end)
		baseDelay := 20 + rand.Intn(30) // 20-50ms base

		// Slower at start and end
		speedFactor := math.Sin(math.Pi * t)
		delay := int(float64(baseDelay) / (0.5 + speedFactor*0.5))

		steps[i] = scrollStep{
			position: position,
			delayMs:  delay,
		}
	}

	return steps
}

// applyScrollCorrection applies a small correction (overshoot and back)
func (s *ScrollController) applyScrollCorrection(page *rod.Page, targetY int) {
	// Overshoot by 20-50 pixels
	overshoot := 20 + rand.Intn(30)
	if rand.Float64() < 0.5 {
		overshoot = -overshoot
	}

	// Quick scroll past target
	page.Eval(`window.scrollTo(0, ` + itoa(targetY+overshoot) + `)`)
	time.Sleep(time.Duration(100+rand.Intn(100)) * time.Millisecond)

	// Correct back
	page.Eval(`window.scrollTo(0, ` + itoa(targetY) + `)`)
	time.Sleep(time.Duration(50+rand.Intn(50)) * time.Millisecond)

	s.logger.Debug().Msg("Applied scroll correction")
}

// getScrollPosition gets current vertical scroll position
func (s *ScrollController) getScrollPosition(page *rod.Page) (int, error) {
	result, err := page.Eval(`window.pageYOffset || document.documentElement.scrollTop`)
	if err != nil {
		return 0, err
	}
	return int(result.Value.Num()), nil
}

// getPageHeight gets total page height
func (s *ScrollController) getPageHeight(page *rod.Page) (int, error) {
	result, err := page.Eval(`Math.max(document.body.scrollHeight, document.documentElement.scrollHeight)`)
	if err != nil {
		return 0, err
	}
	return int(result.Value.Num()), nil
}

// getViewportHeight gets viewport height
func (s *ScrollController) getViewportHeight(page *rod.Page) (int, error) {
	result, err := page.Eval(`window.innerHeight`)
	if err != nil {
		return 0, err
	}
	return int(result.Value.Num()), nil
}
