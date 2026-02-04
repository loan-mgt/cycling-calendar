package request

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTizRaces(t *testing.T) {
	// Read raw.html from project root
	// Assuming test is running from request/ directory, root is ../
	path := filepath.Join("testdata", "raw.html")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read raw.html: %v", err)
	}

	htmlContent := string(content)
	races, err := parseTizRaces(htmlContent)
	if err != nil {
		t.Fatalf("parseTizRaces returned error: %v", err)
	}

	// Based on races.json, we expect around 25 races (just counting JSON objects)
	// But let's just check we got *something*
	if len(races) == 0 {
		t.Fatal("No races found in raw.html")
	}

	t.Logf("Found %d races", len(races))

	// Verify the first race (Exact Cross Maldegem)
	firstRace := races[0]
	if firstRace.Country != "BE" {
		t.Errorf("Expected first race country BE, got %s", firstRace.Country)
	}
	if firstRace.Name != "Exact Cross Maldegem - Parkcross 2026" {
		t.Errorf("Expected first race name 'Exact Cross Maldegem - Parkcross 2026', got '%s'", firstRace.Name)
	}

	if firstRace.StartDate == "" {
		t.Error("Expected first race StartDate to be populated (from TODAY section), got empty")
	} else {
		t.Logf("First race date: %s", firstRace.StartDate)
	}

	// Check if date category was detected correctly (logic depends on section headers being parsed)
	// The first race is under "TODAY" in raw.html
	// Note: Our parser might not map "TODAY" to a specific date yet vs just using it for section logic
	// But let's check basic fields.
}
