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

// buildEventDescription creates a formatted description for an event
func buildEventDescription(event types.Event, hasRaceInfo bool, raceInfoList []*types.RaceInfo, index int) []string {
	var lines []string

	// Add race info if available
	if hasRaceInfo && index < len(raceInfoList) && raceInfoList[index] != nil {
		raceInfo := raceInfoList[index]

		// Add route info (Departure > Arrival) with proper indentation
		if raceInfo.Departure != "" && raceInfo.Arrival != "" {
			var locationLine string
			if raceInfo.Departure == raceInfo.Arrival {
				locationLine = fmt.Sprintf(" Location: %s", raceInfo.Departure)
			} else {
				locationLine = fmt.Sprintf(" Route: %s > %s", raceInfo.Departure, raceInfo.Arrival)
			}

			// Add distance
			if raceInfo.DistanceKm != "" {
				locationLine += fmt.Sprintf(" (%s)", raceInfo.DistanceKm)
			}
			lines = append(lines, locationLine)
			lines = append(lines, "") // Empty line
		}

		// Add race details with ASCII symbols and proper indentation
		if raceInfo.VerticalMeters != "" {
			lines = append(lines, fmt.Sprintf(" * Vertical Meters: %s", raceInfo.VerticalMeters))
		}
		if raceInfo.Classification != "" {
			lines = append(lines, fmt.Sprintf(" * Classification: %s", raceInfo.Classification))
		}
		if raceInfo.RaceCategory != "" {
			lines = append(lines, fmt.Sprintf(" * Category: %s", raceInfo.RaceCategory))
		}
		if raceInfo.ProfileScore != "" {
			lines = append(lines, fmt.Sprintf(" * Profile Score: %s", raceInfo.ProfileScore))
		}

		// Add a separator if we have race info
		if raceInfo.Departure != "" || raceInfo.Classification != "" {
			lines = append(lines, "") // Empty line
		}
	}

	// Always add the link at the end with proper indentation
	if event.Link != "" {
		lines = append(lines, fmt.Sprintf(" More info: %s", event.Link))
	}

	return lines
}

// escapeICSText properly escapes text for ICS format
func escapeICSText(text string) string {
	// Escape backslashes, commas, and semicolons
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, ";", "\\;")

	// Remove or replace problematic characters
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\t", " ")

	return text
}

// foldICSLine properly folds long ICS lines according to RFC 5545
func foldICSLine(line string) string {
	if len(line) <= 75 {
		return line + "\r\n"
	}

	var result strings.Builder
	result.WriteString(line[:75] + "\r\n")

	for i := 75; i < len(line); i += 74 {
		end := i + 74
		if end > len(line) {
			end = len(line)
		}
		result.WriteString(" " + line[i:end] + "\r\n")
	}

	return result.String()
}

// GenerateICS generates an ICS string from a list of events
func GenerateICS(events []types.Event, calendarName string, raceInfoList []*types.RaceInfo) string {
	// Start building the ICS string with proper CRLF line endings
	ics := "BEGIN:VCALENDAR\r\n"
	ics += "VERSION:2.0\r\n"
	ics += "PRODID:-//github.com/qypol342 //Cycling Calendar//EN\r\n"
	ics += fmt.Sprintf("NAME:%s\r\n", calendarName)
	ics += fmt.Sprintf("X-WR-CALNAME:%s\r\n", calendarName)
	ics += fmt.Sprintf("Description:%s: %s\r\n", "Cycling Calendar", calendarName)
	ics += fmt.Sprintf("X-WR-CALDESC:%s: %s\r\n", "Cycling Calendar", calendarName)
	ics += "REFRESH-INTERVAL;VALUE=DURATION:PT1H\r\n"

	// Loop over each event and generate the calendar content
	for i, event := range events {
		summary := fmt.Sprintf("%s | %s", event.Title, event.Stage)

		// Build description with race info if available
		descriptionLines := buildEventDescription(event, i < len(raceInfoList) && raceInfoList[i] != nil, raceInfoList, i)

		// Convert description lines to ICS format with literal \n
		var descriptionBuilder strings.Builder
		for _, line := range descriptionLines {
			if line == "" {
				// Empty line becomes literal \n
				descriptionBuilder.WriteString("\\n")
			} else {
				// Escape the line content and add literal \n
				descriptionBuilder.WriteString(escapeICSText(line))
				descriptionBuilder.WriteString("\\n")
			}
		}
		description := descriptionBuilder.String()

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

		// Add event details to ICS string with proper CRLF line endings
		ics += "BEGIN:VEVENT\r\n"
		ics += fmt.Sprintf("UID:%s%s\r\n", event.Title, event.Stage)
		ics += fmt.Sprintf("DTSTART:%s\r\n", start.Format("20060102T150405Z"))
		ics += fmt.Sprintf("DTEND:%s\r\n", end.Format("20060102T150405Z"))
		ics += fmt.Sprintf("SUMMARY:%s\r\n", summary)
		ics += foldICSLine(fmt.Sprintf("DESCRIPTION:%s", description))
		ics += fmt.Sprintf("URL:%s\r\n", event.Link)
		ics += "END:VEVENT\r\n"
	}

	// Close the VCALENDAR block
	ics += "END:VCALENDAR\r\n"

	// Log the successful generation of the ICS content
	logger.Log.Info().
		Int("eventCount", len(events)).
		Msg("Generated ICS content successfully")

	return ics
}
