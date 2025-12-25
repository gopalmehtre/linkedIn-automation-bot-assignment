// Package browser - page interaction utilities
package browser

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog"
)

// Common errors
var (
	ErrElementNotFound = errors.New("element not found")
	ErrTimeout         = errors.New("operation timed out")
)

// PageHelper provides utilities for page interactions
type PageHelper struct {
	logger zerolog.Logger
}

// NewPageHelper creates a new page helper
func NewPageHelper(logger zerolog.Logger) *PageHelper {
	return &PageHelper{
		logger: logger.With().Str("component", "pagehelper").Logger(),
	}
}

// WaitForElement waits for an element to appear with timeout
func (p *PageHelper) WaitForElement(page *rod.Page, selector string, timeout time.Duration) (*rod.Element, error) {
	p.logger.Debug().
		Str("selector", selector).
		Dur("timeout", timeout).
		Msg("Waiting for element")

	page = page.Timeout(timeout)
	defer page.CancelTimeout()

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrElementNotFound, selector)
	}

	// Wait for it to be visible
	err = element.WaitVisible()
	if err != nil {
		return nil, fmt.Errorf("element not visible: %s: %w", selector, err)
	}

	return element, nil
}

// WaitForElementStable waits for element to be stable (not moving)
func (p *PageHelper) WaitForElementStable(element *rod.Element, timeout time.Duration) error {
	element = element.Timeout(timeout)
	defer element.CancelTimeout()

	return element.WaitStable(200 * time.Millisecond)
}

// ElementExists checks if an element exists on the page
func (p *PageHelper) ElementExists(page *rod.Page, selector string) bool {
	page = page.Timeout(2 * time.Second)
	defer page.CancelTimeout()

	_, err := page.Element(selector)
	return err == nil
}

// ElementVisible checks if an element is visible
func (p *PageHelper) ElementVisible(page *rod.Page, selector string) bool {
	page = page.Timeout(2 * time.Second)
	defer page.CancelTimeout()

	element, err := page.Element(selector)
	if err != nil {
		return false
	}

	visible, err := element.Visible()
	return err == nil && visible
}

// GetElementText safely gets text from an element
func (p *PageHelper) GetElementText(element *rod.Element) string {
	text, err := element.Text()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// GetElementAttribute gets an attribute value from an element
func (p *PageHelper) GetElementAttribute(element *rod.Element, attr string) string {
	val, err := element.Attribute(attr)
	if err != nil || val == nil {
		return ""
	}
	return *val
}

// GetAllElements gets all elements matching a selector
func (p *PageHelper) GetAllElements(page *rod.Page, selector string) ([]*rod.Element, error) {
	elements, err := page.Elements(selector)
	if err != nil {
		return nil, err
	}
	return elements, nil
}

// WaitForNavigation waits for page navigation to complete
func (p *PageHelper) WaitForNavigation(page *rod.Page, timeout time.Duration) error {
	page = page.Timeout(timeout)
	defer page.CancelTimeout()

	// Wait for load event
	if err := page.WaitLoad(); err != nil {
		return err
	}

	// Wait for DOM to stabilize
	page.WaitDOMStable(time.Second, 0.1)

	return nil
}

// WaitForNetworkIdle waits until network is idle
func (p *PageHelper) WaitForNetworkIdle(page *rod.Page, timeout time.Duration) error {
	page = page.Timeout(timeout)
	defer page.CancelTimeout()

	wait := page.WaitRequestIdle(time.Second, nil, nil, nil)
	wait()

	return nil
}

// GetCurrentURL returns the current page URL
func (p *PageHelper) GetCurrentURL(page *rod.Page) string {
	info, err := page.Info()
	if err != nil {
		return ""
	}
	return info.URL
}

// ContainsText checks if the page contains specific text
func (p *PageHelper) ContainsText(page *rod.Page, text string) bool {
	html, err := page.HTML()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(html), strings.ToLower(text))
}

// FindElementByText finds an element containing specific text
func (p *PageHelper) FindElementByText(page *rod.Page, selector, text string) (*rod.Element, error) {
	elements, err := page.Elements(selector)
	if err != nil {
		return nil, err
	}

	for _, el := range elements {
		elText, err := el.Text()
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(elText), strings.ToLower(text)) {
			return el, nil
		}
	}

	return nil, fmt.Errorf("%w: no element with text '%s'", ErrElementNotFound, text)
}

// ClickElementByText finds and clicks an element by its text content
func (p *PageHelper) ClickElementByText(page *rod.Page, selector, text string) error {
	element, err := p.FindElementByText(page, selector, text)
	if err != nil {
		return err
	}

	return element.Click(proto.InputMouseButtonLeft, 1)
}

// WaitForURLContains waits until URL contains a specific string
func (p *PageHelper) WaitForURLContains(page *rod.Page, substring string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		url := p.GetCurrentURL(page)
		if strings.Contains(url, substring) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("%w: URL did not contain '%s'", ErrTimeout, substring)
}

// EvaluateJS runs JavaScript on the page and returns the result
func (p *PageHelper) EvaluateJS(page *rod.Page, js string) (interface{}, error) {
	result, err := page.Eval(js)
	if err != nil {
		return nil, err
	}
	return result.Value.Val(), nil
}

// ScrollToElement scrolls the page to make an element visible
func (p *PageHelper) ScrollToElement(page *rod.Page, element *rod.Element) error {
	return element.ScrollIntoView()
}

// HandleAlert handles JavaScript alert dialogs
func (p *PageHelper) HandleAlert(page *rod.Page, accept bool) {
	go page.HandleDialog()
}
