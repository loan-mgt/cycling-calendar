package ical

import (
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseDateTime combines date (DD/MM) and time (HH:MM) into a time.Time
func parseDateTime(timeStr, dateStr string) (time.Time, error) {
	// Parse the date part (DD/MM)
	dateParts := strings.Split(dateStr, "/")
	if len(dateParts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	day, err := strconv.Atoi(dateParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", dateParts[0])
	}

	month, err := strconv.Atoi(dateParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", dateParts[1])
	}

	// Parse the time part (HH:MM)
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %s", timeParts[0])
	}

	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %s", timeParts[1])
	}

	// Use current year
	year := time.Now().Year()

	// Create time.Time object
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC), nil
}

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

	// Loop over each event and generate the calendar content
	for _, event := range events {
		summary := event.Title
		description := event.Stage

		// Parse the start and end times in the given time zone
		startDateTime, err := parseDateTime(event.StartTime, event.Date)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("StartTime", event.StartTime).
				Str("Date", event.Date).
				Msg("Error parsing start time")
			continue
		}

		// For end time, use start time + 3h if not provided
		var endDateTime time.Time
		if event.EndTime != "" {
			endDateTime, err = parseDateTime(event.EndTime, event.Date)
			if err != nil {
				logger.Log.Error().
					Err(err).
					Str("EndTime", event.EndTime).
					Str("Date", event.Date).
					Msg("Error parsing end time")
				continue
			}
		} else {
			endDateTime = startDateTime.Add(3 * time.Hour)
		}

		// Normalize to specified timezone (Europe/Paris)
		loc, err := time.LoadLocation("Europe/Paris")
		if err != nil {
			logger.Log.Error().
				Err(err).
				Msg("Error loading Paris time zone")
			continue
		}

		start := time.Date(startDateTime.Year(), startDateTime.Month(), startDateTime.Day(),
			startDateTime.Hour(), startDateTime.Minute(), 0, 0, loc).UTC()
		end := time.Date(endDateTime.Year(), endDateTime.Month(), endDateTime.Day(),
			endDateTime.Hour(), endDateTime.Minute(), 0, 0, loc).UTC()

		// Log event details
		logger.Log.Info().
			Str("summary", summary).
			Str("start", start.String()).
			Str("startUTC", start.UTC().String()).
			Str("end", end.String()).
			Msg("Event processed for ICS generation")

		// Add event details to ICS string
		ics += "BEGIN:VEVENT\n"
		ics += fmt.Sprintf("UID:%s%s\n", event.Title, event.Stage)
		ics += fmt.Sprintf("DTSTART:%s\n", start.Format("20060102T150405Z"))
		ics += fmt.Sprintf("DTEND:%s\n", end.Format("20060102T150405Z"))
		ics += fmt.Sprintf("SUMMARY:%s\n", summary)
		ics += fmt.Sprintf("DESCRIPTION:%s\n", description)
		ics += fmt.Sprintf("URL:%s\n", event.Link)
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
