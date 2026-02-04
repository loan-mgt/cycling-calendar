package types

// Event struct to hold individual event data
type Event struct {
	Date          string        `json:"date"`
	Title         string        `json:"title"`
	Stage         string        `json:"stage"`
	StartTime     string        `json:"start_time"`
	EndTime       string        `json:"end_time"`
	Link          string        `json:"link"`
	// NEW fields from Tiz endpoint
	Country       string        `json:"country"`       // ISO 2-letter: BE, FR, ES
	CountryFlag   string        `json:"country_flag"`  // Flag URL
	StreamType    string        `json:"stream_type"`   // LIVE, POSSIBLE LIVE
	StreamLinks   []string      `json:"stream_links"`  // ALL stream URLs
	StreamLang    string        `json:"stream_lang"`   // Language: English, Spanish, Arabic
	Notes         string        `json:"notes"`         // Additional notes
	Categories    []string      `json:"categories"`    // [WE, ME, track, MTB]
	StartDate     string        `json:"start_date"`    // ISO 8601: 2026-02-04
	EndDate       string        `json:"end_date"`      // ISO 8601: 2026-02-08
	Duration      string        `json:"duration"`      // e.g., "90 mins", "4 hrs"
	AllDay        bool          `json:"all_day"`       // True if time is TBA/missing
	Times         []TizTimeSlot `json:"times"`         // Multiple time slots (WE, ME)
}

