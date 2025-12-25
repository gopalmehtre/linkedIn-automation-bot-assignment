// Package config - validation logic for configuration values
package config

import (
	"errors"
	"fmt"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s - %s", e.Field, e.Message)
}

// Validate checks all configuration values for validity
func (c *Config) Validate() error {
	var errs []error

	// Validate limits
	if c.Limits.DailyConnections <= 0 {
		errs = append(errs, ValidationError{
			Field:   "limits.daily_connections",
			Message: "must be greater than 0",
		})
	}

	if c.Limits.DailyMessages <= 0 {
		errs = append(errs, ValidationError{
			Field:   "limits.daily_messages",
			Message: "must be greater than 0",
		})
	}

	if c.Limits.MinDelaySeconds <= 0 {
		errs = append(errs, ValidationError{
			Field:   "limits.min_delay_seconds",
			Message: "must be greater than 0",
		})
	}

	if c.Limits.MaxDelaySeconds < c.Limits.MinDelaySeconds {
		errs = append(errs, ValidationError{
			Field:   "limits.max_delay_seconds",
			Message: "must be greater than or equal to min_delay_seconds",
		})
	}

	// Validate stealth settings
	if c.Stealth.StartHour < 0 || c.Stealth.StartHour > 23 {
		errs = append(errs, ValidationError{
			Field:   "stealth.start_hour",
			Message: "must be between 0 and 23",
		})
	}

	if c.Stealth.EndHour < 0 || c.Stealth.EndHour > 23 {
		errs = append(errs, ValidationError{
			Field:   "stealth.end_hour",
			Message: "must be between 0 and 23",
		})
	}

	if c.Stealth.StartHour >= c.Stealth.EndHour {
		errs = append(errs, ValidationError{
			Field:   "stealth.start_hour/end_hour",
			Message: "start_hour must be less than end_hour",
		})
	}

	if c.Stealth.TypoProbability < 0 || c.Stealth.TypoProbability > 1 {
		errs = append(errs, ValidationError{
			Field:   "stealth.typo_probability",
			Message: "must be between 0 and 1",
		})
	}

	if c.Stealth.MinTypingDelayMs <= 0 {
		errs = append(errs, ValidationError{
			Field:   "stealth.min_typing_delay_ms",
			Message: "must be greater than 0",
		})
	}

	if c.Stealth.MaxTypingDelayMs < c.Stealth.MinTypingDelayMs {
		errs = append(errs, ValidationError{
			Field:   "stealth.max_typing_delay_ms",
			Message: "must be greater than or equal to min_typing_delay_ms",
		})
	}

	if c.Stealth.HoverProbability < 0 || c.Stealth.HoverProbability > 1 {
		errs = append(errs, ValidationError{
			Field:   "stealth.hover_probability",
			Message: "must be between 0 and 1",
		})
	}

	// Validate browser settings
	if c.Browser.ViewportWidth <= 0 {
		errs = append(errs, ValidationError{
			Field:   "browser.viewport_width",
			Message: "must be greater than 0",
		})
	}

	if c.Browser.ViewportHeight <= 0 {
		errs = append(errs, ValidationError{
			Field:   "browser.viewport_height",
			Message: "must be greater than 0",
		})
	}

	// Validate search settings
	if c.Search.MaxPages <= 0 {
		errs = append(errs, ValidationError{
			Field:   "search.max_pages",
			Message: "must be greater than 0",
		})
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ValidateForLogin checks if config is valid for login operation
func (c *Config) ValidateForLogin() error {
	if !c.HasCredentials() {
		return ValidationError{
			Field:   "credentials",
			Message: "LINKEDIN_EMAIL and LINKEDIN_PASSWORD environment variables are required",
		}
	}
	return c.Validate()
}

// ValidateForSearch checks if config is valid for search operation
func (c *Config) ValidateForSearch() error {
	if err := c.Validate(); err != nil {
		return err
	}

	// At least one search parameter should be set
	if c.Search.JobTitle == "" && c.Search.Company == "" && c.Search.Location == "" && len(c.Search.Keywords) == 0 {
		return ValidationError{
			Field:   "search",
			Message: "at least one search parameter (job_title, company, location, or keywords) is required",
		}
	}

	return nil
}
