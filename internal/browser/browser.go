package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
	stealthpkg "linkedin-automation/internal/stealth"
)


type Browser struct {
	browser *rod.Browser
	config  *config.BrowserConfig
	stealth *stealthpkg.Controller
	logger  zerolog.Logger
}

func NewBrowser(cfg *config.BrowserConfig, stealthCtrl *stealthpkg.Controller, logger zerolog.Logger) (*Browser, error) {
	logger = logger.With().Str("component", "browser").Logger()
	logger.Info().Msg("Initializing browser")

	// Ensure user data directory exists
	if cfg.UserDataDir != "" {
		if err := os.MkdirAll(cfg.UserDataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create user data directory: %w", err)
		}
	}

	// Configure launcher
	l := launcher.New()

	// Set user data directory for session persistence
	if cfg.UserDataDir != "" {
		absPath, err := filepath.Abs(cfg.UserDataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for user data dir: %w", err)
		}
		l = l.UserDataDir(absPath)
	}

	// Set headless mode
	if cfg.Headless {
		l = l.Headless(true)
		logger.Info().Msg("Running in headless mode")
	} else {
		l = l.Headless(false)
		logger.Info().Msg("Running in headed mode (visible browser)")
	}

	// Add stealth flags
	l = l.Set("disable-blink-features", "AutomationControlled")
	l = l.Set("disable-infobars")
	l = l.Set("disable-dev-shm-usage")
	l = l.Set("no-first-run")
	l = l.Set("no-default-browser-check")

	// Set random user agent
	userAgent := stealthpkg.GetRandomUserAgent()
	l = l.Set("user-agent", userAgent)
	logger.Debug().Str("userAgent", userAgent).Msg("Set user agent")

	// Launch browser
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// Set default timeout
	browser = browser.Timeout(30 * time.Second)

	logger.Info().Msg("Browser initialized successfully")

	return &Browser{
		browser: browser,
		config:  cfg,
		stealth: stealthCtrl,
		logger:  logger,
	}, nil
}

// NewPage creates a new page with stealth settings applied
func (b *Browser) NewPage() (*rod.Page, error) {
	b.logger.Debug().Msg("Creating new page with stealth")

	// Use stealth plugin to create page
	page, err := stealth.Page(b.browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create stealth page: %w", err)
	}

	// Set viewport
	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  b.config.ViewportWidth,
		Height: b.config.ViewportHeight,
	})
	if err != nil {
		b.logger.Warn().Err(err).Msg("Failed to set viewport")
	}

	// Apply additional stealth settings
	if b.stealth != nil {
		if err := b.stealth.ApplyToPage(page); err != nil {
			b.logger.Warn().Err(err).Msg("Failed to apply stealth settings")
		}
	}

	return page, nil
}

// GetPage returns an existing page or creates a new one
func (b *Browser) GetPage() (*rod.Page, error) {
	pages, err := b.browser.Pages()
	if err != nil {
		return nil, err
	}

	if len(pages) > 0 {
		return pages[0], nil
	}

	return b.NewPage()
}

// Navigate navigates to a URL with proper waiting
func (b *Browser) Navigate(page *rod.Page, url string) error {
	b.logger.Debug().Str("url", url).Msg("Navigating to URL")

	// Navigate
	err := page.Navigate(url)
	if err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for page to be stable
	if err := page.WaitLoad(); err != nil {
		b.logger.Warn().Err(err).Msg("WaitLoad failed, continuing anyway")
	}

	// Additional stability wait
	page.WaitDOMStable(time.Second, 0.1)

	// Add stealth delay after navigation
	if b.stealth != nil {
		b.stealth.Timing().PageLoadDelay()
	}

	return nil
}

// Close closes the browser
func (b *Browser) Close() error {
	b.logger.Info().Msg("Closing browser")
	return b.browser.Close()
}

// Browser returns the underlying rod.Browser
func (b *Browser) Browser() *rod.Browser {
	return b.browser
}

// IsConnected checks if browser is still connected
func (b *Browser) IsConnected() bool {
	pages, err := b.browser.Pages()
	return err == nil && pages != nil
}

// TakeScreenshot captures a screenshot of the current page
func (b *Browser) TakeScreenshot(page *rod.Page, filename string) error {
	data, err := page.Screenshot(true, nil)
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	b.logger.Debug().Str("filename", filename).Msg("Screenshot saved")
	return nil
}
