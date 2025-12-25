// Package messaging handles LinkedIn messaging
package messaging

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/config"
	"linkedin-automation/internal/models"
	"linkedin-automation/internal/stealth"
	"linkedin-automation/internal/storage"
)

// Messaging selectors
const (
	SelectorMessageButton     = "button[aria-label*='Message'], a[href*='/messaging/']"
	SelectorMessageModal      = ".msg-overlay-conversation-bubble"
	SelectorMessageInput      = ".msg-form__contenteditable, div[role='textbox']"
	SelectorSendMessageButton = "button[type='submit'].msg-form__send-button, button.msg-form__send-button"
	SelectorCloseChat         = "button[aria-label*='Close']"
	SelectorConnectionsList   = ".mn-connections"
	SelectorConnectionItem    = ".mn-connection-card"
	SelectorConnectionName    = ".mn-connection-card__name"
	SelectorConnectionLink    = ".mn-connection-card__link"
)

// LinkedIn URLs
const (
	LinkedInConnectionsURL = "https://www.linkedin.com/mynetwork/invite-connect/connections/"
)

// Messenger handles LinkedIn messaging
type Messenger struct {
	browser         *browser.Browser
	pageHelper      *browser.PageHelper
	stealth         *stealth.Controller
	profileStore    *storage.ProfileStore
	connectionStore *storage.ConnectionStore
	messageStore    *storage.MessageStore
	statsStore      *storage.StatsStore
	config          *config.LimitsConfig
	logger          zerolog.Logger
}

// NewMessenger creates a new messenger
func NewMessenger(
	b *browser.Browser,
	profileStore *storage.ProfileStore,
	connectionStore *storage.ConnectionStore,
	messageStore *storage.MessageStore,
	statsStore *storage.StatsStore,
	limitsConfig *config.LimitsConfig,
	stealthCtrl *stealth.Controller,
	logger zerolog.Logger,
) *Messenger {
	return &Messenger{
		browser:         b,
		pageHelper:      browser.NewPageHelper(logger),
		stealth:         stealthCtrl,
		profileStore:    profileStore,
		connectionStore: connectionStore,
		messageStore:    messageStore,
		statsStore:      statsStore,
		config:          limitsConfig,
		logger:          logger.With().Str("component", "messenger").Logger(),
	}
}

// SendMessage sends a message to a profile
func (m *Messenger) SendMessage(profile *models.Profile, messageText string) error {
	m.logger.Info().
		Str("url", profile.URL).
		Str("name", profile.FullName).
		Msg("Sending message")

	// Get page
	page, err := m.browser.GetPage()
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Navigate to profile
	if err := m.browser.Navigate(page, profile.URL); err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	m.stealth.Timing().PageLoadDelay()

	// Simulate reading profile
	m.stealth.SimulateReading(page)

	// Find and click Message button
	msgBtn, err := m.pageHelper.WaitForElement(page, SelectorMessageButton, 10*time.Second)
	if err != nil {
		return fmt.Errorf("message button not found: %w", err)
	}

	m.stealth.Scroll().ScrollIntoView(page, msgBtn)
	m.stealth.Timing().ActionDelay()

	if err := m.stealth.Mouse().ClickElement(page, msgBtn); err != nil {
		return fmt.Errorf("failed to click message button: %w", err)
	}

	// Wait for message modal
	time.Sleep(2 * time.Second)

	// Find message input
	msgInput, err := m.pageHelper.WaitForElement(page, SelectorMessageInput, 5*time.Second)
	if err != nil {
		return fmt.Errorf("message input not found: %w", err)
	}

	// Render message template
	renderedMsg := RenderMessageTemplate(messageText, profile)

	// Type message with human-like behavior
	m.stealth.Mouse().ClickElement(page, msgInput)
	m.stealth.Timing().ActionDelay()

	if err := m.stealth.Typing().TypeWithPauses(msgInput, renderedMsg); err != nil {
		return fmt.Errorf("failed to type message: %w", err)
	}

	m.stealth.Timing().ActionDelay()

	// Click send
	sendBtn, err := m.pageHelper.WaitForElement(page, SelectorSendMessageButton, 3*time.Second)
	if err != nil {
		return fmt.Errorf("send button not found: %w", err)
	}

	if err := m.stealth.Mouse().ClickElement(page, sendBtn); err != nil {
		return fmt.Errorf("failed to click send: %w", err)
	}

	// Wait for message to send
	time.Sleep(time.Second)

	// Close chat modal
	closeBtn, err := page.Element(SelectorCloseChat)
	if err == nil {
		m.stealth.Mouse().ClickElement(page, closeBtn)
	}

	// Record the message
	_, err = m.messageStore.RecordMessage(profile.ID, renderedMsg, models.MessageTypeFollowup)
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to record message")
	}

	// Update stats
	m.statsStore.IncrementMessages()

	m.logger.Info().
		Str("name", profile.FullName).
		Msg("Message sent successfully")

	return nil
}

// DetectNewConnections checks for newly accepted connections
func (m *Messenger) DetectNewConnections() ([]*models.Profile, error) {
	m.logger.Info().Msg("Detecting new connections")

	// Get pending connection requests from database
	pendingRequests, err := m.connectionStore.GetPending()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}

	if len(pendingRequests) == 0 {
		m.logger.Debug().Msg("No pending requests to check")
		return nil, nil
	}

	// Get page
	page, err := m.browser.GetPage()
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	var newConnections []*models.Profile

	// Check each pending profile
	for _, req := range pendingRequests {
		profile, err := m.profileStore.GetByID(req.ProfileID)
		if err != nil || profile == nil {
			continue
		}

		// Navigate to profile
		if err := m.browser.Navigate(page, profile.URL); err != nil {
			m.logger.Warn().Err(err).Str("url", profile.URL).Msg("Failed to navigate")
			continue
		}

		m.stealth.Timing().PageLoadDelay()

		// Check if Message button is visible (indicates connected)
		if m.pageHelper.ElementVisible(page, SelectorMessageButton) {
			// Check if Pending button is NOT visible
			if !m.pageHelper.ElementVisible(page, "button[aria-label*='Pending']") {
				m.logger.Info().
					Str("name", profile.FullName).
					Msg("New connection detected")

				// Update status
				m.connectionStore.MarkAccepted(profile.ID)
				m.profileStore.UpdateStatus(profile.ID, models.ProfileStatusConnected)

				newConnections = append(newConnections, profile)
			}
		}

		// Delay between checks
		m.stealth.Timing().RandomDelay(2, 5)
	}

	m.logger.Info().
		Int("count", len(newConnections)).
		Msg("New connections detected")

	return newConnections, nil
}

// SendFollowups sends follow-up messages to unmessaged connections
func (m *Messenger) SendFollowups(messageTemplate string, limit int) (int, error) {
	m.logger.Info().Int("limit", limit).Msg("Sending follow-up messages")

	// Check daily limit
	canSend, remaining, err := m.statsStore.CanSendMessage(m.config.DailyMessages)
	if err != nil {
		return 0, fmt.Errorf("failed to check daily limit: %w", err)
	}

	if !canSend {
		m.logger.Info().Msg("Daily message limit reached")
		return 0, nil
	}

	if limit > remaining {
		limit = remaining
	}

	// Get unmessaged connections using the profile store's database
	profiles, err := m.profileStore.GetConnectedWithoutFollowup(limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get unmessaged connections: %w", err)
	}

	if len(profiles) == 0 {
		m.logger.Info().Msg("No unmessaged connections")
		return 0, nil
	}

	if len(profiles) > limit {
		profiles = profiles[:limit]
	}

	sentCount := 0

	for i, profile := range profiles {
		// Check schedule
		if !m.stealth.IsWithinSchedule() {
			m.logger.Info().Msg("Outside business hours, stopping")
			break
		}

		// Check hourly limit
		hourlyCount, _ := m.messageStore.GetHourCount()
		if hourlyCount >= m.config.MessagesPerHour {
			m.logger.Info().Msg("Hourly message limit reached")
			break
		}

		// Send message
		err := m.SendMessage(profile, messageTemplate)
		if err != nil {
			m.logger.Error().
				Err(err).
				Str("profile", profile.URL).
				Msg("Failed to send message")
			continue
		}

		sentCount++

		// Delay between messages
		if i < len(profiles)-1 {
			m.stealth.Timing().RandomDelay(m.config.MinDelaySeconds, m.config.MaxDelaySeconds)
		}
	}

	m.logger.Info().
		Int("sent", sentCount).
		Msg("Follow-up messages sent")

	return sentCount, nil
}

// ProcessFollowups detects new connections and sends follow-ups
func (m *Messenger) ProcessFollowups(messageTemplate string, limit int) (int, int, error) {
	// Detect new connections
	newConnections, err := m.DetectNewConnections()
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to detect new connections")
	}

	// Send follow-ups
	sentCount, err := m.SendFollowups(messageTemplate, limit)
	if err != nil {
		return len(newConnections), 0, err
	}

	return len(newConnections), sentCount, nil
}

// RenderMessageTemplate renders a message template with profile data
func RenderMessageTemplate(template string, profile *models.Profile) string {
	data := models.NewTemplateData(profile)

	msg := template
	msg = strings.ReplaceAll(msg, "{{.FirstName}}", data.FirstName)
	msg = strings.ReplaceAll(msg, "{{.LastName}}", data.LastName)
	msg = strings.ReplaceAll(msg, "{{.FullName}}", data.FullName)
	msg = strings.ReplaceAll(msg, "{{.Company}}", data.Company)
	msg = strings.ReplaceAll(msg, "{{.Title}}", data.Title)
	msg = strings.ReplaceAll(msg, "{{.Location}}", data.Location)

	// Clean up empty placeholders
	msg = strings.ReplaceAll(msg, "at  ", "")
	msg = strings.ReplaceAll(msg, "in  ", "")
	msg = strings.ReplaceAll(msg, "  ", " ")

	return strings.TrimSpace(msg)
}
