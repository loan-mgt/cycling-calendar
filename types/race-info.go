package types

type RaceInfo struct {
	Datename              string // e.g. "05 July 2025"
	StartTime             string // e.g. "13:40"
	AvgSpeedWinner        string // e.g. "-"
	Classification        string // e.g. "2.UWT"
	RaceCategory          string // e.g. "ME - Men Elite"
	DistanceKm            string // e.g. "184.9 km"
	PointsScale           string // e.g. "GT.A.Stage"
	UCIScale              string // e.g. "UCI.WR.GT.A.Stage"
	ParcoursType          string // e.g. ""
	ProfileScore          string // e.g. "17"
	VerticalMeters        string // e.g. "1065"
	Departure             string // e.g. "Lille Métropole"
	Arrival               string // e.g. "Lille Métropole"
	RaceRanking           string // e.g. ""
	StartlistQualityScore string // e.g. "1711"
	WonHow                string // e.g. "-"
	AvgTemperature        string // e.g. ""
}
