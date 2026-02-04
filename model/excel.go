package model

type YearStatement struct {
	Year   int     `json:"year"`
	Assets float64 `json:"assets"`
}

type ExcelSheetRequest struct {
	PreviousYears []YearStatement `json:"previous_years"`
}
