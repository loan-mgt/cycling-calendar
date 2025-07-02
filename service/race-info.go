package service

import (
	"cpe/calendar/logger"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"cpe/calendar/types"
)

// RaceInfo represents the race information and the time it was fetched.
type RaceInfoTime struct {
	Data      types.RaceInfo
	Timestamp int64
}

// RaceInfoService manages RaceInfo caching and retrieval.
type RaceInfoService struct {
	mu    sync.Mutex
	cache map[string]*RaceInfoTime
	stop  chan struct{}
}

// NewRaceInfoService creates a new RaceInfoService.
func NewRaceInfoService() *RaceInfoService {
	service := &RaceInfoService{
		cache: make(map[string]*RaceInfoTime),
		stop:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go service.cleanupExpired()

	return service
}

// GetRaceInfo returns RaceInfo for a URL, fetching if not cached or outdated.
func (s *RaceInfoService) GetRaceInfo(url string, fetchFunc func(string) (*types.RaceInfo, error)) (*types.RaceInfo, error) {

	key := strings.TrimSuffix(url, "/results")

	s.mu.Lock()
	info, exists := s.cache[key]
	s.mu.Unlock()

	now := time.Now().Unix()
	if exists && now-info.Timestamp < 24*3600 {
		return &info.Data, nil
	}

	// Fetch new info
	newData, err := fetchFunc(url)
	if err != nil {
		return nil, err
	}

	newInfo := &RaceInfoTime{
		Data:      *newData,
		Timestamp: now,
	}

	s.mu.Lock()
	s.cache[key] = newInfo
	s.mu.Unlock()

	return newData, nil
}

// Example fetch function (to be implemented elsewhere)
func FetchRaceInfo(url string) (*types.RaceInfo, error) {
	// Example: make HTTP request and parse response
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response into RaceInfo.Data as needed
	return ParseRaceInfoFromResponse(resp)
}

func ParseRaceInfoFromResponse(resp *http.Response) (*types.RaceInfo, error) {
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	raceInfo := &types.RaceInfo{}

	// Extract all key-value pairs from the HTML
	values := extractRaceInfoValues(html)

	// Parse each field - keep everything as strings
	if val, exists := values["Datename"]; exists {
		raceInfo.Datename = extractTextFromHTML(val)
	}
	if val, exists := values["Start time"]; exists {
		raceInfo.StartTime = extractTextFromHTML(val)
	}
	if val, exists := values["Avg. speed winner"]; exists {
		raceInfo.AvgSpeedWinner = extractTextFromHTML(val)
	}
	if val, exists := values["Classification"]; exists {
		raceInfo.Classification = extractTextFromHTML(val)
	}
	if val, exists := values["Race category"]; exists {
		raceInfo.RaceCategory = extractTextFromHTML(val)
	}
	if val, exists := values["Distance"]; exists {
		raceInfo.DistanceKm = extractTextFromHTML(val)
	}
	if val, exists := values["Points scale"]; exists {
		raceInfo.PointsScale = extractTextFromHTML(val)
	}
	if val, exists := values["UCI scale"]; exists {
		raceInfo.UCIScale = extractTextFromHTML(val)
	}
	if val, exists := values["Parcours type"]; exists {
		raceInfo.ParcoursType = extractTextFromHTML(val)
	}
	if val, exists := values["ProfileScore"]; exists {
		raceInfo.ProfileScore = extractTextFromHTML(val)
	}
	if val, exists := values["Vertical meters"]; exists {
		raceInfo.VerticalMeters = extractTextFromHTML(val)
	}
	if val, exists := values["Departure"]; exists {
		raceInfo.Departure = extractTextFromHTML(val)
	}
	if val, exists := values["Arrival"]; exists {
		raceInfo.Arrival = extractTextFromHTML(val)
	}
	if val, exists := values["Race ranking"]; exists {
		raceInfo.RaceRanking = extractTextFromHTML(val)
	}
	if val, exists := values["Startlist quality score"]; exists {
		raceInfo.StartlistQualityScore = extractTextFromHTML(val)
	}
	if val, exists := values["Won how"]; exists {
		raceInfo.WonHow = extractTextFromHTML(val)
	}
	if val, exists := values["Avg. temperature"]; exists {
		raceInfo.AvgTemperature = extractTextFromHTML(val)
	}

	return raceInfo, nil
}

// extractRaceInfoValues extracts key-value pairs from the HTML
func extractRaceInfoValues(html string) map[string]string {
	values := make(map[string]string)

	// Pattern to match list items with title and value divs - updated to handle extra spaces and attributes
	pattern := `<li[^>]*>\s*<div class="title[^"]*"[^>]*>\s*([^:]+):\s*</div>\s*<div class="[^"]*value[^"]*"[^>]*>\s*(.*?)\s*</div>\s*</li>`
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(html, -1)

	// Log the number of matches found for debugging
	logger.Log.Info().
		Int("matchCount", len(matches)).
		Msg("Race info extraction matches found")

	for _, match := range matches {
		if len(match) >= 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])
			values[key] = value

			// Log each extracted key-value pair for debugging
			logger.Log.Debug().
				Str("key", key).
				Str("value", value).
				Msg("Extracted race info field")
		}
	}

	return values
}

// extractTextFromHTML extracts text content from HTML (removes tags)
func extractTextFromHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")
	return strings.TrimSpace(text)
}

// cleanupExpired runs a cleanup process every 24 hours to remove expired cache entries
func (s *RaceInfoService) cleanupExpired() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.removeExpiredEntries()
		case <-s.stop:
			return
		}
	}
}

// removeExpiredEntries removes expired cache entries
func (s *RaceInfoService) removeExpiredEntries() {
	now := time.Now().Unix()
	expirationTime := int64(24 * 3600) // 24 hours in seconds

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, info := range s.cache {
		if now-info.Timestamp >= expirationTime {
			delete(s.cache, key)
		}
	}
}

// Stop gracefully stops the cleanup goroutine
func (s *RaceInfoService) Stop() {
	close(s.stop)
}
