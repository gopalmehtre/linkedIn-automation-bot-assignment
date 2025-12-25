// Package auth handles LinkedIn authentication
package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/models"
	"linkedin-automation/internal/stealth"
)

// LinkedIn URLs
const (
	LinkedInLoginURL = "https://www.linkedin.com/login"
	LinkedInFeedURL  = "https://www.linkedin.com/feed/"
	LinkedInHomeURL  = "https://www.linkedin.com/"
)

// Selectors for login page
const (
	SelectorEmailInput    = "input#username"
	SelectorPasswordInput = "input#password"
	SelectorLoginButton   = "button[type='submit']"
	SelectorFeedContainer = ".feed-shared-update-v2"
	SelectorNavProfile    = ".global-nav__me"
	SelectorErrorMessage  = "#error-for-username, #error-for-password, .form__label--error"
)

// Authenticator handles LinkedIn login flow
type Authenticator struct {
	browser        *browser.Browser
	sessionManager *browser.SessionManager
	pageHelper     *browser.PageHelper
	stealth        *stealth.Controller
	logger         zerolog.Logger
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(
	b *browser.Browser,
	sessionMgr *browser.SessionManager,
	stealth *stealth.Controller,
	logger zerolog.Logger,
) *Authenticator {
	return &Authenticator{
		browser:        b,
		sessionManager: sessionMgr,
		pageHelper:     browser.NewPageHelper(logger),
		stealth:        stealth,
		logger:         logger.With().Str("component", "auth").Logger(),
	}
}

// Login performs the LinkedIn login flow
func (a *Authenticator) Login(email, password string) (*models.LoginResult, error) {
	a.logger.Info().Msg("Starting login flow")

	// Try to reuse existing session first
	if a.sessionManager.HasSavedSession() && a.sessionManager.IsSessionValid() {
		a.logger.Info().Msg("Found existing session, attempting to reuse")

		// Load cookies
		if err := a.sessionManager.LoadCookies(a.browser.Browser()); err != nil {
			a.logger.Warn().Err(err).Msg("Failed to load cookies")
		} else {
			// Check if session is still valid
			isValid, err := a.VerifySession()
			if err == nil && isValid {
				a.logger.Info().Msg("Existing session is valid")
				return &models.LoginResult{
					Success:      true,
					SessionSaved: true,
				}, nil
			}
			a.logger.Info().Msg("Existing session expired, proceeding with fresh login")
		}
	}

	// Get or create page
	page, err := a.browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Navigate to login page
	a.logger.Debug().Msg("Navigating to login page")
	if err := a.browser.Navigate(page, LinkedInLoginURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to login: %w", err)
	}

	// Wait for login form
	a.stealth.Timing().PageLoadDelay()

	// Check if already logged in (redirected to feed)
	currentURL := a.pageHelper.GetCurrentURL(page)
	if strings.Contains(currentURL, "/feed") {
		a.logger.Info().Msg("Already logged in (redirected to feed)")
		a.sessionManager.SaveCookies(a.browser.Browser())
		return &models.LoginResult{
			Success:      true,
			SessionSaved: true,
		}, nil
	}

	// Enter email
	a.logger.Debug().Msg("Entering email")
	emailInput, err := a.pageHelper.WaitForElement(page, SelectorEmailInput, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("email input not found: %w", err)
	}

	// Simulate human-like reading of the page
	a.stealth.SimulateReading(page)

	// Click on email field
	if err := a.stealth.Mouse().ClickElement(page, emailInput); err != nil {
		return nil, fmt.Errorf("failed to click email input: %w", err)
	}

	// Type email with human-like patterns
	if err := a.stealth.Typing().TypeText(emailInput, email); err != nil {
		return nil, fmt.Errorf("failed to type email: %w", err)
	}

	a.stealth.Timing().ActionDelay()

	// Enter password
	a.logger.Debug().Msg("Entering password")
	passwordInput, err := a.pageHelper.WaitForElement(page, SelectorPasswordInput, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("password input not found: %w", err)
	}

	// Click on password field
	if err := a.stealth.Mouse().ClickElement(page, passwordInput); err != nil {
		return nil, fmt.Errorf("failed to click password input: %w", err)
	}

	// Type password
	if err := a.stealth.Typing().TypeText(passwordInput, password); err != nil {
		return nil, fmt.Errorf("failed to type password: %w", err)
	}

	a.stealth.Timing().ActionDelay()

	// Click login button
	a.logger.Debug().Msg("Clicking login button")
	loginButton, err := a.pageHelper.WaitForElement(page, SelectorLoginButton, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("login button not found: %w", err)
	}

	// Human-like click on login button
	if err := a.stealth.Mouse().ClickElement(page, loginButton); err != nil {
		return nil, fmt.Errorf("failed to click login button: %w", err)
	}

	// Wait for navigation
	a.logger.Debug().Msg("Waiting for login result")
	time.Sleep(3 * time.Second)

	// Check for checkpoint/verification
	checkpointType := a.DetectCheckpoint(page)
	if checkpointType != models.CheckpointNone {
		a.logger.Warn().
			Str("type", string(checkpointType)).
			Msg("Security checkpoint detected")

		return &models.LoginResult{
			Success:        false,
			CheckpointType: checkpointType,
			ErrorMessage:   "Security checkpoint requires manual intervention",
		}, nil
	}

	// Check for login errors
	if a.pageHelper.ElementExists(page, SelectorErrorMessage) {
		errorEl, _ := page.Element(SelectorErrorMessage)
		errorMsg := a.pageHelper.GetElementText(errorEl)
		a.logger.Error().Str("error", errorMsg).Msg("Login error")

		return &models.LoginResult{
			Success:      false,
			ErrorMessage: errorMsg,
		}, nil
	}

	// Verify successful login
	currentURL = a.pageHelper.GetCurrentURL(page)
	if strings.Contains(currentURL, "/feed") || strings.Contains(currentURL, "/mynetwork") {
		a.logger.Info().Msg("Login successful")

		// Save session cookies
		if err := a.sessionManager.SaveCookies(a.browser.Browser()); err != nil {
			a.logger.Warn().Err(err).Msg("Failed to save cookies")
		}

		return &models.LoginResult{
			Success:      true,
			SessionSaved: true,
		}, nil
	}

	// Check for other issues
	if strings.Contains(currentURL, "checkpoint") {
		return &models.LoginResult{
			Success:        false,
			CheckpointType: models.CheckpointUnknown,
			ErrorMessage:   "Checkpoint page detected",
		}, nil
	}

	return &models.LoginResult{
		Success:      false,
		ErrorMessage: "Login failed for unknown reason",
	}, nil
}

// VerifySession checks if the current session is still valid
func (a *Authenticator) VerifySession() (bool, error) {
	a.logger.Debug().Msg("Verifying session")

	page, err := a.browser.NewPage()
	if err != nil {
		return false, err
	}

	// Navigate to LinkedIn feed
	if err := a.browser.Navigate(page, LinkedInFeedURL); err != nil {
		return false, err
	}

	// Wait a moment
	time.Sleep(2 * time.Second)

	// Check if we're on the feed or redirected to login
	currentURL := a.pageHelper.GetCurrentURL(page)

	if strings.Contains(currentURL, "/login") {
		a.logger.Debug().Msg("Session invalid - redirected to login")
		return false, nil
	}

	if strings.Contains(currentURL, "/feed") {
		// Additional check - look for profile nav element
		if a.pageHelper.ElementVisible(page, SelectorNavProfile) {
			a.logger.Debug().Msg("Session valid - profile nav found")
			return true, nil
		}
	}

	// Check for checkpoint
	checkpoint := a.DetectCheckpoint(page)
	if checkpoint != models.CheckpointNone {
		a.logger.Debug().Str("checkpoint", string(checkpoint)).Msg("Session invalid - checkpoint detected")
		return false, nil
	}

	return false, nil
}

// Logout performs logout (clears session)
func (a *Authenticator) Logout() error {
	a.logger.Info().Msg("Logging out")

	// Clear saved cookies
	if err := a.sessionManager.ClearCookies(); err != nil {
		return err
	}

	// Clear browser cookies
	a.browser.Browser().SetCookies(nil)

	return nil
}
