package cronutil

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// IsInEnabledPeriod checks if the current time falls within the enabled window
// defined by the cron schedule and duration.
func IsInEnabledPeriod(schedule string, duration time.Duration) (bool, error) {
	return IsInEnabledPeriodAt(schedule, duration, time.Now().UTC())
}

// IsInEnabledPeriodAt checks if the given time falls within the enabled window
// defined by the cron schedule and duration. This function is time-injectable for testing.
func IsInEnabledPeriodAt(schedule string, duration time.Duration, now time.Time) (bool, error) {
	cronSchedule, err := ParseCronSchedule(schedule)
	if err != nil {
		return false, fmt.Errorf("failed to parse cron schedule: %w", err)
	}

	// Find the most recent time the schedule was triggered
	// We look back up to the duration + 24 hours to handle edge cases
	lookbackWindow := duration + 24*time.Hour
	lastTrigger := findLastTriggerTime(cronSchedule, now, lookbackWindow)

	if lastTrigger.IsZero() {
		// No trigger found in lookback window, not in enabled period
		return false, nil
	}

	// Check if we're within the duration window from the last trigger
	enabledUntil := lastTrigger.Add(duration)
	return now.Before(enabledUntil), nil
}

// findLastTriggerTime finds the most recent time the cron schedule was triggered
// before the given time, within the lookback window.
func findLastTriggerTime(schedule cron.Schedule, now time.Time, lookbackWindow time.Duration) time.Time {
	earliestTime := now.Add(-lookbackWindow)

	// Get the next scheduled time before now
	// We do this by stepping backward in small increments
	var lastTrigger time.Time

	// Use the schedule's Next function to find the next occurrence after earliestTime
	// Then iterate forward to find the last one before now
	triggerTime := schedule.Next(earliestTime)

	for !triggerTime.After(now) && !triggerTime.IsZero() {
		lastTrigger = triggerTime
		// Find next trigger after this one
		triggerTime = schedule.Next(triggerTime)
	}

	return lastTrigger
}

// ParseCronSchedule validates and parses a 5-item cron expression.
// Format: minute hour day month weekday
func ParseCronSchedule(schedule string) (cron.Schedule, error) {
	if schedule == "" {
		return nil, fmt.Errorf("cron schedule cannot be empty")
	}

	cronSchedule, err := parser.Parse(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression '%s': %w", schedule, err)
	}

	return cronSchedule, nil
}

// ValidateDuration parses and validates a duration string.
// Accepts formats like "1h", "30m", "24h", etc.
func ValidateDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return 0, fmt.Errorf("invalid duration '%s': %w", duration, err)
	}

	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive, got: %s", duration)
	}

	return d, nil
}
