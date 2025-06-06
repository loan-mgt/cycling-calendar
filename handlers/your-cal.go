package handlers

import (
	"cpe/calendar/ical"
	"cpe/calendar/logger"
	"cpe/calendar/request"
	"cpe/calendar/types"
	"net/http"
	"os"
	"strings"
)

var allowedCategory = []string{
	"",
	"1.1",
	"1.2",
	"1.2U",
	"1.Ncup",
	"1.Pro",
	"1.UWT",
	"1.WWT",
	"2.1",
	"2.2",
	"2.2U",
	"2.Ncup",
	"2.Pro",
	"2.UWT",
	"2.WWT",
	"CC",
	"JR",
	"NC",
	"WC",
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	logger.Log.Info().
		Msg("Health check endpoint hit, status OK")
}

// GenerateICSHandler generates the ICS file and sends it in the response with a given filename
func GenerateICSHandler(w http.ResponseWriter, r *http.Request) {
	// Get start and end times from environment variables
	timezone := os.Getenv("TIMEZONE")

	logger.Log.Info().
		Str("timezone", timezone).
		Msg("Using environment variables for timezone")

	filename := "cycling-calendar.ics"
	calendarName := "Cycling Calendar"

	// Fetch data from the source
	events, err := request.GetInfo(timezone)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to fetch data")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	// Parse 'class' query parameters into a slice
	requestClasses := r.URL.Query()["class"]
	if len(requestClasses) == 0 || (len(requestClasses) == 1 && requestClasses[0] == "") {
		requestClasses = []string{"2.UWT", "1.UWT"}
	}

	// Optional: log or use the values
	logger.Log.Info().
		Strs("classes", requestClasses).
		Msg("Received class filters")

	// Check if the request classes are in the allowed list
	for _, c := range requestClasses {
		if !contains(allowedCategory, c) {
			logger.Log.Error().
				Str("class", c).
				Msg("Class is not allowed")
			http.Error(w, "Class is not allowed", http.StatusBadRequest)
			return
		}
	}

	// Get allowed events
	allowedEvents := make([]string, 0)
	for _, category := range requestClasses {
		e, err := request.GetAllowedRace(category)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("category", category).
				Msg("Failed to fetch allowed races")
			http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
			return
		}
		allowedEvents = append(allowedEvents, e...)
	}

	logger.Log.Info().
		Int("eventsCount", len(events)).
		Int("allowedEventsCount", len(allowedEvents)).
		Msg("Fetched events and allowed events successfully")

	// Build a set of allowed event names for fast lookup
	allowedSet := make(map[string]struct{})
	for _, name := range allowedEvents {
		allowedSet[strings.ToLower(name)] = struct{}{}
	}

	// Filter events by allowed list
	var filteredEvents []types.Event
	for _, e := range events {
		if _, ok := allowedSet[strings.ToLower(e.Title)]; ok {
			filteredEvents = append(filteredEvents, e)
		}
	}

	logger.Log.Info().
		Strs("allowedEvents", allowedEvents).
		Msg("List of allowed events")

	logger.Log.Info().
		Int("filteredEventsCount", len(filteredEvents)).
		Msg("Filtered allowed events")

	// Generate the iCal file
	icsContent := ical.GenerateICS(filteredEvents, calendarName)

	// Set headers and write content
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Write([]byte(icsContent))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
