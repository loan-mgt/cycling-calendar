package types

// Event struct to hold individual event data
type Event struct {
	Date      string `json:"date"`
	Title     string `json:"title"`
	Stage     string `json:"stage"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}
