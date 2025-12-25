// Package connection handles sending LinkedIn connection requests
package connection

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/config"
	"linkedin-automation/internal/models"
	"linkedin-automation/internal/stealth"
	"linkedin-automation/internal/storage"
)

// Profile page selectors
const (
	SelectorConnectButton       = "button[aria-label*='Connect'], button.pvs-profile-actions__action"
	SelectorConnectButtonAlt    = "button:has-text('Connect')"
	SelectorMoreButton          = "button[aria-label='More actions']"
	SelectorDropdownConnect     = "[data-control-name='connect'], div[aria-label*='Connect']"
	SelectorAddNoteButton       = "button[aria-label='Add a note']"
	SelectorNoteTextarea        = "textarea[name='message'], textarea#custom-message"
	SelectorSendButton          = "button[aria-label='Send now'], button[aria-label='Send invitation']"
	SelectorSendButtonAlt       = "button:has-text('Send')"
	SelectorDismissButton       = "button[aria-label='Dismiss']"
	SelectorPendingButton       = "button[aria-label*='Pending']"
	SelectorMessageButton       = "button[aria-label*='Message']"
	SelectorProfileName         = "h1.text-heading-xlarge"
	SelectorProfileTitle        = ".text-body-medium"
	SelectorProfileCompany      = ".pv-text-details__right-panel-item-text"
	SelectorConnectionModal     = ".send-invite"
)

// ConnectionManager handles connection requests
type ConnectionManager struct {
	browser         *browser.Browser
	pageHelper      *browser.PageHelper
	stealth         *stealth.Controller
	profileStore    *storage.ProfileStore
	connectionStore *storage.ConnectionStore
	statsStore      *storage.StatsStore
	config          *config.LimitsConfig
	stealthConfig   *config.StealthConfig
	logger          zerolog.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(
	b *browser.Browser,
	profileStore *storage.ProfileStore,
	connectionStore *storage.ConnectionStore,
	statsStore *storage.StatsStore,
	limitsConfig *config.LimitsConfig,
	stealthConfig *config.StealthConfig,
	stealthCtrl *stealth.Controller,
	logger zerolog.Logger,
) *ConnectionManager {
	return &ConnectionManager{
		browser:         b,
		pageHelper:      browser.NewPageHelper(logger),
		stealth:         stealthCtrl,
		profileStore:    profileStore,
		connectionStore: connectionStore,
		statsStore:      statsStore,
		config:          limitsConfig,
		stealthConfig:   stealthConfig,
		logger:          logger.With().Str("component", "connection").Logger(),
	}
}

// SendConnectionRequest sends a connection request to a profile
func (c *ConnectionManager) SendConnectionRequest(profile *models.Profile, noteTemplate string) error {
	c.logger.Info().
		Str("url", profile.URL).
		Str("name", profile.FullName).
		Msg("Sending connection request")

	// Get page
	page, err := c.browser.GetPage()
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Navigate to profile
	if err := c.browser.Navigate(page, profile.URL); err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page and simulate reading
	c.stealth.Timing().PageLoadDelay()

	// Update profile info from page
	c.updateProfileFromPage(page, profile)

	// Simulate browsing the profile
	stealth.SimulateProfileBrowsing(page, c.stealth.Mouse(), c.stealth.Scroll(), c.stealth.Timing(), c.logger)

	// Check if already connected or pending
	if c.isAlreadyConnected(page) {
		c.logger.Info().Msg("Already connected or pending")
		c.profileStore.UpdateStatus(profile.ID, models.ProfileStatusConnected)
		return nil
	}

	// Find and click Connect button
	if err := c.clickConnectButton(page); err != nil {
		return fmt.Errorf("failed to click connect button: %w", err)
	}

	// Wait for modal
	time.Sleep(time.Second)

	// Add personalized note if template provided
	if noteTemplate != "" {
		if err := c.addNote(page, profile, noteTemplate); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to add note, sending without note")
		}
	}

	// Click Send
	if err := c.clickSendButton(page); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Record the connection request
	_, err = c.connectionStore.RecordRequest(profile.ID, noteTemplate)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to record connection request")
	}

	// Update profile status
	c.profileStore.UpdateStatus(profile.ID, models.ProfileStatusRequested)

	// Update stats
	c.statsStore.IncrementConnections()

	c.logger.Info().
		Str("name", profile.FullName).
		Msg("Connection request sent successfully")

	return nil
}

// ProcessPendingProfiles processes all pending profiles
func (c *ConnectionManager) ProcessPendingProfiles(noteTemplate string, limit int) (int, error) {
	c.logger.Info().Int("limit", limit).Msg("Processing pending profiles")

	// Check daily limit
	canSend, remaining, err := c.statsStore.CanSendConnection(c.config.DailyConnections)
	if err != nil {
		return 0, fmt.Errorf("failed to check daily limit: %w", err)
	}

	if !canSend {
		c.logger.Info().Msg("Daily connection limit reached")
		return 0, nil
	}

	// Adjust limit based on remaining quota
	if limit > remaining {
		limit = remaining
	}

	// Get pending profiles
	profiles, err := c.profileStore.GetPending(limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending profiles: %w", err)
	}

	if len(profiles) == 0 {
		c.logger.Info().Msg("No pending profiles to process")
		return 0, nil
	}

	sentCount := 0

	for i, profile := range profiles {
		// Check business hours
		if !c.stealth.IsWithinSchedule() {
			c.logger.Info().Msg("Outside business hours, stopping")
			break
		}

		// Check hourly limit
		hourlyCount, err := c.connectionStore.GetHourCount()
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to get hourly count")
		} else if hourlyCount >= c.config.ConnectionsPerHour {
			c.logger.Info().Msg("Hourly connection limit reached, waiting")
			// Wait until next hour
			time.Sleep(time.Hour)
		}

		// Check for break
		if c.stealth.ShouldTakeBreak(time.Now(), sentCount) {
			c.stealth.Scheduler().TakeBreak()
		}

		// Send request
		err = c.SendConnectionRequest(profile, noteTemplate)
		if err != nil {
			c.logger.Error().
				Err(err).
				Str("profile", profile.URL).
				Msg("Failed to send connection request")

			// Mark as skipped on persistent error
			c.profileStore.UpdateStatus(profile.ID, models.ProfileStatusSkipped)
			continue
		}

		sentCount++

		// Delay between requests (if not last)
		if i < len(profiles)-1 {
			delay := c.config.MinDelaySeconds + (c.config.MaxDelaySeconds-c.config.MinDelaySeconds)/2
			c.stealth.Timing().RandomDelay(c.config.MinDelaySeconds, delay)
		}
	}

	c.logger.Info().
		Int("sent", sentCount).
		Int("total", len(profiles)).
		Msg("Finished processing profiles")

	return sentCount, nil
}

// clickConnectButton finds and clicks the Connect button
func (c *ConnectionManager) clickConnectButton(page *rod.Page) error {
	c.logger.Debug().Msg("Looking for Connect button")

	// Try primary selector
	connectBtn, err := c.pageHelper.WaitForElement(page, SelectorConnectButton, 5*time.Second)
	if err == nil {
		// Scroll into view
		c.stealth.Scroll().ScrollIntoView(page, connectBtn)
		c.stealth.Timing().ActionDelay()

		// Click with human-like behavior
		return c.stealth.Mouse().ClickElement(page, connectBtn)
	}

	// Try More button dropdown
	c.logger.Debug().Msg("Connect button not found directly, trying More dropdown")

	moreBtn, err := c.pageHelper.WaitForElement(page, SelectorMoreButton, 3*time.Second)
	if err == nil {
		// Click More button
		c.stealth.Mouse().ClickElement(page, moreBtn)
		time.Sleep(500 * time.Millisecond)

		// Find Connect in dropdown
		dropdownConnect, err := c.pageHelper.WaitForElement(page, SelectorDropdownConnect, 3*time.Second)
		if err == nil {
			c.stealth.Timing().ShortDelay()
			return c.stealth.Mouse().ClickElement(page, dropdownConnect)
		}
	}

	// Try finding by text content
	c.logger.Debug().Msg("Trying to find Connect button by text")
	if err := c.pageHelper.ClickElementByText(page, "button", "Connect"); err == nil {
		return nil
	}

	return fmt.Errorf("connect button not found")
}

// addNote adds a personalized note to the connection request
func (c *ConnectionManager) addNote(page *rod.Page, profile *models.Profile, template string) error {
	c.logger.Debug().Msg("Adding personalized note")

	// Check if "Add a note" button exists
	addNoteBtn, err := page.Element(SelectorAddNoteButton)
	if err != nil {
		c.logger.Debug().Msg("Add note button not found, proceeding without note")
		return nil
	}

	// Click Add note
	c.stealth.Mouse().ClickElement(page, addNoteBtn)
	time.Sleep(500 * time.Millisecond)

	// Find textarea
	textarea, err := c.pageHelper.WaitForElement(page, SelectorNoteTextarea, 3*time.Second)
	if err != nil {
		return fmt.Errorf("note textarea not found: %w", err)
	}

	// Generate personalized note
	note := RenderNoteTemplate(template, profile)

	// Truncate to LinkedIn's 300 character limit
	if len(note) > 300 {
		note = note[:297] + "..."
	}

	// Type note with human-like behavior
	c.stealth.Mouse().ClickElement(page, textarea)
	c.stealth.Timing().ShortDelay()

	if err := c.stealth.Typing().TypeWithPauses(textarea, note); err != nil {
		return fmt.Errorf("failed to type note: %w", err)
	}

	return nil
}

// clickSendButton clicks the Send button
func (c *ConnectionManager) clickSendButton(page *rod.Page) error {
	c.logger.Debug().Msg("Clicking Send button")

	// Wait a moment for the button to be ready
	c.stealth.Timing().ShortDelay()

	// Try primary selector
	sendBtn, err := c.pageHelper.WaitForElement(page, SelectorSendButton, 5*time.Second)
	if err == nil {
		return c.stealth.Mouse().ClickElement(page, sendBtn)
	}

	// Try alternative selector
	sendBtn, err = page.Element(SelectorSendButtonAlt)
	if err == nil {
		return c.stealth.Mouse().ClickElement(page, sendBtn)
	}

	// Try finding by text
	return c.pageHelper.ClickElementByText(page, "button", "Send")
}

// isAlreadyConnected checks if already connected or request pending
func (c *ConnectionManager) isAlreadyConnected(page *rod.Page) bool {
	// Check for Pending button
	if c.pageHelper.ElementExists(page, SelectorPendingButton) {
		return true
	}

	// Check for Message button (indicates already connected)
	if c.pageHelper.ElementExists(page, SelectorMessageButton) {
		// But also check there's no Connect button
		if !c.pageHelper.ElementExists(page, SelectorConnectButton) {
			return true
		}
	}

	return false
}

// updateProfileFromPage updates profile info from the page
func (c *ConnectionManager) updateProfileFromPage(page *rod.Page, profile *models.Profile) {
	// Get name
	nameEl, err := page.Element(SelectorProfileName)
	if err == nil {
		profile.FullName = strings.TrimSpace(c.pageHelper.GetElementText(nameEl))
		parts := strings.SplitN(profile.FullName, " ", 2)
		if len(parts) > 0 {
			profile.FirstName = parts[0]
		}
		if len(parts) > 1 {
			profile.LastName = parts[1]
		}
	}

	// Get title
	titleEl, err := page.Element(SelectorProfileTitle)
	if err == nil {
		profile.Title = strings.TrimSpace(c.pageHelper.GetElementText(titleEl))
	}

	// Get company
	companyEl, err := page.Element(SelectorProfileCompany)
	if err == nil {
		profile.Company = strings.TrimSpace(c.pageHelper.GetElementText(companyEl))
	}

	// Save updated profile
	c.profileStore.Save(profile)
}

// RenderNoteTemplate renders a connection note template with profile data
func RenderNoteTemplate(template string, profile *models.Profile) string {
	data := models.NewTemplateData(profile)

	note := template
	note = strings.ReplaceAll(note, "{{.FirstName}}", data.FirstName)
	note = strings.ReplaceAll(note, "{{.LastName}}", data.LastName)
	note = strings.ReplaceAll(note, "{{.FullName}}", data.FullName)
	note = strings.ReplaceAll(note, "{{.Company}}", data.Company)
	note = strings.ReplaceAll(note, "{{.Title}}", data.Title)
	note = strings.ReplaceAll(note, "{{.Location}}", data.Location)

	// Clean up empty placeholders
	note = strings.ReplaceAll(note, "at  ", "")
	note = strings.ReplaceAll(note, "in  ", "")
	note = strings.ReplaceAll(note, "  ", " ")

	return strings.TrimSpace(note)
}
