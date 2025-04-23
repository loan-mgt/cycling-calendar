package handlers

import (
	"cpe/calendar/ical"
	"cpe/calendar/logger"
	"cpe/calendar/request"
	"net/http"
	"os"
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

	// Log environment variables
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

	logger.Log.Info().
		Int("eventsCount", len(events)).
		Msg("Fetched events successfully")

	// Generate the iCal file with the calendar name
	icsContent := ical.GenerateICS(events, calendarName)

	// Set headers for the iCal file response with the provided filename
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Write the iCal content to the response
	w.Write([]byte(icsContent))
}
