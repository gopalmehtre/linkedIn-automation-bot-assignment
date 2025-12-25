// Package stealth - mouse hovering behavior
// This is additional stealth technique #3
package stealth

import (
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"
)

// HoverRandomElement hovers over a random visible element on the page
func HoverRandomElement(page *rod.Page, mouse *MouseController, timing *TimingController, logger zerolog.Logger) error {
	logger.Debug().Msg("Performing random hover")

	// Find hoverable elements (links, buttons, images)
	elements, err := page.Elements("a, button, img, [role='button'], .hoverable")
	if err != nil {
		return err
	}

	if len(elements) == 0 {
		return nil
	}

	// Pick a random element
	element := elements[rand.Intn(len(elements))]

	// Check if element is visible
	visible, err := element.Visible()
	if err != nil || !visible {
		return nil
	}

	// Move to element
	if err := mouse.MoveToElement(page, element); err != nil {
		return err
	}

	// Hover duration (200-800ms)
	hoverTime := time.Duration(200+rand.Intn(600)) * time.Millisecond
	time.Sleep(hoverTime)

	logger.Debug().
		Dur("duration", hoverTime).
		Msg("Hovered random element")

	return nil
}

// HoverElement hovers over a specific element
func HoverElement(page *rod.Page, element *rod.Element, mouse *MouseController, timing *TimingController, logger zerolog.Logger) error {
	// Move to element
	if err := mouse.MoveToElement(page, element); err != nil {
		return err
	}

	// Random hover duration
	hoverTime := time.Duration(300+rand.Intn(500)) * time.Millisecond
	time.Sleep(hoverTime)

	return nil
}

// HoverAndRead simulates hovering over an element while "reading" its content
func HoverAndRead(page *rod.Page, element *rod.Element, mouse *MouseController, timing *TimingController, logger zerolog.Logger) error {
	logger.Debug().Msg("Hover and read simulation")

	// Move to element
	if err := mouse.MoveToElement(page, element); err != nil {
		return err
	}

	// Get element text length to estimate reading time
	text, err := element.Text()
	if err != nil {
		text = ""
	}

	// Calculate reading time (average 200-250 WPM, ~5 chars per word)
	wordCount := len(text) / 5
	if wordCount < 2 {
		wordCount = 2
	}
	if wordCount > 50 {
		wordCount = 50 // Cap at 50 words worth of reading
	}

	// Reading speed: 200-250 WPM = ~3.3-4.2 words per second
	readingTime := time.Duration(float64(wordCount)/3.5*1000) * time.Millisecond

	// Add some variance
	readingTime = time.Duration(float64(readingTime) * (0.8 + rand.Float64()*0.4))

	time.Sleep(readingTime)

	return nil
}

// SimulateProfileBrowsing simulates natural browsing behavior on a profile page
func SimulateProfileBrowsing(page *rod.Page, mouse *MouseController, scroll *ScrollController, timing *TimingController, logger zerolog.Logger) error {
	logger.Debug().Msg("Simulating profile browsing behavior")

	// Initial pause to "take in" the profile
	timing.ThinkDelay()

	// Random number of actions (3-7)
	numActions := 3 + rand.Intn(5)

	for i := 0; i < numActions; i++ {
		action := rand.Intn(4)

		switch action {
		case 0:
			// Scroll down
			if err := scroll.ScrollDown(page); err != nil {
				logger.Warn().Err(err).Msg("Scroll down failed")
			}
			timing.ActionDelay()

		case 1:
			// Scroll up (less frequently)
			if rand.Float64() < 0.3 {
				if err := scroll.ScrollUp(page); err != nil {
					logger.Warn().Err(err).Msg("Scroll up failed")
				}
				timing.ActionDelay()
			}

		case 2:
			// Hover over a random element
			if err := HoverRandomElement(page, mouse, timing, logger); err != nil {
				logger.Warn().Err(err).Msg("Random hover failed")
			}

		case 3:
			// Just pause (thinking/reading)
			timing.ThinkDelay()
		}
	}

	return nil
}

// WanderMouse moves the mouse randomly without clicking (simulates idle movement)
func WanderMouse(page *rod.Page, mouse *MouseController, logger zerolog.Logger) error {
	logger.Debug().Msg("Mouse wandering")

	// Get viewport size
	viewportWidth := 1920  // Default, could be fetched from page
	viewportHeight := 1080

	// Generate random target within viewport
	targetX := float64(100 + rand.Intn(viewportWidth-200))
	targetY := float64(100 + rand.Intn(viewportHeight-200))

	// Move to random position
	return mouse.MoveTo(page, targetX, targetY)
}
