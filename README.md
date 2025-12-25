# LinkedIn Automation Tool

> ⚠️ **EDUCATIONAL PURPOSE ONLY** - This project is designed for technical evaluation and learning about browser automation techniques. Using automation tools on LinkedIn violates their Terms of Service and may result in account suspension or permanent ban.

A sophisticated Go-based LinkedIn automation tool demonstrating advanced browser automation, human-like behavior simulation, and anti-bot detection techniques using the Rod library.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Stealth Techniques](#stealth-techniques)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Project Structure](#project-structure)
- [Technical Deep Dive](#technical-deep-dive)
- [Troubleshooting](#troubleshooting)

## Features

### Core Capabilities
- 🔐 **Authentication** - Secure login with session persistence and checkpoint handling
- 🔍 **Profile Search** - Find profiles by job title, company, location, and keywords
- 🤝 **Connection Requests** - Send personalized connection requests with custom notes
- 💬 **Messaging** - Automated follow-up messages to new connections
- 📊 **Statistics** - Track daily activity and profile status

### Anti-Detection Suite
- 🖱️ **Bézier Mouse Movement** - Natural cursor paths with overshoot and micro-movements
- ⏱️ **Randomized Timing** - Human-like delays with normal distribution
- 🎭 **Fingerprint Masking** - WebGL, canvas, navigator property modifications
- 📜 **Natural Scrolling** - Acceleration/deceleration patterns
- ⌨️ **Human-like Typing** - Variable speed with occasional typos and corrections
- 🎯 **Random Hovering** - Organic element interaction patterns
- 📅 **Business Hours Scheduler** - Activity restricted to natural working hours
- 🚦 **Rate Limiting** - Configurable daily limits and cooldowns

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI (cmd/main.go)                       │
│              login | search | connect | message | run           │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      LinkedIn Modules                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐    │
│  │   Auth   │  │  Search  │  │ Connect  │  │  Messaging   │    │
│  │  login   │  │  people  │  │ request  │  │  follow-up   │    │
│  │checkpoint│  │ profiles │  │  notes   │  │  templates   │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Stealth Layer                              │
│  ┌─────────┐ ┌─────────┐ ┌───────────┐ ┌─────────┐            │
│  │ Mouse   │ │ Timing  │ │Fingerprint│ │Scrolling│            │
│  │ Bézier  │ │ Random  │ │  Masking  │ │ Natural │            │
│  └─────────┘ └─────────┘ └───────────┘ └─────────┘            │
│  ┌─────────┐ ┌─────────┐ ┌───────────┐ ┌─────────┐            │
│  │ Typing  │ │ Hover   │ │ Scheduler │ │  Rate   │            │
│  │ Human   │ │ Random  │ │ BizHours  │ │ Limiter │            │
│  └─────────┘ └─────────┘ └───────────┘ └─────────┘            │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Browser Layer                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐      │
│  │    Rod +     │  │   Session    │  │   Page Helpers   │      │
│  │   Stealth    │  │   Manager    │  │   & Navigation   │      │
│  └──────────────┘  └──────────────┘  └──────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Storage Layer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Profiles │  │Connections│ │ Messages │  │  Stats   │       │
│  │  SQLite  │  │  SQLite  │  │  SQLite  │  │  SQLite  │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Stealth Techniques

### 1. Bézier Mouse Movement (Mandatory) ✅
Human-like cursor paths using cubic Bézier curves with:
- Randomized control points for organic trajectories
- Occasional overshoot near targets
- Variable speed (faster mid-path, slower at endpoints)
- Micro-movements and jitter

```go
// Example: Natural mouse move to element
stealth.MoveTo(element) // Uses Bézier curves internally
```

### 2. Randomized Timing (Mandatory) ✅
Human typing and interaction timing with:
- Normal distribution for delay variations
- Context-aware delays (reading time based on content length)
- Random pauses between actions
- Variable typing speed (50-150ms per character)

```go
// Example: Human-like delay
stealth.RandomDelay(500*time.Millisecond, 2*time.Second)
stealth.ReadingDelay("Some text content") // Adjusts based on word count
```

### 3. Browser Fingerprint Masking (Mandatory) ✅
Anti-fingerprinting JavaScript injections:
- WebGL vendor/renderer spoofing
- Canvas fingerprint randomization
- Navigator property modifications
- Plugin and MIME type masking
- Timezone and language consistency

```go
// Automatically injected on page load
stealth.ApplyFingerprint(page)
```

### 4. Natural Scrolling ✅
Realistic scroll behavior with:
- Acceleration at start, deceleration at end
- Variable scroll speeds
- Occasional overscroll and correction
- Random pauses during long scrolls

### 5. Human-like Typing ✅
Natural keyboard input simulation:
- Variable inter-key delays
- Occasional typos with backspace corrections
- Speed variations based on character patterns
- Random pauses mid-sentence

### 6. Random Element Hovering ✅
Organic browsing behavior:
- Random element hovering during waits
- Variable hover durations
- Natural cursor movement patterns

### 7. Business Hours Scheduler ✅
Activity timing control:
- Configurable business hours (default: 9 AM - 6 PM)
- Break suggestions after extended use
- Workday-only operation (Mon-Fri)

### 8. Rate Limiting ✅
Configurable action limits:
- Daily connection request limits
- Daily message limits
- Cooldown periods between actions
- Dynamic rate adjustment

## Installation

### Prerequisites
- Go 1.21 or later
- Chrome/Chromium browser (automatically managed by Rod)
- SQLite3 (usually included with the system)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd linkedin-automation

# Download dependencies
go mod download

# Build the binary
go build -o linkedin-automation ./cmd/main.go

# Or run directly
go run ./cmd/main.go help
```

### Verify Installation

```bash
./linkedin-automation help
```

## Configuration

### Step 1: Environment Variables

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
```

Edit `.env`:
```env
LINKEDIN_EMAIL=your-email@example.com
LINKEDIN_PASSWORD=your-password

# Optional overrides
# LOG_LEVEL=debug
# HEADLESS=true
```

### Step 2: Configuration File

Edit `config/config.yaml` to customize behavior:

```yaml
search:
  job_title: "Software Engineer"
  company: ""
  location: "San Francisco Bay Area"
  keywords: "golang, distributed systems"
  max_pages: 5

limits:
  daily_connections: 20
  daily_messages: 30
  min_delay_seconds: 3
  max_delay_seconds: 10

stealth:
  bezier_enabled: true
  random_timing: true
  fingerprint_masking: true
  human_scrolling: true
  typing_simulation: true
  random_hovering: true
  business_hours_only: true
  business_hour_start: 9
  business_hour_end: 18
  rate_limit_enabled: true

messages:
  connection_note: |
    Hi {{.FirstName}}, I came across your profile and was impressed 
    by your work at {{.Company}}. Would love to connect!
  followup: |
    Hi {{.FirstName}}, thanks for connecting! I'd love to learn more 
    about your experience in {{.Industry}}.
```

### Template Variables

Available variables for message templates:
- `{{.FirstName}}` - First name
- `{{.LastName}}` - Last name
- `{{.FullName}}` - Full name
- `{{.Headline}}` - Profile headline
- `{{.Company}}` - Current company
- `{{.Industry}}` - Industry

## Usage

### Commands

```bash
# Authenticate with LinkedIn
./linkedin-automation login

# Search for profiles
./linkedin-automation search

# Send connection requests
./linkedin-automation connect

# Send follow-up messages
./linkedin-automation message

# Full automation cycle
./linkedin-automation run

# View status and statistics
./linkedin-automation status
```

### Command Options

```bash
# Custom config file
./linkedin-automation -config /path/to/config.yaml login

# Enable debug logging
./linkedin-automation -log-level debug search

# Run headless (no visible browser)
./linkedin-automation -headless run
```

### Typical Workflow

```bash
# 1. First time: Login and save session
./linkedin-automation login

# 2. Search for profiles matching your criteria
./linkedin-automation search

# 3. Send connection requests (uses saved profiles)
./linkedin-automation connect

# 4. After connections are accepted, send follow-ups
./linkedin-automation message

# Or run everything automatically
./linkedin-automation run
```

## Project Structure

```
linkedin-automation/
├── cmd/
│   └── main.go              # CLI entry point
├── config/
│   └── config.yaml          # Default configuration
├── internal/
│   ├── browser/
│   │   ├── browser.go       # Rod browser initialization
│   │   ├── session.go       # Cookie/session management
│   │   └── page.go          # Page helper functions
│   ├── config/
│   │   ├── config.go        # Configuration loading
│   │   └── validation.go    # Config validation
│   ├── linkedin/
│   │   ├── auth/
│   │   │   ├── login.go     # Authentication flow
│   │   │   └── checkpoint.go # Security checkpoint handling
│   │   ├── connection/
│   │   │   └── connect.go   # Connection request logic
│   │   ├── messaging/
│   │   │   └── messaging.go # Message sending
│   │   └── search/
│   │       └── search.go    # Profile search
│   ├── models/
│   │   └── models.go        # Data structures
│   ├── stealth/
│   │   ├── stealth.go       # Stealth controller
│   │   ├── mouse.go         # Bézier mouse movement
│   │   ├── timing.go        # Random timing
│   │   ├── fingerprint.go   # Browser fingerprint masking
│   │   ├── scrolling.go     # Natural scrolling
│   │   ├── typing.go        # Human-like typing
│   │   ├── hover.go         # Random hovering
│   │   ├── scheduler.go     # Business hours
│   │   └── ratelimit.go     # Rate limiting
│   └── storage/
│       ├── database.go      # SQLite connection
│       ├── profiles.go      # Profile storage
│       ├── connections.go   # Connection tracking
│       ├── messages.go      # Message history
│       └── stats.go         # Daily statistics
├── data/                    # Runtime data (gitignored)
│   ├── linkedin.db          # SQLite database
│   └── cookies.json         # Session cookies
├── .env                     # Environment variables (gitignored)
├── .env.example             # Environment template
├── go.mod                   # Go module definition
└── README.md                # This file
```

## Technical Deep Dive

### Browser Automation with Rod

Rod is a high-level Chrome DevTools Protocol driver:

```go
// Browser initialization with stealth
browser := rod.New().
    ControlURL(launcher.New().
        Set(flags.UserAgent, customUserAgent).
        Headless(false).
        MustLaunch()).
    MustConnect()

// Inject stealth scripts before page load
page.EvalOnNewDocument(stealthJS)
```

### Session Management

Sessions are persisted as JSON cookies:
- Cookies are saved after successful login
- Loaded on startup to skip authentication
- Session age and validity are tracked
- Automatic refresh when expired

### Database Schema

SQLite with four main tables:
- `profiles` - Discovered LinkedIn profiles
- `connection_requests` - Sent connection tracking
- `messages` - Message history
- `daily_stats` - Activity counters

### Rate Limiting Algorithm

```go
// Token bucket with configurable limits
type RateLimiter struct {
    dailyLimit int
    usedToday  int
    cooldown   time.Duration
    lastAction time.Time
}

func (r *RateLimiter) Allow() bool {
    if r.usedToday >= r.dailyLimit {
        return false
    }
    if time.Since(r.lastAction) < r.cooldown {
        return false
    }
    return true
}
```

## Troubleshooting

### Common Issues

**Browser doesn't open**
```bash
# Ensure Chrome is installed or let Rod download it
go run ./cmd/main.go -log-level debug login
```

**Login fails with checkpoint**
- Complete the verification manually in the browser
- The tool will wait up to 5 minutes for resolution
- 2FA codes can be entered manually

**Connection requests not sending**
- Check daily limits in `config.yaml`
- Verify business hours settings
- Review `./linkedin-automation status`

**"Session invalid" errors**
- Delete `data/cookies.json` and re-login
- Sessions expire after ~24 hours

### Debug Mode

Enable detailed logging:
```bash
./linkedin-automation -log-level debug <command>
```

### Reset Everything

```bash
# Clear all data
rm -rf data/
rm .env
cp .env.example .env
```

## Legal Disclaimer

This tool is provided for **educational purposes only** to demonstrate:
- Browser automation techniques
- Anti-detection methodologies
- Go programming patterns
- Clean architecture principles

**DO NOT** use this tool to:
- Automate actions on real LinkedIn accounts
- Violate LinkedIn's Terms of Service
- Send spam or unsolicited messages
- Harvest user data without consent

The authors are not responsible for any misuse of this software or any resulting account suspensions, bans, or legal actions.

## License

MIT License - See LICENSE file for details.

---

Built with ❤️ for educational purposes using Go + Rod
