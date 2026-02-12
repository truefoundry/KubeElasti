package cronutil

import (
	"testing"
	"time"
)

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		name      string
		schedule  string
		wantError bool
	}{
		{
			name:      "valid daily schedule",
			schedule:  "0 0 * * *",
			wantError: false,
		},
		{
			name:      "valid weekday business hours",
			schedule:  "0 9 * * 1-5",
			wantError: false,
		},
		{
			name:      "valid with ranges",
			schedule:  "*/15 8-17 * * 1-5",
			wantError: false,
		},
		{
			name:      "empty schedule",
			schedule:  "",
			wantError: true,
		},
		{
			name:      "invalid format - too few fields",
			schedule:  "0 0 *",
			wantError: true,
		},
		{
			name:      "invalid format - too many fields",
			schedule:  "0 0 * * * *",
			wantError: true,
		},
		{
			name:      "invalid range",
			schedule:  "0 25 * * *",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCronSchedule(tt.schedule)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseCronSchedule() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name      string
		duration  string
		want      time.Duration
		wantError bool
	}{
		{
			name:      "valid hours",
			duration:  "24h",
			want:      24 * time.Hour,
			wantError: false,
		},
		{
			name:      "valid minutes",
			duration:  "30m",
			want:      30 * time.Minute,
			wantError: false,
		},
		{
			name:      "valid mixed",
			duration:  "1h30m",
			want:      90 * time.Minute,
			wantError: false,
		},
		{
			name:      "empty duration",
			duration:  "",
			wantError: true,
		},
		{
			name:      "invalid format",
			duration:  "invalid",
			wantError: true,
		},
		{
			name:      "negative duration",
			duration:  "-1h",
			wantError: true,
		},
		{
			name:      "zero duration",
			duration:  "0h",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateDuration(tt.duration)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDuration() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("ValidateDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInEnabledPeriod(t *testing.T) {
	tests := []struct {
		name      string
		schedule  string
		duration  time.Duration
		mockNow   time.Time
		want      bool
		wantError bool
	}{
		{
			name:      "currently in period - daily at midnight",
			schedule:  "0 0 * * *",
			duration:  24 * time.Hour,
			mockNow:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			want:      true,
			wantError: false,
		},
		{
			name:      "outside period - daily at midnight",
			schedule:  "0 0 * * *",
			duration:  1 * time.Hour,
			mockNow:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			want:      false,
			wantError: false,
		},
		{
			name:      "business hours - during hours",
			schedule:  "0 9 * * 1-5",
			duration:  8 * time.Hour,
			mockNow:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), // Monday 2PM
			want:      true,
			wantError: false,
		},
		{
			name:      "business hours - after hours",
			schedule:  "0 9 * * 1-5",
			duration:  8 * time.Hour,
			mockNow:   time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC), // Monday 6PM
			want:      false,
			wantError: false,
		},
		{
			name:      "business hours - weekend",
			schedule:  "0 9 * * 1-5",
			duration:  8 * time.Hour,
			mockNow:   time.Date(2024, 1, 13, 14, 0, 0, 0, time.UTC), // Saturday 2PM
			want:      false,
			wantError: false,
		},
		{
			name:      "invalid schedule",
			schedule:  "invalid",
			duration:  8 * time.Hour,
			mockNow:   time.Now().UTC(),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsInEnabledPeriodAt(tt.schedule, tt.duration, tt.mockNow)
			if (err != nil) != tt.wantError {
				t.Errorf("IsInEnabledPeriodAt() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError {
				return
			}
			if got != tt.want {
				t.Errorf("IsInEnabledPeriodAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInEnabledPeriodEdgeCases(t *testing.T) {
	// Test that a schedule that just triggered is in the enabled period
	t.Run("just after trigger time", func(t *testing.T) {
		// Fixed time: 2024-01-15 14:05:00 UTC
		mockNow := time.Date(2024, 1, 15, 14, 5, 0, 0, time.UTC)

		// Schedule: Every day at 14:00 (trigger was 5 minutes ago)
		schedule := "0 14 * * *"
		duration := 1 * time.Hour

		got, err := IsInEnabledPeriodAt(schedule, duration, mockNow)
		if err != nil {
			t.Errorf("IsInEnabledPeriodAt() unexpected error: %v", err)
		}
		// 5 minutes after trigger, within 1 hour duration - should be true
		if !got {
			t.Errorf("IsInEnabledPeriodAt() 5 minutes after trigger should be true, got false")
		}
	})

	// Test very long duration
	t.Run("long duration", func(t *testing.T) {
		// Fixed time: 2024-01-15 18:00:00 UTC (Monday, 6 PM)
		mockNow := time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC)

		// Schedule: Daily at midnight
		schedule := "0 0 * * *"
		duration := 168 * time.Hour // 7 days

		got, err := IsInEnabledPeriodAt(schedule, duration, mockNow)
		if err != nil {
			t.Errorf("IsInEnabledPeriodAt() unexpected error: %v", err)
		}
		// Last trigger was at midnight (18 hours ago), within 7-day duration - should be true
		if !got {
			t.Errorf("IsInEnabledPeriodAt() with 7-day duration should be true, got false")
		}
	})
}
