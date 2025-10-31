package rag

import (
	"testing"
	"time"
)

// TestExpandDateRange tests the date range expansion function
func TestExpandDateRange(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		days     int
		wantLen  int
		wantErr  bool
		wantLast string // Expected last date in range
	}{
		{
			name:     "7 days from 2025-10-31",
			date:     "2025-10-31",
			days:     7,
			wantLen:  7,
			wantErr:  false,
			wantLast: "2025-10-25",
		},
		{
			name:     "3 days from 2025-10-15",
			date:     "2025-10-15",
			days:     3,
			wantLen:  3,
			wantErr:  false,
			wantLast: "2025-10-13",
		},
		{
			name:     "1 day (same date)",
			date:     "2025-10-31",
			days:     1,
			wantLen:  1,
			wantErr:  false,
			wantLast: "2025-10-31",
		},
		{
			name:    "invalid date format",
			date:    "2025/10/31",
			days:    7,
			wantErr: true,
		},
		{
			name:    "empty date",
			date:    "",
			days:    7,
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandDateRange(tt.date, tt.days)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandDateRange() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ExpandDateRange() unexpected error = %v", err)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("ExpandDateRange() length = %d, want %d", len(got), tt.wantLen)
			}

			if tt.wantLen > 0 {
				// First date should be the input date
				if got[0] != tt.date {
					t.Errorf("ExpandDateRange() first date = %s, want %s", got[0], tt.date)
				}

				// Last date should match expected
				if tt.wantLast != "" && got[len(got)-1] != tt.wantLast {
					t.Errorf("ExpandDateRange() last date = %s, want %s", got[len(got)-1], tt.wantLast)
				}

				// Dates should be in descending order (newest first)
				for i := 1; i < len(got); i++ {
					if got[i] >= got[i-1] {
						t.Errorf("ExpandDateRange() dates not in descending order: %v", got)
						break
					}
				}

				t.Logf("Date range: %v", got)
			}
		})
	}
}

// TestGetTodayDate tests the today's date function
func TestGetTodayDate(t *testing.T) {
	got := GetTodayDate()

	// Verify format YYYY-MM-DD
	_, err := time.Parse("2006-01-02", got)
	if err != nil {
		t.Errorf("GetTodayDate() returned invalid format: %s, error: %v", got, err)
	}

	// Verify it's today's date
	expected := time.Now().Format("2006-01-02")
	if got != expected {
		t.Errorf("GetTodayDate() = %s, want %s", got, expected)
	}

	t.Logf("Today's date: %s", got)
}

// TestExpandDateRange_EdgeCases tests edge cases
func TestExpandDateRange_EdgeCases(t *testing.T) {
	t.Run("month boundary", func(t *testing.T) {
		// Starting from Oct 3, going back 7 days should cross into September
		got, err := ExpandDateRange("2025-10-03", 7)
		if err != nil {
			t.Fatalf("ExpandDateRange() error = %v", err)
		}

		if len(got) != 7 {
			t.Errorf("Expected 7 dates, got %d", len(got))
		}

		// Should include dates from September
		lastDate := got[len(got)-1]
		if lastDate != "2025-09-27" {
			t.Errorf("Expected last date to be 2025-09-27, got %s", lastDate)
		}

		t.Logf("Month boundary range: %v", got)
	})

	t.Run("year boundary", func(t *testing.T) {
		// Starting from Jan 3, going back 7 days should cross into previous year
		got, err := ExpandDateRange("2025-01-03", 7)
		if err != nil {
			t.Fatalf("ExpandDateRange() error = %v", err)
		}

		// Should include dates from December 2024
		lastDate := got[len(got)-1]
		if lastDate != "2024-12-28" {
			t.Errorf("Expected last date to be 2024-12-28, got %s", lastDate)
		}

		t.Logf("Year boundary range: %v", got)
	})

	t.Run("leap year", func(t *testing.T) {
		// Testing around Feb 29 in a leap year
		got, err := ExpandDateRange("2024-03-01", 5)
		if err != nil {
			t.Fatalf("ExpandDateRange() error = %v", err)
		}

		// Should include Feb 29, 2024 (leap day)
		found := false
		for _, date := range got {
			if date == "2024-02-29" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected to find 2024-02-29 (leap day) in range: %v", got)
		}

		t.Logf("Leap year range: %v", got)
	})
}
