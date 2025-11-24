package rag

import (
	"time"
)

// ExpandDateRange takes a date string in YYYY-MM-DD format and returns an array
// of date strings spanning backwards from the given date for the specified number of days.
// Example: ExpandDateRange("2025-10-14", 7) returns ["2025-10-14", "2025-10-13", ..., "2025-10-08"]
func ExpandDateRange(dateStr string, days int) ([]string, error) {
	if dateStr == "" {
		return nil, nil
	}

	// Parse the date string
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}

	// Generate array from target date backwards
	dateArray := make([]string, 0, days)
	for i := 0; i < days; i++ {
		date := targetDate.AddDate(0, 0, -i)
		dateArray = append(dateArray, date.Format("2006-01-02"))
	}

	return dateArray, nil
}

// GetTodayDate returns today's date in YYYY-MM-DD format
func GetTodayDate() string {
	return time.Now().Format("2006-01-02")
}
