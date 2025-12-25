// Package search handles LinkedIn people search
package search

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/browser"
	"linkedin-automation/internal/config"
	"linkedin-automation/internal/models"
	"linkedin-automation/internal/stealth"
	"linkedin-automation/internal/storage"
)

// LinkedIn search URLs
const (
	LinkedInSearchURL = "https://www.linkedin.com/search/results/people/"
)

// Search result selectors
const (
	SelectorSearchResults    = ".search-results-container"
	SelectorResultItem       = ".entity-result"
	SelectorResultLink       = ".entity-result__title-text a"
	SelectorResultName       = ".entity-result__title-text a span[aria-hidden='true']"
	SelectorResultTitle      = ".entity-result__primary-subtitle"
	SelectorResultLocation   = ".entity-result__secondary-subtitle"
	SelectorNextButton       = "button[aria-label='Next']"
	SelectorNoResults        = ".search-reusable-search-no-results"
	SelectorResultsCount     = ".search-results-container h2"
)

// Searcher handles LinkedIn people search
type Searcher struct {
	browser      *browser.Browser
	pageHelper   *browser.PageHelper
	stealth      *stealth.Controller
	profileStore *storage.ProfileStore
	statsStore   *storage.StatsStore
	config       *config.SearchConfig
	logger       zerolog.Logger
}

// NewSearcher creates a new searcher
func NewSearcher(
	b *browser.Browser,
	profileStore *storage.ProfileStore,
	statsStore *storage.StatsStore,
	searchConfig *config.SearchConfig,
	stealthCtrl *stealth.Controller,
	logger zerolog.Logger,
) *Searcher {
	return &Searcher{
		browser:      b,
		pageHelper:   browser.NewPageHelper(logger),
		stealth:      stealthCtrl,
		profileStore: profileStore,
		statsStore:   statsStore,
		config:       searchConfig,
		logger:       logger.With().Str("component", "search").Logger(),
	}
}

// SearchResult holds a single search result
type SearchResult struct {
	ProfileURL string
	Name       string
	Title      string
	Location   string
}

// Search performs a LinkedIn people search
func (s *Searcher) Search(params models.SearchParams) ([]SearchResult, error) {
	s.logger.Info().
		Str("jobTitle", params.JobTitle).
		Str("company", params.Company).
		Str("location", params.Location).
		Msg("Starting search")

	// Build search URL
	searchURL := s.buildSearchURL(params)
	s.logger.Debug().Str("url", searchURL).Msg("Search URL built")

	// Get page
	page, err := s.browser.GetPage()
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	// Navigate to search
	if err := s.browser.Navigate(page, searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to search: %w", err)
	}

	// Wait for results
	s.stealth.Timing().PageLoadDelay()

	// Check for no results
	if s.pageHelper.ElementExists(page, SelectorNoResults) {
		s.logger.Info().Msg("No search results found")
		return []SearchResult{}, nil
	}

	// Collect results from all pages
	var allResults []SearchResult
	pageNum := 1

	for pageNum <= s.config.MaxPages {
		s.logger.Info().Int("page", pageNum).Msg("Processing search results page")

		// Simulate reading behavior
		s.stealth.SimulateReading(page)

		// Extract results from current page
		results, err := s.extractResultsFromPage(page)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to extract results from page")
		} else {
			allResults = append(allResults, results...)
			s.logger.Debug().
				Int("count", len(results)).
				Int("total", len(allResults)).
				Msg("Extracted results")
		}

		// Check for next page
		if !s.hasNextPage(page) {
			s.logger.Info().Msg("No more pages")
			break
		}

		// Go to next page
		if err := s.goToNextPage(page); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to go to next page")
			break
		}

		pageNum++

		// Delay between pages
		s.stealth.Timing().RandomDelay(3, 8)
	}

	s.logger.Info().
		Int("totalResults", len(allResults)).
		Int("pagesProcessed", pageNum).
		Msg("Search completed")

	return allResults, nil
}

// SearchAndSave searches and saves new profiles to database
func (s *Searcher) SearchAndSave(params models.SearchParams) (int, int, error) {
	results, err := s.Search(params)
	if err != nil {
		return 0, 0, err
	}

	newCount := 0
	duplicateCount := 0

	for _, result := range results {
		// Check if profile already exists
		exists, err := s.profileStore.Exists(result.ProfileURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", result.ProfileURL).Msg("Failed to check profile existence")
			continue
		}

		if exists {
			duplicateCount++
			continue
		}

		// Parse name
		firstName, lastName := parseName(result.Name)

		// Save profile
		profile := &models.Profile{
			URL:       result.ProfileURL,
			FirstName: firstName,
			LastName:  lastName,
			FullName:  result.Name,
			Title:     result.Title,
			Location:  result.Location,
			Status:    models.ProfileStatusFound,
		}

		if err := s.profileStore.Save(profile); err != nil {
			s.logger.Warn().Err(err).Str("url", result.ProfileURL).Msg("Failed to save profile")
			continue
		}

		newCount++
	}

	// Update stats
	s.statsStore.IncrementSearches(newCount)

	s.logger.Info().
		Int("new", newCount).
		Int("duplicates", duplicateCount).
		Msg("Profiles saved")

	return newCount, duplicateCount, nil
}

// buildSearchURL constructs the LinkedIn search URL with parameters
func (s *Searcher) buildSearchURL(params models.SearchParams) string {
	baseURL := LinkedInSearchURL
	queryParams := url.Values{}

	// Build keywords from all search terms
	var keywords []string

	if params.JobTitle != "" {
		keywords = append(keywords, params.JobTitle)
	}
	if params.Company != "" {
		keywords = append(keywords, params.Company)
	}
	if params.Location != "" {
		keywords = append(keywords, params.Location)
	}
	keywords = append(keywords, params.Keywords...)

	if len(keywords) > 0 {
		queryParams.Set("keywords", strings.Join(keywords, " "))
	}

	// Origin parameter (required for search to work)
	queryParams.Set("origin", "GLOBAL_SEARCH_HEADER")

	if len(queryParams) > 0 {
		return baseURL + "?" + queryParams.Encode()
	}

	return baseURL
}

// extractResultsFromPage extracts search results from the current page
func (s *Searcher) extractResultsFromPage(page *rod.Page) ([]SearchResult, error) {
	// Wait for results to load
	_, err := s.pageHelper.WaitForElement(page, SelectorSearchResults, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("search results container not found: %w", err)
	}

	// Get all result items
	items, err := page.Elements(SelectorResultItem)
	if err != nil {
		return nil, fmt.Errorf("failed to get result items: %w", err)
	}

	var results []SearchResult

	for _, item := range items {
		result, err := s.extractResultFromItem(item)
		if err != nil {
			s.logger.Debug().Err(err).Msg("Failed to extract result from item")
			continue
		}

		if result.ProfileURL != "" {
			results = append(results, result)
		}
	}

	return results, nil
}

// extractResultFromItem extracts a single result from a result item element
func (s *Searcher) extractResultFromItem(item *rod.Element) (SearchResult, error) {
	result := SearchResult{}

	// Get profile link
	linkEl, err := item.Element(SelectorResultLink)
	if err != nil {
		return result, fmt.Errorf("profile link not found")
	}

	href := s.pageHelper.GetElementAttribute(linkEl, "href")
	result.ProfileURL = normalizeProfileURL(href)

	// Get name
	nameEl, err := item.Element(SelectorResultName)
	if err == nil {
		result.Name = strings.TrimSpace(s.pageHelper.GetElementText(nameEl))
	}

	// Get title
	titleEl, err := item.Element(SelectorResultTitle)
	if err == nil {
		result.Title = strings.TrimSpace(s.pageHelper.GetElementText(titleEl))
	}

	// Get location
	locationEl, err := item.Element(SelectorResultLocation)
	if err == nil {
		result.Location = strings.TrimSpace(s.pageHelper.GetElementText(locationEl))
	}

	return result, nil
}

// hasNextPage checks if there's a next page button
func (s *Searcher) hasNextPage(page *rod.Page) bool {
	nextButton, err := page.Element(SelectorNextButton)
	if err != nil {
		return false
	}

	// Check if button is disabled
	disabled, err := nextButton.Attribute("disabled")
	if err == nil && disabled != nil {
		return false
	}

	return true
}

// goToNextPage clicks the next page button
func (s *Searcher) goToNextPage(page *rod.Page) error {
	nextButton, err := s.pageHelper.WaitForElement(page, SelectorNextButton, 5*time.Second)
	if err != nil {
		return err
	}

	// Scroll button into view
	s.stealth.Scroll().ScrollIntoView(page, nextButton)
	s.stealth.Timing().ActionDelay()

	// Click with stealth mouse movement
	if err := s.stealth.Mouse().ClickElement(page, nextButton); err != nil {
		return err
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)
	page.WaitDOMStable(time.Second, 0.1)

	return nil
}

// normalizeProfileURL cleans up a LinkedIn profile URL
func normalizeProfileURL(href string) string {
	if href == "" {
		return ""
	}

	// Parse URL
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Extract path (remove query params)
	path := u.Path

	// Remove trailing slashes
	path = strings.TrimSuffix(path, "/")

	// Ensure it's a profile URL
	if !strings.HasPrefix(path, "/in/") {
		return ""
	}

	return "https://www.linkedin.com" + path
}

// parseName splits a full name into first and last name
func parseName(fullName string) (string, string) {
	fullName = strings.TrimSpace(fullName)
	parts := strings.SplitN(fullName, " ", 2)

	firstName := ""
	lastName := ""

	if len(parts) > 0 {
		firstName = parts[0]
	}
	if len(parts) > 1 {
		lastName = parts[1]
	}

	// Remove any LinkedIn additions like "(He/Him)" or connection degree
	re := regexp.MustCompile(`\s*\([^)]*\)\s*$`)
	lastName = re.ReplaceAllString(lastName, "")

	// Remove "• 1st", "• 2nd", "• 3rd" etc.
	re = regexp.MustCompile(`\s*•\s*\d+(st|nd|rd|th)?\s*$`)
	lastName = re.ReplaceAllString(lastName, "")

	return strings.TrimSpace(firstName), strings.TrimSpace(lastName)
}
