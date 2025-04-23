package ical

import (
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"fmt"
	"time"
)

// GenerateICS generates an ICS string from a list of events
func GenerateICS(events []types.Event, calendarName string) string {
	// Start building the ICS string
	ics := "BEGIN:VCALENDAR\n"
	ics += "VERSION:2.0\n"
	ics += "PRODID:-//github.com/qypol342 //Cycling Calendar//EN\n"
	ics += fmt.Sprintf("NAME:%s\n", calendarName)
	ics += fmt.Sprintf("X-WR-CALNAME:%s\n", calendarName)
	ics += fmt.Sprintf("Description:%s: %s\n", "Cycling Calendar", calendarName)
	ics += fmt.Sprintf("X-WR-CALDESC:%s: %s\n", "Cycling Calendar", calendarName)
	ics += "REFRESH-INTERVAL;VALUE=DURATION:PT1H\n"

	// Define the layout for parsing the datetime with a timezone offset
	const layout = "2006-01-02T15:04:05.000"

	// Loop over each event and generate the calendar content
	for _, event := range events {

		if event.Favori == nil {
			// Log skipped events due to missing Favori field
			logger.Log.Warn().
				Str("eventID", fmt.Sprintf("%v", event.ID)).
				Msg("Skipping event due to missing Favori data")
			continue
		}

		// Extract components from regex matches
		location := event.Favori.F2
		summary := event.Favori.F5 + event.Favori.F3
		description := event.Favori.F4

		// Parse the start and end times in the given time zone
		start, err := time.Parse(layout, event.DateDebut)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int64("eventID", *event.ID).
				Str("startDate", event.DateDebut).
				Msg("Error parsing start time")
			continue
		}

		end, err := time.Parse(layout, event.DateFin)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int64("eventID", *event.ID).
				Str("endDate", event.DateFin).
				Msg("Error parsing end time")
			continue
		}

		// Normalize start and end times to UTC
		loc, err := time.LoadLocation("Europe/Paris")
		if err != nil {
			logger.Log.Error().
				Err(err).
				Msg("Error loading Paris time zone")
			continue
		}

		start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second(), 0, loc).UTC()
		end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), end.Minute(), end.Second(), 0, loc).UTC()

		// Log event details
		logger.Log.Info().
			Int64("eventID", *event.ID).
			Str("summary", summary).
			Str("start", start.String()).
			Str("startUTC", start.UTC().String()).
			Str("end", end.String()).
			Msg("Event processed for ICS generation")

		// Add event details to ICS string
		ics += "BEGIN:VEVENT\n"
		ics += fmt.Sprintf("UID:%d\n", event.ID)
		ics += fmt.Sprintf("DTSTART:%s\n", start.Format("20060102T150405Z"))
		ics += fmt.Sprintf("DTEND:%s\n", end.Format("20060102T150405Z"))
		ics += fmt.Sprintf("LOCATION:%s\n", location)
		ics += fmt.Sprintf("SUMMARY:%s\n", summary)
		ics += fmt.Sprintf("DESCRIPTION:%s\n", description)
		ics += "END:VEVENT\n"
	}

	// Close the VCALENDAR block
	ics += "END:VCALENDAR\n"

	// Log the successful generation of the ICS content
	logger.Log.Info().
		Int("eventCount", len(events)).
		Msg("Generated ICS content successfully")

	return ics
}
