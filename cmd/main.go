// LinkedIn Automation Tool - Educational Purpose Only
// This tool demonstrates browser automation techniques and anti-detection patterns.
// DO NOT use this on live LinkedIn accounts - it violates their Terms of Service.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/config"
	"linkedin-automation/internal/linkedin/auth"
	"linkedin-automation/internal/linkedin/connection"
	"linkedin-automation/internal/linkedin/messaging"
	"linkedin-automation/internal/linkedin/search"
	"linkedin-automation/internal/models"
	"linkedin-automation/internal/stealth"
	"linkedin-automation/internal/storage"
)

// Version info
const (
	AppName    = "linkedin-automation"
	AppVersion = "1.0.0"
)

// Command line flags
var (
	configPath = flag.String("config", "./config/config.yaml", "Path to config file")
	logLevel   = flag.String("log-level", "", "Log level (debug, info, warn, error)")
	headless   = flag.Bool("headless", false, "Run in headless mode")
)

// App holds all application dependencies
type App struct {
	config          *config.Config
	logger          zerolog.Logger
	db              *storage.Database
	browser         *browser.Browser
	sessionManager  *browser.SessionManager
	stealth         *stealth.Controller
	authenticator   *auth.Authenticator
	searcher        *search.Searcher
	connectionMgr   *connection.ConnectionManager
	messenger       *messaging.Messenger
	profileStore    *storage.ProfileStore
	connectionStore *storage.ConnectionStore
	messageStore    *storage.MessageStore
	statsStore      *storage.StatsStore
}

func main() {
	flag.Parse()

	// Print banner
	printBanner()

	// Check for command
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	// Initialize app
	app, err := NewApp()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize application")
	}
	defer app.Cleanup()

	// Setup graceful shutdown
	app.setupSignalHandler()

	// Execute command
	var cmdErr error
	switch command {
	case "login":
		cmdErr = app.cmdLogin()
	case "search":
		cmdErr = app.cmdSearch()
	case "connect":
		cmdErr = app.cmdConnect()
	case "message":
		cmdErr = app.cmdMessage()
	case "run":
		cmdErr = app.cmdRun()
	case "status":
		cmdErr = app.cmdStatus()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if cmdErr != nil {
		app.logger.Error().Err(cmdErr).Msg("Command failed")
		os.Exit(1)
	}
}

// NewApp creates and initializes the application
func NewApp() (*App, error) {
	app := &App{}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	app.config = cfg

	// Override with command line flags
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *headless {
		cfg.Browser.Headless = true
	}

	// Setup logging
	app.setupLogging()
	app.logger.Info().Str("version", AppVersion).Msg("Starting application")

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize database
	db, err := storage.Open(cfg.Storage.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	app.db = db

	// Initialize stores
	app.profileStore = storage.NewProfileStore(db)
	app.connectionStore = storage.NewConnectionStore(db)
	app.messageStore = storage.NewMessageStore(db)
	app.statsStore = storage.NewStatsStore(db)

	// Initialize stealth controller
	app.stealth = stealth.NewController(&cfg.Stealth, app.logger)

	// Initialize session manager
	app.sessionManager = browser.NewSessionManager(cfg.Storage.CookiesPath, app.logger)

	app.logger.Info().Msg("Application initialized")
	return app, nil
}

// initBrowser initializes the browser (lazy initialization)
func (app *App) initBrowser() error {
	if app.browser != nil {
		return nil
	}

	app.logger.Info().Msg("Initializing browser")

	b, err := browser.NewBrowser(&app.config.Browser, app.stealth, app.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize browser: %w", err)
	}
	app.browser = b

	// Load saved cookies
	if err := app.sessionManager.LoadCookies(b.Browser()); err != nil {
		app.logger.Warn().Err(err).Msg("Failed to load saved cookies")
	}

	// Initialize LinkedIn components
	app.authenticator = auth.NewAuthenticator(b, app.sessionManager, app.stealth, app.logger)
	app.searcher = search.NewSearcher(b, app.profileStore, app.statsStore, &app.config.Search, app.stealth, app.logger)
	app.connectionMgr = connection.NewConnectionManager(
		b, app.profileStore, app.connectionStore, app.statsStore,
		&app.config.Limits, &app.config.Stealth, app.stealth, app.logger,
	)
	app.messenger = messaging.NewMessenger(
		b, app.profileStore, app.connectionStore, app.messageStore, app.statsStore,
		&app.config.Limits, app.stealth, app.logger,
	)

	return nil
}

// setupLogging configures the logger
func (app *App) setupLogging() {
	// Pretty console output
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	// Set log level
	level := zerolog.InfoLevel
	switch app.config.LogLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}

	app.logger = zerolog.New(output).Level(level).With().Timestamp().Logger()
	log.Logger = app.logger
}

// setupSignalHandler handles graceful shutdown
func (app *App) setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		app.Cleanup()
		os.Exit(0)
	}()
}

// Cleanup releases all resources
func (app *App) Cleanup() {
	app.logger.Info().Msg("Cleaning up resources")

	if app.browser != nil {
		// Save cookies before closing
		app.sessionManager.SaveCookies(app.browser.Browser())
		app.browser.Close()
	}

	if app.db != nil {
		app.db.Close()
	}
}

// cmdLogin handles the login command
func (app *App) cmdLogin() error {
	app.logger.Info().Msg("=== Login Command ===")

	// Validate credentials
	if err := app.config.ValidateForLogin(); err != nil {
		return err
	}

	// Initialize browser
	if err := app.initBrowser(); err != nil {
		return err
	}

	// Perform login
	result, err := app.authenticator.Login(app.config.LinkedInEmail, app.config.LinkedInPassword)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if result.Success {
		app.logger.Info().Msg("Login successful!")
		if result.SessionSaved {
			app.logger.Info().Msg("Session saved for future use")
		}
	} else {
		if result.CheckpointType != models.CheckpointNone {
			app.logger.Warn().
				Str("checkpoint", string(result.CheckpointType)).
				Msg("Security checkpoint detected")
			fmt.Println("\n" + auth.GetCheckpointInstructions(result.CheckpointType))
			fmt.Println("\nPlease complete the verification in the browser window.")

			// Wait for manual resolution
			page, _ := app.browser.GetPage()
			if err := app.authenticator.WaitForManualResolution(page, 5*time.Minute); err != nil {
				return fmt.Errorf("checkpoint not resolved: %w", err)
			}
			app.logger.Info().Msg("Checkpoint resolved!")
		} else {
			return fmt.Errorf("login failed: %s", result.ErrorMessage)
		}
	}

	return nil
}

// cmdSearch handles the search command
func (app *App) cmdSearch() error {
	app.logger.Info().Msg("=== Search Command ===")

	// Validate search config
	if err := app.config.ValidateForSearch(); err != nil {
		return err
	}

	// Initialize browser
	if err := app.initBrowser(); err != nil {
		return err
	}

	// Check session
	valid, err := app.authenticator.VerifySession()
	if err != nil || !valid {
		app.logger.Info().Msg("Session invalid, logging in first")
		if err := app.cmdLogin(); err != nil {
			return err
		}
	}

	// Build search params
	params := models.SearchParams{
		JobTitle: app.config.Search.JobTitle,
		Company:  app.config.Search.Company,
		Location: app.config.Search.Location,
		Keywords: app.config.Search.Keywords,
	}

	// Perform search
	newCount, dupCount, err := app.searcher.SearchAndSave(params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	app.logger.Info().
		Int("new", newCount).
		Int("duplicates", dupCount).
		Msg("Search completed")

	// Print stats
	total, _ := app.profileStore.Count()
	pending, _ := app.profileStore.CountByStatus(models.ProfileStatusFound)
	fmt.Printf("\nDatabase Stats:\n")
	fmt.Printf("  Total profiles: %d\n", total)
	fmt.Printf("  Pending (not contacted): %d\n", pending)

	return nil
}

// cmdConnect handles the connect command
func (app *App) cmdConnect() error {
	app.logger.Info().Msg("=== Connect Command ===")

	// Initialize browser
	if err := app.initBrowser(); err != nil {
		return err
	}

	// Check session
	valid, err := app.authenticator.VerifySession()
	if err != nil || !valid {
		app.logger.Info().Msg("Session invalid, logging in first")
		if err := app.cmdLogin(); err != nil {
			return err
		}
	}

	// Check schedule
	if !app.stealth.IsWithinSchedule() {
		app.logger.Info().Msg("Outside business hours")
		if app.config.Stealth.BusinessHoursOnly {
			fmt.Println("Waiting for business hours...")
			app.stealth.WaitForSchedule()
		}
	}

	// Process pending profiles
	sentCount, err := app.connectionMgr.ProcessPendingProfiles(
		app.config.Messages.ConnectionNote,
		app.config.Limits.DailyConnections,
	)
	if err != nil {
		return fmt.Errorf("connection processing failed: %w", err)
	}

	app.logger.Info().Int("sent", sentCount).Msg("Connection requests sent")
	return nil
}

// cmdMessage handles the message command
func (app *App) cmdMessage() error {
	app.logger.Info().Msg("=== Message Command ===")

	// Initialize browser
	if err := app.initBrowser(); err != nil {
		return err
	}

	// Check session
	valid, err := app.authenticator.VerifySession()
	if err != nil || !valid {
		app.logger.Info().Msg("Session invalid, logging in first")
		if err := app.cmdLogin(); err != nil {
			return err
		}
	}

	// Process follow-ups
	newConnections, sentCount, err := app.messenger.ProcessFollowups(
		app.config.Messages.Followup,
		app.config.Limits.DailyMessages,
	)
	if err != nil {
		return fmt.Errorf("messaging failed: %w", err)
	}

	app.logger.Info().
		Int("newConnections", newConnections).
		Int("messagesSent", sentCount).
		Msg("Messaging completed")

	return nil
}

// cmdRun handles the full automation run
func (app *App) cmdRun() error {
	app.logger.Info().Msg("=== Full Automation Run ===")

	// Initialize browser
	if err := app.initBrowser(); err != nil {
		return err
	}

	// Login
	if err := app.cmdLogin(); err != nil {
		return err
	}

	app.logger.Info().Msg("Starting automation cycle")

	// Search if no pending profiles
	pending, _ := app.profileStore.CountByStatus(models.ProfileStatusFound)
	if pending == 0 {
		app.logger.Info().Msg("No pending profiles, running search")
		if err := app.cmdSearch(); err != nil {
			app.logger.Warn().Err(err).Msg("Search failed")
		}
	}

	// Send connection requests
	app.logger.Info().Msg("Processing connection requests")
	if err := app.cmdConnect(); err != nil {
		app.logger.Warn().Err(err).Msg("Connection processing failed")
	}

	// Send follow-up messages
	app.logger.Info().Msg("Processing follow-up messages")
	if err := app.cmdMessage(); err != nil {
		app.logger.Warn().Err(err).Msg("Messaging failed")
	}

	// Print summary
	app.cmdStatus()

	return nil
}

// cmdStatus prints current status and statistics
func (app *App) cmdStatus() error {
	fmt.Println("\n========== Status ==========")

	// Profile counts
	total, _ := app.profileStore.Count()
	found, _ := app.profileStore.CountByStatus(models.ProfileStatusFound)
	requested, _ := app.profileStore.CountByStatus(models.ProfileStatusRequested)
	connected, _ := app.profileStore.CountByStatus(models.ProfileStatusConnected)

	fmt.Printf("\nProfiles:\n")
	fmt.Printf("  Total:      %d\n", total)
	fmt.Printf("  Pending:    %d\n", found)
	fmt.Printf("  Requested:  %d\n", requested)
	fmt.Printf("  Connected:  %d\n", connected)

	// Today's stats
	stats, _ := app.statsStore.GetOrCreateToday()
	fmt.Printf("\nToday's Activity:\n")
	fmt.Printf("  Connections sent:  %d / %d\n", stats.ConnectionsSent, app.config.Limits.DailyConnections)
	fmt.Printf("  Messages sent:     %d / %d\n", stats.MessagesSent, app.config.Limits.DailyMessages)
	fmt.Printf("  Profiles searched: %d\n", stats.ProfilesSearched)

	// Session status
	fmt.Printf("\nSession:\n")
	if app.sessionManager.HasSavedSession() {
		age, _ := app.sessionManager.GetSessionAge()
		fmt.Printf("  Saved session: %s ago\n", age.Round(time.Minute))
		fmt.Printf("  Valid: %v\n", app.sessionManager.IsSessionValid())
	} else {
		fmt.Printf("  No saved session\n")
	}

	fmt.Println("\n============================")
	return nil
}

// printBanner prints the application banner
func printBanner() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║          LinkedIn Automation Tool - v` + AppVersion + `                   ║
║                                                               ║
║  ⚠️  EDUCATIONAL PURPOSE ONLY - DO NOT USE IN PRODUCTION  ⚠️  ║
║                                                               ║
║  This tool violates LinkedIn's Terms of Service.              ║
║  Using it on real accounts may result in permanent bans.      ║
╚═══════════════════════════════════════════════════════════════╝
`)
}

// printUsage prints usage information
func printUsage() {
	fmt.Println(`
Usage: linkedin-automation [options] <command>

Commands:
  login     Authenticate with LinkedIn and save session
  search    Search for profiles matching criteria
  connect   Send connection requests to pending profiles  
  message   Send follow-up messages to new connections
  run       Full automation cycle (search → connect → message)
  status    Show current statistics and status
  help      Show this help message

Options:
  -config string    Path to config file (default "./config/config.yaml")
  -log-level string Log level: debug, info, warn, error
  -headless         Run browser in headless mode

Examples:
  linkedin-automation login
  linkedin-automation search
  linkedin-automation -log-level debug connect
  linkedin-automation run

Configuration:
  1. Copy .env.example to .env and add your LinkedIn credentials
  2. Edit config/config.yaml to customize search parameters and limits
  3. Run 'linkedin-automation login' to authenticate

For more information, see README.md
`)
}
