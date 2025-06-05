package request

import (
	"compress/gzip"
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func GetInfo(timezone string) ([]types.Event, error) {
	// Set up the request
	baseURL := "https://www.procyclingstats.com/races.php?filter=Filter&p=uci&s=start-finish-schedule"
	url := fmt.Sprintf("%s&timezone=%s", baseURL, timezone)

	logger.Log.Info().Str("timezone", timezone).Msg("Fetching calendar data")
	logger.Log.Debug().Str("url", url).Msg("Generated final URL")

	// Create and send request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("User-Agent", "Dalvik/2.1.0 (Linux; U; Android 15; sdk_gphone64_x86_64 Build/AE3A.240806.005)")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Connection", "Keep-Alive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error().Str("url", url).Int("statusCode", resp.StatusCode).Msg("Non-200 response")
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Handle gzip if needed
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			logger.Log.Error().Str("url", url).Err(err).Msg("Failed to create gzip reader")
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.(*gzip.Reader).Close()
	}

	// Parse HTML and extract events
	doc, err := html.Parse(reader)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to parse HTML")
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	events, err := extractEventsFromHTML(doc)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to extract events")
		return nil, fmt.Errorf("failed to extract events: %w", err)
	}

	logger.Log.Info().Str("timezone", timezone).Int("eventCount", len(events)).Msg("Calendar data fetched successfully")
	return events, nil
}

func GetAllowedRace(category string) ([]string, error) {
	baseUrl := "https://www.procyclingstats.com/races.php?year=2025&circuit=1&class=&filter=Filter&s=upcoming"
	url := fmt.Sprintf("%s&%s", baseUrl, category)

	logger.Log.Info().Str("category", category).Msg("Fetching allowed races")
	logger.Log.Debug().Str("url", url).Msg("Generated final URL")

	// Create and send request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("User-Agent", "Dalvik/2.1.0 (Linux; U; Android 15; sdk_gphone64_x86_64 Build/AE3A.240806.005)")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Connection", "Keep-Alive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error().Str("url", url).Int("statusCode", resp.StatusCode).Msg("Non-200 response")
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Handle gzip if needed
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			logger.Log.Error().Str("url", url).Err(err).Msg("Failed to create gzip reader")
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.(*gzip.Reader).Close()
	}

	// Parse HTML and extract events
	doc, err := html.Parse(reader)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to parse HTML")
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	races, err := extractRacesFromHTML(doc)
	if err != nil {
		logger.Log.Error().Str("url", url).Err(err).Msg("Failed to extract races")
		return nil, fmt.Errorf("failed to extract races: %w", err)
	}

	logger.Log.Info().Str("category", category).Int("raceCount", len(races)).Msg("Allowed races fetched successfully")
	return races, nil
}

func extractRacesFromHTML(doc *html.Node) ([]string, error) {
	var races []string

	table := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "table" && hasClass(n, "basic")
	})
	if table == nil {
		return nil, fmt.Errorf("table with class 'basic' not found")
	}

	tbody := findNode(table, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "tbody"
	})
	if tbody == nil {
		return nil, fmt.Errorf("tbody not found in table")
	}

	for tr := tbody.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}

		tdCount := 0
		for td := tr.FirstChild; td != nil; td = td.NextSibling {
			if td.Type != html.ElementNode || td.Data != "td" {
				continue
			}

			if tdCount == 2 { // second column (index 1)
				// Try to find anchor tag, or grab raw text
				var eventName string
				for c := td.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "a" && c.FirstChild != nil {
						eventName = c.FirstChild.Data
						break
					} else if c.Type == html.TextNode {
						eventName = c.Data
					}
				}
				eventName = strings.TrimSpace(eventName)
				if eventName != "" {
					races = append(races, eventName)
				}
				break
			}

			tdCount++
		}
	}

	return races, nil
}

// extractEventsFromHTML parses the HTML and extracts events from the table
func extractEventsFromHTML(doc *html.Node) ([]types.Event, error) {
	// Find table rows directly with a CSS-like selector approach
	var events []types.Event

	// First find the table.basic element
	table := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode &&
			n.Data == "table" &&
			hasClass(n, "basic")
	})

	if table == nil {
		return nil, fmt.Errorf("table with class 'basic' not found")
	}

	// Then find the tbody within that table
	tbody := findNode(table, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "tbody"
	})

	if tbody == nil {
		return nil, fmt.Errorf("tbody not found in table")
	}

	// Process all rows in the tbody
	for tr := tbody.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}

		// Collect all td elements in this row
		var cells []*html.Node
		for td := tr.FirstChild; td != nil; td = td.NextSibling {
			if td.Type == html.ElementNode && td.Data == "td" {
				cells = append(cells, td)
			}
		}

		if len(cells) < 5 {
			continue
		}

		// Extract data from cells
		date := extractTextContent(cells[0])
		raceInfo := extractTextContent(cells[2])
		startTime := extractTextContent(cells[3])
		endTime := extractTextContent(cells[4])

		// Parse title and stage info
		title, stage := parseRaceInfo(raceInfo)

		// Handle empty start or end times
		if startTime == "-" {
			continue
		}

		events = append(events, types.Event{
			Date:      date,
			Title:     title,
			Stage:     stage,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return events, nil
}

// Helper functions
func parseRaceInfo(raceInfo string) (title, stage string) {
	parts := strings.Split(raceInfo, "|")
	title = strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		stage = strings.TrimSpace(parts[1])
	}

	return title, stage
}

func extractTextContent(n *html.Node) string {
	var buf strings.Builder
	extractText(n, &buf)
	return strings.TrimSpace(buf.String())
}

func extractText(n *html.Node, buf *strings.Builder) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, buf)
	}
}

func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, className) {
			return true
		}
	}
	return false
}

func findNode(n *html.Node, predicate func(*html.Node) bool) *html.Node {
	if predicate(n) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, predicate); found != nil {
			return found
		}
	}

	return nil
}
