package request

import (
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	tizURL    = "https://cyclingtiz.live/sys-parse.php?file=db/races.txt"
	userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0"
)

var (
	raceCache struct {
		sync.RWMutex
		Races     []types.TizRace
		LastFetch time.Time
	}
)

// GetTizRaces fetches race data from Tiz-cycling endpoint with 24h caching
func GetTizRaces() ([]types.TizRace, error) {
	// Check cache
	raceCache.RLock()
	if !raceCache.LastFetch.IsZero() && time.Since(raceCache.LastFetch) < 24*time.Hour {
		logger.Log.Info().Msg("Returning cached Tiz race data")
		races := make([]types.TizRace, len(raceCache.Races))
		copy(races, raceCache.Races)
		raceCache.RUnlock()
		return races, nil
	}
	raceCache.RUnlock()

	logger.Log.Info().Msg("Fetching Tiz race data (cache miss or expired)")

	req, err := http.NewRequest("GET", tizURL, nil)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from curl command
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Removed Accept-Encoding to let http.Transport handle it
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://cyclingtiz.live/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Log.Error().Int("statusCode", resp.StatusCode).Msg("Non-200 response")
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to read response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logger.Log.Info().Int("bodyLength", len(body)).Msg("Response body length")
	if len(body) > 100 {
		logger.Log.Debug().Str("bodyStart", string(body[:100])).Msg("Body start")
	} else {
		logger.Log.Debug().Str("body", string(body)).Msg("Body content")
	}

	htmlContent := string(body)
	races, err := parseTizRaces(htmlContent)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to parse races")
		return nil, fmt.Errorf("failed to parse races: %w", err)
	}

	logger.Log.Info().Int("raceCount", len(races)).Msg("Tiz races fetched successfully")

	// Update cache
	raceCache.Lock()
	raceCache.Races = races
	raceCache.LastFetch = time.Now()
	raceCache.Unlock()

	return races, nil
}

// parseTizRaces parses HTML content and extracts race information
func parseTizRaces(htmlContent string) ([]types.TizRace, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var races []types.TizRace
	var currentSection string
	var todayDate string

	// Helper functions for HTML traversall LI elements
	var findAllLi func(*html.Node) []*html.Node
	findAllLi = func(n *html.Node) []*html.Node {
		var lis []*html.Node
		if n.Type == html.ElementNode && n.Data == "li" {
			lis = append(lis, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			lis = append(lis, findAllLi(c)...)
		}
		return lis
	}

	// Find all <li> elements
	allLis := findAllLi(doc)
	logger.Log.Info().Int("totalLisFound", len(allLis)).Msg("Found LI elements in document")

	for _, li := range allLis {
		liText, _ := extractText(li)

		// Check for section headers
		if strings.Contains(liText, "TODAY") || strings.Contains(liText, "TOMORROW") || strings.Contains(liText, "UPCOMING") {
			logger.Log.Debug().Str("section", liText).Msg("Found section header")
			if strings.Contains(liText, "TODAY") {
				currentSection = parseDateFromHeader(liText)
				todayDate = currentSection
			} else if strings.Contains(liText, "TOMORROW") {
				date := parseDateFromHeader(liText)
				if date != "" {
					currentSection = date
				} else if todayDate != "" {
					// Calculate tomorrow from today
					t, err := time.Parse("2006-01-02", todayDate)
					if err == nil {
						currentSection = t.AddDate(0, 0, 1).Format("2006-01-02")
					} else {
						currentSection = "TOMORROW"
					}
				} else {
					currentSection = "TOMORROW"
				}
			} else if strings.Contains(liText, "UPCOMING") {
				currentSection = "UPCOMING"
			}
			continue
		}

		// Check if this <li> has an image (flag) - indicates a race entry
		img := findNode(li, isImgElement)
		if img == nil {
			// Only log failure if we think it might be a race (has some length)
			if len(liText) > 20 {
				logger.Log.Debug().Str("text", liText).Msg("No image found in LI, skipping")
			}
			continue
		}

		logger.Log.Debug().Str("section", currentSection).Msg("Attempting to parse race from LI")

		logger.Log.Debug().Str("section", currentSection).Msg("Attempting to parse race from LI")

		// Parse race from this <li>
		race, err := parseRaceFromLi(li, currentSection)
		if err != nil {
			logger.Log.Debug().Err(err).Msg("Failed to parse race from li")
			continue
		}

		if race.Name != "" {
			logger.Log.Debug().Str("name", race.Name).Msg("Successfully parsed race")
			races = append(races, race)
		} else {
			logger.Log.Warn().Str("text", liText).Msg("Parsed race but name is empty")
		}
	}

	logger.Log.Info().Int("racesFound", len(races)).Msg("Finished parsing Tiz races")

	return races, nil
}

// parseRaceFromLi extracts race data from a single <li> element
func parseRaceFromLi(li *html.Node, sectionDate string) (types.TizRace, error) {
	race := types.TizRace{}

	// Extract flag and country
	img := findNode(li, isImgElement)
	if img != nil {
		src := getAttribute(img, "src")
		if src != "" {
			race.Country = extractCountryFromFlag(src)
			race.CountryFlag = src
		}
	}

	// Parse text content for race info
	text, _ := extractText(li)

	// Parse dates
	race.StartDate, race.EndDate = parseDatesFromText(text)

	// Fallback to section date if specific date not found
	if race.StartDate == "" && sectionDate != "" && sectionDate != "UPCOMING" {
		race.StartDate = sectionDate
		race.EndDate = sectionDate
	}

	race.AllDay = strings.Contains(text, "times TBA") || strings.Contains(text, "time TBA")

	// Parse categories
	race.Categories = extractCategories(text)

	// Parse stream type
	if strings.Contains(text, "POSSIBLE LIVE") {
		race.StreamType = "POSSIBLE LIVE"
	} else if strings.Contains(text, "PROBABLE LIVE") {
		race.StreamType = "PROBABLE LIVE"
	} else if strings.Contains(text, "LIVE") {
		race.StreamType = "LIVE"
	} else if strings.Contains(text, "RECORDED") {
		race.StreamType = "RECORDED"
	}

	// Parse stream links
	race.StreamLinks = extractLinks(li)
	if len(race.StreamLinks) > 0 {
		// Determine language from text
		if strings.Contains(text, "(English or Spanish)") {
			race.StreamLang = "English or Spanish"
		} else if strings.Contains(text, "(Spanish)") {
			race.StreamLang = "Spanish"
		} else if strings.Contains(text, "(Slovenian)") {
			race.StreamLang = "Slovenian"
		} else if strings.Contains(text, "(Flemish)") {
			race.StreamLang = "Flemish"
		} else if strings.Contains(text, "(Arabic)") {
			race.StreamLang = "Arabic"
		}
	}

	// Parse notes
	race.Notes = extractNotes(text)

	// Parse times
	race.Times = extractTimes(text)

	// Calculate duration from times if not already set
	if race.Duration == "" && len(race.Times) > 0 {
		race.Duration = calculateDurationFromTimes(race.Times)
	}

	// Parse name and stage
	race.Name, race.Stage = parseNameAndStage(text)

	return race, nil
}

// parseDateFromHeader parses a date from section header like "TODAY Wednesday 4th February"
func parseDateFromHeader(header string) string {
	// Extract "Wednesday 4th February"
	// Replace known prefixes/suffixes with empty string
	header = strings.Replace(header, "TODAY", "", -1)
	header = strings.Replace(header, "TOMORROW", "", -1)
	header = strings.Replace(header, "& ONGOING", "", -1)
	header = strings.TrimSpace(header)

	// header should now be "Wednesday 4th February"
	// Example: "Wednesday 4th February"
	parts := strings.Fields(header)
	if len(parts) >= 3 {
		dayStr := parts[1]
		monthStr := parts[2]

		// Remove st, nd, rd, th from day
		dayStr = strings.TrimSuffix(dayStr, "st")
		dayStr = strings.TrimSuffix(dayStr, "nd")
		dayStr = strings.TrimSuffix(dayStr, "rd")
		dayStr = strings.TrimSuffix(dayStr, "th")

		day, _ := strconv.Atoi(dayStr)

		// Parse month
		date, err := time.Parse("January", monthStr)
		if err != nil {
			logger.Log.Debug().Str("header", header).Str("month", monthStr).Err(err).Msg("Failed to parse month in header")
			return ""
		}

		// Construct date for current year
		// Note: This assumes current year. For end of year/beginning of year logic we might need improvement
		currentYear := time.Now().Year()
		fullDate := time.Date(currentYear, date.Month(), day, 0, 0, 0, 0, time.UTC)

		logger.Log.Debug().Str("header", header).Str("parsed", fullDate.Format("2006-01-02")).Msg("Parsed date from header")
		return fullDate.Format("2006-01-02")
	}

	logger.Log.Debug().Str("header", header).Str("parts_count", fmt.Sprintf("%d", len(parts))).Msg("Failed to parse date from header (parts < 3)")
	return ""
}

// extractCountryFromFlag extracts country code from flag URL
func extractCountryFromFlag(src string) string {
	// Pattern: https://tiz-cycling.io/flags/B_be.png -> BE
	// Pattern: https://flagpedia.net/data/flags/w580/fr.png -> FR
	re := regexp.MustCompile(`([A-Z_]+|[a-z]{2})\.(png|webp)`)
	matches := re.FindStringSubmatch(src)
	if len(matches) < 2 {
		return ""
	}

	code := matches[1]
	// Handle special cases like B_be, E_es, UAE_ae, SLO_si
	if strings.Contains(code, "_") {
		parts := strings.Split(code, "_")
		code = parts[len(parts)-1] // Take last part
	}

	return strings.ToUpper(code[:2])
}

// parseDatesFromText extracts start and end dates from text
func parseDatesFromText(text string) (startDate, endDate string) {
	startDate = ""
	endDate = ""

	// Pattern for explicit date: "Friday 6th February"
	datePattern := regexp.MustCompile(`(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\s+(\d+)(?:st|nd|rd|th)\s+(January|February|March|April|May|June|July|August|September|October|November|December)`)
	dateMatch := datePattern.FindString(text)

	if dateMatch != "" {
		year := time.Now().Year()
		startDate = parseDate(dateMatch, year)
	}

	// Pattern for multi-day: "for 3 days", "for 5 days"
	durationPattern := regexp.MustCompile(`for\s+(\d+)\s+days?`)
	durationMatch := durationPattern.FindString(text)

	if durationMatch != "" && startDate != "" {
		durationDays, _ := strconv.Atoi(durationMatch)
		if durationDays > 0 {
			startTime, _ := time.Parse("2006-01-02", startDate)
			endTime := startTime.AddDate(0, 0, durationDays-1)
			endDate = endTime.Format("2006-01-02")
		}
	}

	return startDate, endDate
}

// parseDate converts "Friday 6th February" to "2026-02-06"
func parseDate(dateStr string, year int) string {
	monthMap := map[string]int{
		"January": 1, "February": 2, "March": 3, "April": 4,
		"May": 5, "June": 6, "July": 7,
		"August": 8, "September": 9, "October": 10,
		"November": 11, "December": 12,
	}

	re := regexp.MustCompile(`(\d+)(?:st|nd|rd|th)\s+(\w+)`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) < 3 {
		return ""
	}

	day, _ := strconv.Atoi(matches[1])
	monthName := matches[2]

	month, ok := monthMap[monthName]
	if !ok {
		return ""
	}

	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// extractCategories extracts race categories from text
func extractCategories(text string) []string {
	var categories []string

	// Find content inside parentheses
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(text, -1)

	validCategories := map[string]bool{
		"WE":          true,
		"ME":          true,
		"track":       true,
		"MTB":         true,
		"NC":          true,
		"JR":          true,
		"WC":          true,
		"Elite":       true,
		"Women Elite": true,
		"Men Elite":   true,
		"Women":       true,
		"Men":         true,
	}

	for _, match := range matches {
		if len(match) > 1 {
			// Split by comma
			parts := strings.Split(match[1], ",")
			for _, part := range parts {
				cat := strings.TrimSpace(part)
				if validCategories[cat] {
					if !containsString(categories, cat) {
						categories = append(categories, cat)
					}
				}
			}
		}
	}

	// Also check for standalone category mentions (legacy check or for unparenthesized ones)
	if strings.Contains(text, "Women Elite") && !containsString(categories, "Women Elite") {
		categories = append(categories, "Women Elite")
	}
	if strings.Contains(text, "Men Elite") && !containsString(categories, "Men Elite") {
		categories = append(categories, "Men Elite")
	}

	return categories
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractLinks extracts all href links from <li> element
func extractLinks(li *html.Node) []string {
	var links []string

	for a := findNode(li, isAElement); a != nil; a = findNode(a.NextSibling, isAElement) {
		href := getAttribute(a, "href")
		if href != "" && !strings.HasPrefix(href, "javascript:") {
			// Clean URL if needed
			if strings.HasPrefix(href, "//") {
				href = "https:" + href
			}
			links = append(links, href)
		}
	}

	return links
}

// extractNotes extracts text from <em> tags
func extractNotes(text string) string {
	re := regexp.MustCompile(`<em>([^<]+)</em>`)
	matches := re.FindAllStringSubmatch(text, -1)

	var notes []string
	for _, match := range matches {
		if len(match) > 1 {
			note := strings.TrimSpace(match[1])
			if note != "" {
				notes = append(notes, note)
			}
		}
	}

	return strings.Join(notes, " | ")
}

// extractTimes extracts time slots from text
func extractTimes(text string) []types.TizTimeSlot {
	var timeSlots []types.TizTimeSlot

	// Pattern for WE and ME times: "WE 12.40 UTC (60 mins) - ME 14.00 UTC (90 mins)"
	timePattern := regexp.MustCompile(`(WE|ME)\s+(\d+\.\d+)\s+UTC\s+\(([^)]+)\)`)
	matches := timePattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			timeSlots = append(timeSlots, types.TizTimeSlot{
				Category: match[1],
				Time:     fmt.Sprintf("%s:00 UTC", strings.Replace(match[2], ".", ":", 1)),
				Duration: match[3],
			})
		}
	}

	// Pattern for single time: "14.45 UTC (90 mins)"
	singleTimePattern := regexp.MustCompile(`(\d+\.\d+)\s+UTC\s+\(([^)]+)\)`)
	singleMatch := singleTimePattern.FindStringSubmatch(text)

	if singleMatch != nil && len(singleMatch) >= 2 && len(timeSlots) == 0 {
		timeSlots = append(timeSlots, types.TizTimeSlot{
			Category: "",
			Time:     fmt.Sprintf("%s:00 UTC", strings.Replace(singleMatch[1], ".", ":", 1)),
			Duration: singleMatch[2],
		})
	}

	return timeSlots
}

// calculateDurationFromTimes calculates duration from time slots
func calculateDurationFromTimes(times []types.TizTimeSlot) string {
	if len(times) == 0 {
		return ""
	}

	// Use duration from first time slot for display
	return times[0].Duration
}

// parseNameAndStage extracts race name and stage information
func parseNameAndStage(text string) (name, stage string) {
	// Remove stream info and other noise
	name = text

	// Remove stream info
	re := regexp.MustCompile(`\s*-\s*LIVE\s*[^-]*`)
	name = re.ReplaceAllString(name, "")

	// Remove POSSIBLE LIVE
	re = regexp.MustCompile(`\s*-\s*POSSIBLE LIVE\s*[^-]*`)
	name = re.ReplaceAllString(name, "")

	// Remove PROBABLE LIVE
	re = regexp.MustCompile(`\s*-\s*PROBABLE LIVE\s*[^-]*`)
	name = re.ReplaceAllString(name, "")

	// Remove stream page links
	re = regexp.MustCompile(`\s*-\s*<strong>Stream Page</strong>\s*[^-]*`)
	name = re.ReplaceAllString(name, "")

	// Remove direct links
	re = regexp.MustCompile(`\s*-\s*<strong><a[^>]+>Link</a></strong>\s*\([^)]*\)\s*[^-]*`)
	name = re.ReplaceAllString(name, "")

	// Remove date prefixes
	re = regexp.MustCompile(`^(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\s+\d+(?:st|nd|rd|th)\s+\w+\s+for\s+\d+\s+days?\s*-\s*`)
	name = re.ReplaceAllString(name, "")

	re = regexp.MustCompile(`^(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\s+\d+(?:st|nd|rd|th)\s+\w+\s*-\s*`)
	name = re.ReplaceAllString(name, "")

	// Remove Info links at the end
	re = regexp.MustCompile(`\s*-\s*Info\s*$`)
	name = re.ReplaceAllString(name, "")

	re = regexp.MustCompile(`\s*-\s*<strong><a[^>]+>Info</a></strong>\s*$`)
	name = re.ReplaceAllString(name, "")

	// Remove time info with Category prefix (e.g. - WE 12.40 UTC ...)
	re = regexp.MustCompile(`\s*-\s*(WE|ME|track|MTB)\s+\d+(?:[\.:]\d+)?\s+UTC.*`)
	name = re.ReplaceAllString(name, "")

	// Remove time info (e.g. - 12.40 UTC ...)
	re = regexp.MustCompile(`\s*-\s*\d+(?:[\.:]\d+)?\s+UTC.*`)
	name = re.ReplaceAllString(name, "")

	// Remove categories (WE, ME, etc.) anywhere in the name if in parentheses
	// We matched (WE, ME), (WE), (ME), etc.
	re = regexp.MustCompile(`\s*\((?:WE|ME|track|MTB|NC|JR|WC|Elite|Women Elite|Men Elite|Women|Men)(?:,\s*(?:WE|ME|track|MTB|NC|JR|WC|Elite|Women Elite|Men Elite|Women|Men))*\)\s*`)
	name = re.ReplaceAllString(name, "")

	// Remove trailing dashes and spaces
	name = strings.Trim(name, "- ")
	name = strings.TrimSpace(name)

	// Extract stage if present
	stagePattern := regexp.MustCompile(`stage\s+(\d+)\s*\(of\s+(\d+)\)`)
	stageMatch := stagePattern.FindStringSubmatch(text)

	if len(stageMatch) >= 3 {
		stage = fmt.Sprintf("stage %s (of %s)", stageMatch[1], stageMatch[2])
	}

	dayPattern := regexp.MustCompile(`day\s+(\d+)\s*\(of\s+(\d+)\)`)
	dayMatch := dayPattern.FindStringSubmatch(text)

	if len(dayMatch) >= 3 {
		stage = fmt.Sprintf("day %s (of %s)", dayMatch[1], dayMatch[2])
	}

	return name, stage
}

// Helper functions for HTML traversal - using standard library functions
func isLiElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "li"
}

func isImgElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "img"
}

func isAElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "a"
}

func getAttribute(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

func extractText(n *html.Node) (text, link string) {
	if n.Type == html.TextNode {
		// Preserve whitespace for accurate date parsing
		return n.Data, href(n)
	}
	buf := ""
	link = href(n)

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf2, link2 := extractText(c)
		buf += buf2
		link += link2
	}

	// We only trim the final result, not intermediate steps to avoid merging words
	return strings.TrimSpace(buf), link
}

func href(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				return attr.Val
			}
		}
	}

	return ""
}

func findNode(n *html.Node, predicate func(*html.Node) bool) *html.Node {
	if n == nil {
		return nil
	}
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
