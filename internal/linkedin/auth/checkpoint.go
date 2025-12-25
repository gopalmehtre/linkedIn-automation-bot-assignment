// Package auth - checkpoint/security challenge detection
package auth

import (
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/models"
)

// Checkpoint selectors
const (
	Selector2FAInput         = "input[name='pin']"
	SelectorCaptchaFrame     = "iframe[src*='captcha'], iframe[src*='recaptcha']"
	SelectorPhoneVerify      = "input[name='phoneNumber'], #phone-number-input"
	SelectorEmailVerify      = "input[name='email'], .email-verification"
	SelectorUnusualActivity  = ".unusual-activity, [data-test='unusual-activity']"
	SelectorChallenge        = ".challenge, [data-test='challenge']"
	SelectorSecurityCheck    = ".security-verification, #captcha-internal"
)

// CheckpointDetector detects security checkpoints
type CheckpointDetector struct {
	pageHelper *browser.PageHelper
	logger     zerolog.Logger
}

// NewCheckpointDetector creates a new checkpoint detector
func NewCheckpointDetector(logger zerolog.Logger) *CheckpointDetector {
	return &CheckpointDetector{
		pageHelper: browser.NewPageHelper(logger),
		logger:     logger.With().Str("component", "checkpoint").Logger(),
	}
}

// DetectCheckpoint checks the current page for security checkpoints
func (a *Authenticator) DetectCheckpoint(page *rod.Page) models.CheckpointType {
	a.logger.Debug().Msg("Checking for security checkpoints")

	currentURL := a.pageHelper.GetCurrentURL(page)

	// Check URL patterns
	if strings.Contains(currentURL, "checkpoint") {
		// Determine type based on page content
		if a.pageHelper.ElementExists(page, Selector2FAInput) {
			return models.CheckpointTwoFactor
		}
		if a.pageHelper.ElementExists(page, SelectorCaptchaFrame) {
			return models.CheckpointCaptcha
		}
		if a.pageHelper.ElementExists(page, SelectorPhoneVerify) {
			return models.CheckpointPhoneVerify
		}
		if a.pageHelper.ElementExists(page, SelectorEmailVerify) {
			return models.CheckpointEmailVerify
		}
		return models.CheckpointUnknown
	}

	// Check for 2FA
	if a.pageHelper.ElementExists(page, Selector2FAInput) {
		a.logger.Debug().Msg("2FA checkpoint detected")
		return models.CheckpointTwoFactor
	}

	// Check for CAPTCHA
	if a.pageHelper.ElementExists(page, SelectorCaptchaFrame) {
		a.logger.Debug().Msg("CAPTCHA checkpoint detected")
		return models.CheckpointCaptcha
	}

	// Check for phone verification
	if a.pageHelper.ElementExists(page, SelectorPhoneVerify) {
		a.logger.Debug().Msg("Phone verification checkpoint detected")
		return models.CheckpointPhoneVerify
	}

	// Check for email verification
	if a.pageHelper.ElementExists(page, SelectorEmailVerify) {
		a.logger.Debug().Msg("Email verification checkpoint detected")
		return models.CheckpointEmailVerify
	}

	// Check for unusual activity warning
	if a.pageHelper.ElementExists(page, SelectorUnusualActivity) {
		a.logger.Debug().Msg("Unusual activity checkpoint detected")
		return models.CheckpointUnusualActivity
	}

	// Check page content for security challenges
	if a.pageHelper.ContainsText(page, "verify it's you") ||
		a.pageHelper.ContainsText(page, "security verification") ||
		a.pageHelper.ContainsText(page, "let's do a quick security check") {
		a.logger.Debug().Msg("Security check detected from page content")
		return models.CheckpointUnknown
	}

	return models.CheckpointNone
}

// WaitForManualResolution waits for user to manually resolve a checkpoint
func (a *Authenticator) WaitForManualResolution(page *rod.Page, timeout time.Duration) error {
	a.logger.Info().
		Dur("timeout", timeout).
		Msg("Waiting for manual checkpoint resolution")

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if checkpoint is resolved
		checkpoint := a.DetectCheckpoint(page)
		if checkpoint == models.CheckpointNone {
			// Check if we reached the feed
			currentURL := a.pageHelper.GetCurrentURL(page)
			if strings.Contains(currentURL, "/feed") {
				a.logger.Info().Msg("Checkpoint resolved - reached feed")
				return nil
			}
		}

		a.logger.Debug().
			Str("checkpoint", string(checkpoint)).
			Msg("Still waiting for resolution")

		time.Sleep(2 * time.Second)
	}

	return browser.ErrTimeout
}

// GetCheckpointInstructions returns user instructions for resolving a checkpoint
func GetCheckpointInstructions(checkpointType models.CheckpointType) string {
	switch checkpointType {
	case models.CheckpointTwoFactor:
		return "Two-factor authentication required. Please enter the verification code sent to your device."
	case models.CheckpointCaptcha:
		return "CAPTCHA verification required. Please complete the CAPTCHA challenge in the browser."
	case models.CheckpointPhoneVerify:
		return "Phone verification required. Please enter your phone number and verify the code sent via SMS."
	case models.CheckpointEmailVerify:
		return "Email verification required. Please check your email and enter the verification code."
	case models.CheckpointUnusualActivity:
		return "LinkedIn detected unusual activity. Please complete the security check in the browser."
	default:
		return "Security checkpoint detected. Please complete the verification in the browser."
	}
}

// IsRecoverableCheckpoint returns true if the checkpoint can potentially be resolved
func IsRecoverableCheckpoint(checkpointType models.CheckpointType) bool {
	switch checkpointType {
	case models.CheckpointTwoFactor, models.CheckpointEmailVerify:
		return true // User can enter code
	case models.CheckpointPhoneVerify:
		return true // User can verify phone
	case models.CheckpointCaptcha:
		return true // User can solve CAPTCHA
	case models.CheckpointUnusualActivity:
		return true // User can verify identity
	default:
		return false
	}
}
