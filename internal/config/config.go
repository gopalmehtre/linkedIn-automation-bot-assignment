// Package config handles configuration loading and validation for the LinkedIn automation tool.
// It supports YAML configuration files with environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the LinkedIn automation tool
type Config struct {
	Search   SearchConfig   `yaml:"search"`
	Limits   LimitsConfig   `yaml:"limits"`
	Stealth  StealthConfig  `yaml:"stealth"`
	Browser  BrowserConfig  `yaml:"browser"`
	Messages MessagesConfig `yaml:"messages"`
	Storage  StorageConfig  `yaml:"storage"`

	// Credentials loaded from environment
	LinkedInEmail    string `yaml:"-"`
	LinkedInPassword string `yaml:"-"`
	LogLevel         string `yaml:"-"`
}

// SearchConfig holds search parameters
type SearchConfig struct {
	JobTitle string   `yaml:"job_title"`
	Company  string   `yaml:"company"`
	Location string   `yaml:"location"`
	Keywords []string `yaml:"keywords"`
	MaxPages int      `yaml:"max_pages"`
}

// LimitsConfig holds rate limiting settings
type LimitsConfig struct {
	DailyConnections   int `yaml:"daily_connections"`
	DailyMessages      int `yaml:"daily_messages"`
	MinDelaySeconds    int `yaml:"min_delay_seconds"`
	MaxDelaySeconds    int `yaml:"max_delay_seconds"`
	ConnectionsPerHour int `yaml:"connections_per_hour"`
	MessagesPerHour    int `yaml:"messages_per_hour"`
}

// StealthConfig holds anti-detection settings
type StealthConfig struct {
	// Business hours
	BusinessHoursOnly bool `yaml:"business_hours_only"`
	StartHour         int  `yaml:"start_hour"`
	EndHour           int  `yaml:"end_hour"`

	// Typing simulation
	EnableTypos       bool    `yaml:"enable_typos"`
	TypoProbability   float64 `yaml:"typo_probability"`
	MinTypingDelayMs  int     `yaml:"min_typing_delay_ms"`
	MaxTypingDelayMs  int     `yaml:"max_typing_delay_ms"`

	// Mouse movement
	MouseSpeedMin   float64 `yaml:"mouse_speed_min"`
	MouseSpeedMax   float64 `yaml:"mouse_speed_max"`
	EnableOvershoot bool    `yaml:"enable_overshoot"`

	// Scrolling
	ScrollSpeedMin   int  `yaml:"scroll_speed_min"`
	ScrollSpeedMax   int  `yaml:"scroll_speed_max"`
	EnableScrollBack bool `yaml:"enable_scroll_back"`

	// Random actions
	EnableRandomHovers bool    `yaml:"enable_random_hovers"`
	HoverProbability   float64 `yaml:"hover_probability"`
}

// BrowserConfig holds browser settings
type BrowserConfig struct {
	Headless       bool   `yaml:"headless"`
	UserDataDir    string `yaml:"user_data_dir"`
	ViewportWidth  int    `yaml:"viewport_width"`
	ViewportHeight int    `yaml:"viewport_height"`
}

// MessagesConfig holds message templates
type MessagesConfig struct {
	ConnectionNote string `yaml:"connection_note"`
	Followup       string `yaml:"followup"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	DatabasePath string `yaml:"database_path"`
	CookiesPath  string `yaml:"cookies_path"`
}

// Load reads configuration from YAML file and environment variables
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Set defaults
	cfg := &Config{
		Search: SearchConfig{
			MaxPages: 5,
		},
		Limits: LimitsConfig{
			DailyConnections:   50,
			DailyMessages:      100,
			MinDelaySeconds:    30,
			MaxDelaySeconds:    90,
			ConnectionsPerHour: 10,
			MessagesPerHour:    20,
		},
		Stealth: StealthConfig{
			BusinessHoursOnly: true,
			StartHour:         9,
			EndHour:           18,
			EnableTypos:       true,
			TypoProbability:   0.03,
			MinTypingDelayMs:  50,
			MaxTypingDelayMs:  200,
			MouseSpeedMin:     0.5,
			MouseSpeedMax:     2.0,
			EnableOvershoot:   true,
			ScrollSpeedMin:    100,
			ScrollSpeedMax:    400,
			EnableScrollBack:  true,
			EnableRandomHovers: true,
			HoverProbability:  0.3,
		},
		Browser: BrowserConfig{
			Headless:       false,
			UserDataDir:    "./data/browser",
			ViewportWidth:  1920,
			ViewportHeight: 1080,
		},
		Storage: StorageConfig{
			DatabasePath: "./data/linkedin.db",
			CookiesPath:  "./data/cookies.json",
		},
		LogLevel: "info",
	}

	// Load YAML config if file exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	cfg.loadEnvOverrides()

	return cfg, nil
}

// loadEnvOverrides applies environment variable overrides to config
func (c *Config) loadEnvOverrides() {
	// Required credentials
	c.LinkedInEmail = os.Getenv("LINKEDIN_EMAIL")
	c.LinkedInPassword = os.Getenv("LINKEDIN_PASSWORD")

	// Optional overrides
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = strings.ToLower(v)
	}

	if v := os.Getenv("HEADLESS"); v != "" {
		c.Browser.Headless = strings.ToLower(v) == "true"
	}

	if v := os.Getenv("DAILY_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Limits.DailyConnections = n
		}
	}

	if v := os.Getenv("DAILY_MESSAGES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Limits.DailyMessages = n
		}
	}

	if v := os.Getenv("DATABASE_PATH"); v != "" {
		c.Storage.DatabasePath = v
	}

	if v := os.Getenv("COOKIES_PATH"); v != "" {
		c.Storage.CookiesPath = v
	}
}

// HasCredentials checks if LinkedIn credentials are configured
func (c *Config) HasCredentials() bool {
	return c.LinkedInEmail != "" && c.LinkedInPassword != ""
}
