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

	// Get allowed events
	allowedEvents, err := request.GetAllowedRace("1")
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to fetch allowed races")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
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
		Int("filteredEventsCount", len(filteredEvents)).
		Msg("Filtered allowed events")

	// Generate the iCal file
	icsContent := ical.GenerateICS(filteredEvents, calendarName)

	// Set headers and write content
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Write([]byte(icsContent))
}
