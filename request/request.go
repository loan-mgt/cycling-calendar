package request

import (
	"compress/gzip"
	"cpe/calendar/logger"
	"cpe/calendar/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func FetchData(timezone string) ([]types.Event, error) {
	// Log the operation with context about the start and end times
	logger.Log.Info().
		Str("timezone", timezone).
		Msg("Fetching data from PCS")

	body, err := getInfo(timezone)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to fetch calendar data")
		return nil, err
	}

	logger.Log.Info().
		Str("timezone", timezone).
		Msg("Data fetched successfully")
	return body, nil
}

func getInfo(timezone string) ([]types.Event, error) {
	// Log the request to fetch calendar data with the token and time context
	logger.Log.Info().
		Str("timezone", timezone).
		Msg("Fetching calendar data")

	// Define the base URL and query parameters
	baseURL := "https://www.procyclingstats.com/races.php?filter=Filter&p=uci&s=start-finish-schedule"

	query := fmt.Sprintf("&timezone=%s", timezone)
	logger.Log.Debug().
		Str("finalURL", baseURL+query).
		Msg("Generated final URL")

	// Create the GET request
	req, err := http.NewRequest("GET", baseURL+query, nil)
	if err != nil {
		logger.Log.Error().
			Str("finalURL", baseURL+query).
			Err(err).
			Msg("Failed to create calendar request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to match the curl request
	req.Header.Add("User-Agent", "Dalvik/2.1.0 (Linux; U; Android 15; sdk_gphone64_x86_64 Build/AE3A.240806.005)")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Connection", "Keep-Alive")

	// Send the GET request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error().
			Str("finalURL", baseURL+query).
			Err(err).
			Msg("Request failed to get calendar data")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle gzip encoding if necessary
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			logger.Log.Error().
				Str("finalURL", baseURL+query).
				Err(err).
				Msg("Failed to create gzip reader")
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.(*gzip.Reader).Close()
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error().
			Str("finalURL", baseURL+query).
			Int("statusCode", resp.StatusCode).
			Msg("Received non-200 response while fetching calendar")
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(reader)
	if err != nil {
		logger.Log.Error().
			Str("finalURL", baseURL+query).
			Err(err).
			Msg("Failed to read calendar response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response into the events slice
	var events []types.Event
	err = json.Unmarshal(body, &events)
	if err != nil {
		logger.Log.Error().
			Str("finalURL", baseURL+query).
			Err(err).
			Msg("Failed to parse calendar JSON response")
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	logger.Log.Info().
		Str("timezone", timezone).
		Msg("Calendar data fetched successfully")
	return events, nil
}
