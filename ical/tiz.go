package ical

import (
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var monthMap = map[string]int{
	"January": 1, "February": 2, "March": 3, "April": 4,
	"May": 5, "June": 6, "July": 7,
	"August": 8, "September": 9, "October": 10,
	"November": 11, "December": 12,
}

// GenerateTizICS generates an ICS string from a list of Tiz events
func GenerateTizICS(events []types.Event, calendarName string) string {
	// Start building ICS string with proper CRLF line endings
	ics := "BEGIN:VCALENDAR\r\n"
	ics += "VERSION:2.0\r\n"
	ics += "PRODID:-//github.com/qypol342 //Cycling Calendar//EN\r\n"
	ics += fmt.Sprintf("NAME:%s\r\n", calendarName)
	ics += fmt.Sprintf("X-WR-CALNAME:%s\r\n", calendarName)
	ics += fmt.Sprintf("Description:%s: %s\r\n", "Cycling Calendar", calendarName)
	ics += fmt.Sprintf("X-WR-CALDESC:%s: %s\r\n", "Cycling Calendar", calendarName)
	ics += "REFRESH-INTERVAL;VALUE=DURATION:PT1H\r\n"

	// Get current year for parsing
	currentYear := time.Now().Year()

	// Loop over each event and generate calendar content
	// Loop over each event and generate calendar content
	for _, event := range events {
		summary := buildTizSummary(event)

		// Build description with race info
		descriptionLines := buildTizEventDescription(event)

		// Convert description lines to ICS format with literal \n
		var descriptionBuilder strings.Builder
		for _, line := range descriptionLines {
			if line == "" {
				// Empty line becomes literal \n
				descriptionBuilder.WriteString("\\n")
			} else {
				// Escape line content and add literal \n
				descriptionBuilder.WriteString(escapeICSText(line))
				descriptionBuilder.WriteString("\\n")
			}
		}
		description := descriptionBuilder.String()

		var start, end time.Time
		var err error

		if event.AllDay {
			// All-day event - date only
			start, err = parseTizDateOnly(event.StartDate, currentYear)
			if err != nil {
				logger.Log.Error().
					Err(err).
					Str("startDate", event.StartDate).
					Msg("Error parsing start date")
				continue
			}
			end, err = parseTizDateOnly(event.EndDate, currentYear)
			if err != nil {
				logger.Log.Warn().
					Err(err).
					Str("endDate", event.EndDate).
					Msg("Error parsing end date, defaulting to start date")
				end = start
			}

			// Use date-only format
			ics += "BEGIN:VEVENT\r\n"
			ics += fmt.Sprintf("UID:%s%s\r\n", event.Title, event.Stage)
			ics += fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", start.Format("20060102"))
			ics += fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", end.Format("20060102"))
			ics += fmt.Sprintf("SUMMARY:%s\r\n", summary)
			ics += foldICSLine(fmt.Sprintf("DESCRIPTION:%s", description))
			if len(event.StreamLinks) > 0 {
				ics += fmt.Sprintf("URL:%s\r\n", event.StreamLinks[0])
			}
			ics += "END:VEVENT\r\n"

		} else {
			// Normal datetime event
			if len(event.Times) > 0 {
				// Parse start from first time slot
				start, err = parseTizTime(event.Times[0].Time, event.StartDate)
				if err != nil {
					logger.Log.Error().
						Err(err).
						Str("startTime", event.Times[0].Time).
						Msg("Error parsing start time")
					continue
				}

				// Calculate end time using duration
				durationMins := parseDurationMinutes(event.Duration)
				end = start.Add(time.Duration(durationMins) * time.Minute)
			} else if event.StartTime != "" {
				start, err = parseTizTime(event.StartTime, event.StartDate)
				if err != nil {
					logger.Log.Error().
						Err(err).
						Str("startTime", event.StartTime).
						Msg("Error parsing start time")
					continue
				}
				// Default to +3 hours if no duration
				if event.Duration == "" {
					end = start.Add(3 * time.Hour)
				} else {
					durationMins := parseDurationMinutes(event.Duration)
					end = start.Add(time.Duration(durationMins) * time.Minute)
				}
			} else {
				// Skip event if no time info
				continue
			}

			// Log event details
			logger.Log.Info().
				Str("summary", summary).
				Str("start", start.String()).
				Str("startUTC", start.UTC().String()).
				Str("end", end.String()).
				Str("allDay", fmt.Sprintf("%v", event.AllDay)).
				Msg("Event processed for Tiz ICS generation")

			// Add event details to ICS string with proper CRLF line endings
			ics += "BEGIN:VEVENT\r\n"
			ics += fmt.Sprintf("UID:%s%s\r\n", event.Title, event.Stage)
			ics += fmt.Sprintf("DTSTART:%s\r\n", start.Format("20060102T150405Z"))
			ics += fmt.Sprintf("DTEND:%s\r\n", end.Format("20060102T150405Z"))
			ics += fmt.Sprintf("SUMMARY:%s\r\n", summary)
			ics += foldICSLine(fmt.Sprintf("DESCRIPTION:%s", description))
			if len(event.StreamLinks) > 0 {
				ics += fmt.Sprintf("URL:%s\r\n", event.StreamLinks[0])
			}
			ics += "END:VEVENT\r\n"
		}
	}

	// Close VCALENDAR block
	ics += "END:VCALENDAR\r\n"

	// Log successful generation of ICS content
	logger.Log.Info().
		Int("eventCount", len(events)).
		Msg("Generated Tiz ICS content successfully")

	return ics
}

// buildTizSummary builds the event summary from race data
func buildTizSummary(event types.Event) string {
	var summary strings.Builder

	// Title
	summary.WriteString(event.Title)

	// Stage
	if event.Stage != "" {
		summary.WriteString(" | ")
		summary.WriteString(event.Stage)
	}

	// Categories
	if len(event.Categories) > 0 {
		catNames := make([]string, len(event.Categories))
		for i, cat := range event.Categories {
			if displayName, ok := types.TizCategoryMap[cat]; ok {
				catNames[i] = displayName
			} else {
				catNames[i] = cat
			}
		}
		summary.WriteString(" (")
		summary.WriteString(strings.Join(catNames, ", "))
		summary.WriteString(")")
	}

	return summary.String()
}

// buildTizEventDescription creates a formatted description for an event
func buildTizEventDescription(event types.Event) []string {
	var lines []string

	// Add country
	if event.Country != "" {
		lines = append(lines, fmt.Sprintf(" Country: %s", event.Country))
	}

	// Add categories
	if len(event.Categories) > 0 {
		catNames := make([]string, len(event.Categories))
		for i, cat := range event.Categories {
			if displayName, ok := types.TizCategoryMap[cat]; ok {
				catNames[i] = displayName
			} else {
				catNames[i] = cat
			}
		}
		lines = append(lines, fmt.Sprintf(" Categories: %s", strings.Join(catNames, ", ")))
	}

	// Add stream type and language
	if event.StreamType != "" {
		lines = append(lines, fmt.Sprintf(" Stream: %s", event.StreamType))
	}
	if event.StreamLang != "" {
		lines = append(lines, fmt.Sprintf(" Commentary: %s", event.StreamLang))
	}

	// Add duration
	if event.Duration != "" {
		lines = append(lines, fmt.Sprintf(" Duration: %s", event.Duration))
	}

	// Add time slots if available
	if len(event.Times) > 0 {
		lines = append(lines, " Time slots:")
		for _, timeSlot := range event.Times {
			lines = append(lines, fmt.Sprintf("  %s: %s (%s)", timeSlot.Category, timeSlot.Time, timeSlot.Duration))
		}
	}

	// Add notes
	if event.Notes != "" {
		lines = append(lines, fmt.Sprintf(" Note: %s", event.Notes))
	}

	// Add all stream links
	if len(event.StreamLinks) > 0 {
		lines = append(lines, " Stream links:")
		for _, link := range event.StreamLinks {
			lines = append(lines, fmt.Sprintf("  - %s", link))
		}
	}

	// Add info link
	if event.Link != "" {
		lines = append(lines, fmt.Sprintf(" More info: %s", event.Link))
	}

	return lines
}

// parseTizDateOnly parses a date-only string (e.g., "2026-02-04")
func parseTizDateOnly(dateStr string, year int) (time.Time, error) {
	dateParts := strings.Split(dateStr, "-")
	if len(dateParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// parseTizTime parses a time string (e.g., "14:00 UTC") combined with a date string
func parseTizTime(timeStr, dateStr string) (time.Time, error) {
	// Parse date part
	dateParts := strings.Split(dateStr, "-")
	if len(dateParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}
	year, _ := strconv.Atoi(dateParts[0])
	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])

	// Parse time part (HH:MM UTC or HH:MM:SS UTC)
	// Remove " UTC" suffix and trim
	cleanTime := strings.TrimSuffix(strings.TrimSpace(timeStr), " UTC")
	cleanTime = strings.TrimSpace(cleanTime)

	timeParts := strings.Split(cleanTime, ":")
	if len(timeParts) < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour, _ := strconv.Atoi(strings.TrimSpace(timeParts[0]))
	minute, _ := strconv.Atoi(strings.TrimSpace(timeParts[1]))
	second := 0
	if len(timeParts) >= 3 {
		second, _ = strconv.Atoi(strings.TrimSpace(timeParts[2]))
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), nil
}

// parseDurationMinutes converts duration string to minutes
func parseDurationMinutes(duration string) int {
	// Pattern: "90 mins", "2 hrs", "3.25 hrs"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mins?|hrs?|hours?)`)
	matches := re.FindStringSubmatch(duration)
	if len(matches) < 2 {
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
