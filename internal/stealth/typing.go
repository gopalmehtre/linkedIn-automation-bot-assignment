// Package stealth - human-like typing simulation
// This is additional stealth technique #2
package stealth

import (
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// TypingController handles human-like text input
type TypingController struct {
	config *config.StealthConfig
	logger zerolog.Logger
}

// NewTypingController creates a new typing controller
func NewTypingController(cfg *config.StealthConfig, logger zerolog.Logger) *TypingController {
	return &TypingController{
		config: cfg,
		logger: logger.With().Str("module", "typing").Logger(),
	}
}

// TypeText types text into an element with human-like patterns
func (t *TypingController) TypeText(element *rod.Element, text string) error {
	t.logger.Debug().
		Int("length", len(text)).
		Msg("Typing text with human-like patterns")

	// Focus the element first
	if err := element.Focus(); err != nil {
		return err
	}

	// Small delay after focusing
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Type character by character
	runes := []rune(text)
	for i, char := range runes {
		// Determine if we should make a typo
		if t.config.EnableTypos && t.shouldMakeTypo() {
			t.makeAndCorrectTypo(element, char)
		} else {
			// Type the character
			if err := element.Type(input.Key(char)); err != nil {
				// Fallback to Input for special characters
				if err := element.Input(string(char)); err != nil {
					return err
				}
			}
		}

		// Calculate delay before next character
		delay := t.calculateKeystrokeDelay(char, runes, i)
		time.Sleep(delay)
	}

	return nil
}

// TypeTextFast types text faster (for less critical fields)
func (t *TypingController) TypeTextFast(element *rod.Element, text string) error {
	t.logger.Debug().
		Int("length", len(text)).
		Msg("Typing text (fast mode)")

	if err := element.Focus(); err != nil {
		return err
	}

	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	for _, char := range text {
		if err := element.Type(input.Key(char)); err != nil {
			if err := element.Input(string(char)); err != nil {
				return err
			}
		}

		// Faster base delay
		delay := time.Duration(30+rand.Intn(50)) * time.Millisecond
		time.Sleep(delay)
	}

	return nil
}

// ClearAndType clears an input field and types new text
func (t *TypingController) ClearAndType(element *rod.Element, text string) error {
	// Select all existing text
	if err := element.SelectAllText(); err != nil {
		// If select fails, try clicking and using keyboard shortcut
		element.MustClick()
		time.Sleep(100 * time.Millisecond)
	}

	// Delete selected text
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// Type the new text
	return t.TypeText(element, text)
}

// shouldMakeTypo determines if a typo should be made based on probability
func (t *TypingController) shouldMakeTypo() bool {
	return rand.Float64() < t.config.TypoProbability
}

// makeAndCorrectTypo types a wrong character, then corrects it
func (t *TypingController) makeAndCorrectTypo(element *rod.Element, intended rune) {
	t.logger.Debug().Msg("Simulating typo and correction")

	// Type a nearby key (simulate mis-press)
	typoChar := t.getNearbyKey(intended)
	element.Type(input.Key(typoChar))

	// Pause to "notice" the mistake
	time.Sleep(time.Duration(200+rand.Intn(400)) * time.Millisecond)

	// Press backspace
	element.Type(input.Backspace)
	time.Sleep(time.Duration(100+rand.Intn(150)) * time.Millisecond)

	// Type the correct character
	element.Type(input.Key(intended))
}

// getNearbyKey returns a key near the intended key on a QWERTY keyboard
func (t *TypingController) getNearbyKey(char rune) rune {
	// QWERTY keyboard layout neighbors
	neighbors := map[rune][]rune{
		'q': {'w', 'a', '1', '2'},
		'w': {'q', 'e', 's', 'a', '2', '3'},
		'e': {'w', 'r', 'd', 's', '3', '4'},
		'r': {'e', 't', 'f', 'd', '4', '5'},
		't': {'r', 'y', 'g', 'f', '5', '6'},
		'y': {'t', 'u', 'h', 'g', '6', '7'},
		'u': {'y', 'i', 'j', 'h', '7', '8'},
		'i': {'u', 'o', 'k', 'j', '8', '9'},
		'o': {'i', 'p', 'l', 'k', '9', '0'},
		'p': {'o', 'l', '0', '-'},
		'a': {'q', 'w', 's', 'z'},
		's': {'a', 'w', 'e', 'd', 'z', 'x'},
		'd': {'s', 'e', 'r', 'f', 'x', 'c'},
		'f': {'d', 'r', 't', 'g', 'c', 'v'},
		'g': {'f', 't', 'y', 'h', 'v', 'b'},
		'h': {'g', 'y', 'u', 'j', 'b', 'n'},
		'j': {'h', 'u', 'i', 'k', 'n', 'm'},
		'k': {'j', 'i', 'o', 'l', 'm', ','},
		'l': {'k', 'o', 'p', ',', '.'},
		'z': {'a', 's', 'x'},
		'x': {'z', 's', 'd', 'c'},
		'c': {'x', 'd', 'f', 'v'},
		'v': {'c', 'f', 'g', 'b'},
		'b': {'v', 'g', 'h', 'n'},
		'n': {'b', 'h', 'j', 'm'},
		'm': {'n', 'j', 'k', ','},
	}

	lowerChar := unicode.ToLower(char)
	if nearby, ok := neighbors[lowerChar]; ok && len(nearby) > 0 {
		typo := nearby[rand.Intn(len(nearby))]
		// Preserve case
		if unicode.IsUpper(char) {
			return unicode.ToUpper(typo)
		}
		return typo
	}

	// If no neighbor found, return a random nearby character
	return char
}

// calculateKeystrokeDelay calculates delay based on character context
func (t *TypingController) calculateKeystrokeDelay(char rune, text []rune, index int) time.Duration {
	minDelay := t.config.MinTypingDelayMs
	maxDelay := t.config.MaxTypingDelayMs

	// Base delay with variation
	baseDelay := minDelay + rand.Intn(maxDelay-minDelay)

	// Modify based on context
	multiplier := 1.0

	// Slower after punctuation (thinking)
	if index > 0 && strings.ContainsRune(".,!?;:", text[index-1]) {
		multiplier = 1.5 + rand.Float64()*0.5
	}

	// Slower at word boundaries
	if char == ' ' {
		multiplier = 1.2 + rand.Float64()*0.3
	}

	// Faster for common letter combinations
	if index > 0 {
		pair := strings.ToLower(string([]rune{text[index-1], char}))
		commonPairs := []string{"th", "he", "in", "er", "an", "re", "on", "at", "en", "nd", "ti", "es", "or", "te", "of", "ed", "is", "it", "al", "ar", "st", "to", "nt", "ng", "se", "ha", "as", "ou", "io", "le", "ve", "co", "me", "de", "hi", "ri", "ro", "ic", "ne", "ea", "ra", "ce"}
		for _, cp := range commonPairs {
			if pair == cp {
				multiplier = 0.7 + rand.Float64()*0.2
				break
			}
		}
	}

	// Occasionally pause longer (thinking mid-word)
	if rand.Float64() < 0.02 {
		multiplier = 2.0 + rand.Float64()
	}

	delay := int(float64(baseDelay) * multiplier)
	return time.Duration(delay) * time.Millisecond
}

// TypeWithPauses types text with random pauses at word boundaries
func (t *TypingController) TypeWithPauses(element *rod.Element, text string) error {
	words := strings.Split(text, " ")

	for i, word := range words {
		// Type the word
		if err := t.TypeText(element, word); err != nil {
			return err
		}

		// Add space if not last word
		if i < len(words)-1 {
			element.Type(input.Key(' '))

			// Random pause between words
			if rand.Float64() < 0.2 {
				pause := time.Duration(500+rand.Intn(1500)) * time.Millisecond
				t.logger.Debug().Dur("pause", pause).Msg("Pausing between words")
				time.Sleep(pause)
			} else {
				time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
			}
		}
	}

	return nil
}
