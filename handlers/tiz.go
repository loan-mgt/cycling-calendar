package handlers

import (
	"cpe/calendar/ical"
	"cpe/calendar/logger"
	"cpe/calendar/request"
	"cpe/calendar/types"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var allowedTizCategory = []string{
	// Tiz cycling categories
	"WE", "ME", "track", "MTB",
	"NC", "JR", "WC",
	"Women Elite", "Men Elite", "Women", "Men",
}

// GenerateTizICSHandler generates ICS file and sends it in response
func GenerateTizICSHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info().Msg("Generating ICS from Tiz endpoint")

	filename := "cycling-calendar.ics"
	calendarName := "Cycling Calendar"

	// Fetch data from Tiz endpoint
	tizRaces, err := request.GetTizRaces()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to fetch Tiz data")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	// Parse 'class' query parameters into a slice
	requestClasses := r.URL.Query()["class"]
	// If no classes specified, we return all races (empty slice implies no filter in filterTizRaces)

	logger.Log.Info().
		Strs("classes", requestClasses).
		Msg("Received class filters")

	// Check if request classes are in allowed list
	for _, c := range requestClasses {
		if !contains(allowedTizCategory, c) {
			logger.Log.Error().
				Str("class", c).
				Msg("Class is not allowed")
			http.Error(w, "Class is not allowed", http.StatusBadRequest)
			return
		}
	}

	// Filter Tiz races by categories
	filteredRaces := filterTizRaces(tizRaces, requestClasses)

	logger.Log.Info().
		Int("filteredRacesCount", len(filteredRaces)).
		Msg("Filtered Tiz races successfully")

	// Convert Tiz races to Events
	events := convertTizRacesToEvents(filteredRaces)

	// Generate iCal file
	icsContent := ical.GenerateTizICS(events, calendarName)

	// Set headers and write content
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Write([]byte(icsContent))
}

// filterTizRaces filters Tiz races by requested categories
func filterTizRaces(races []types.TizRace, categories []string) []types.TizRace {
	if len(categories) == 0 {
		return races
	}

	var filtered []types.TizRace
	for _, race := range races {
		if raceMatchesCategories(race.Categories, categories) {
			filtered = append(filtered, race)
		}
	}

	return filtered
}

// raceMatchesCategories checks if race has any of the requested categories
func raceMatchesCategories(raceCategories []string, requestedCategories []string) bool {
	for _, reqCat := range requestedCategories {
		for _, raceCat := range raceCategories {
			if strings.EqualFold(raceCat, reqCat) {
				return true
			}
		}
	}
	return false
}

// convertTizRacesToEvents converts Tiz races to Events
func convertTizRacesToEvents(tizRaces []types.TizRace) []types.Event {
	var events []types.Event

	for _, tizRace := range tizRaces {
		event := types.Event{
			Date:        tizRace.StartDate,
			Title:       tizRace.Name,
			Stage:       tizRace.Stage,
			Country:     tizRace.Country,
			CountryFlag: tizRace.CountryFlag,
			StreamType:  tizRace.StreamType,
			StreamLinks: tizRace.StreamLinks,
			StreamLang:  tizRace.StreamLang,
			Notes:       tizRace.Notes,
			Categories:  tizRace.Categories,
			StartDate:   tizRace.StartDate,
			EndDate:     tizRace.EndDate,
			Duration:    tizRace.Duration,
			AllDay:      tizRace.AllDay,
			Times:       tizRace.Times,
		}

		// Parse times
		if tizRace.AllDay {
			event.StartTime = ""
			event.EndTime = ""
		} else if len(tizRace.Times) > 0 {
			// Use first time slot as default
			event.StartTime = tizRace.Times[0].Time
			// Calculate end time using duration
			event.EndTime = calculateEndTime(tizRace.Times[0].Time, tizRace.Duration)
		}

		// Set info link (first non-stream link)
		if len(tizRace.StreamLinks) > 0 {
			// Use last link that's not a stream page
			for _, link := range tizRace.StreamLinks {
				if !strings.Contains(link, "stream") && !strings.Contains(link, "cyclingtiz") {
					event.Link = link
					break
				}
			}
			if event.Link == "" && len(tizRace.StreamLinks) > 0 {
				event.Link = tizRace.StreamLinks[0]
			}
		}

		events = append(events, event)
	}

	return events
}

// calculateEndTime calculates end time based on start time and duration
func calculateEndTime(startTimeStr, duration string) string {
	if startTimeStr == "" || duration == "" {
		return ""
	}

	// Parse duration
	durationMins := parseDurationMinutes(duration)
	if durationMins == 0 {
		// Default to +3 hours
		return startTimeStr // Will be handled in ICS generation
	}

	// Parse start time - remove "UTC" suffix if present
	cleanTime := strings.TrimSuffix(strings.TrimSpace(startTimeStr), "UTC")
	cleanTime = strings.TrimSpace(cleanTime)
	startParts := strings.Split(cleanTime, ":")
	if len(startParts) != 2 {
		return ""
	}

	hour, err := strconv.Atoi(strings.TrimSpace(startParts[0]))
	if err != nil {
		return ""
	}
	minute, err := strconv.Atoi(strings.TrimSpace(startParts[1]))
	if err != nil {
		return ""
	}

	// Calculate end time
	totalMins := hour*60 + minute + durationMins
	endHour := (totalMins / 60) % 24
	endMin := totalMins % 60

	return fmt.Sprintf("%02d:%02d UTC", endHour, endMin)
}

// parseDurationMinutes converts duration string to minutes
func parseDurationMinutes(duration string) int {
	// Pattern: "90 mins", "2 hrs", "3.25 hrs"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mins?|hrs?|hours?)`)
	matches := re.FindStringSubmatch(duration)
	if len(matches) < 3 {
		return 0
	}

	valueStr := matches[1]
	unit := matches[2]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

	switch strings.ToLower(unit) {
	case "min", "mins":
		return int(value)
	case "hr", "hrs", "hour", "hours":
		return int(value * 60)
	default:
		return 0
	}
}

// Health is a simple health check handler
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
