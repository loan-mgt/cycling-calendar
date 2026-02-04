package types

// TizRace represents raw race data from Tiz endpoint
type TizRace struct {
	RawHTML       string
	Country       string
	CountryFlag   string
	Name          string
	Stage         string
	Categories    []string
	StreamType    string
	StreamLinks   []string
	StreamLang    string
	Notes         string
	StartDate     string
	EndDate       string
	Duration      string
	Times         []TizTimeSlot
	AllDay       bool
}

type TizTimeSlot struct {
	Category string  // WE, ME
	Time     string  // 14:00 UTC
	Duration string  // 60 mins
}

// Map Tiz categories to display names
var TizCategoryMap = map[string]string{
	"WE":   "Women Elite",
	"ME":   "Men Elite",
	"track": "Track",
	"MTB":   "Mountain Bike",
	"NC":   "National Championships",
	"JR":   "Junior",
	"WC":   "World Championships",
}
