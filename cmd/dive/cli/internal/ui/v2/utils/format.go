// Package utils provides utility functions for formatting values used throughout the UI.
//
//revive:disable:var-naming
package utils

import "fmt"

// FormatSize formats bytes into compact human-readable size (max 4 chars)
// Examples: 0, 999b, 1k, 1.5k, 10k, 12M, 100M, 1.5G
func FormatSize(bytes uint64) string {
	const unit = 1024

	// Special case: zero bytes should show as "0" not "0b"
	if bytes == 0 {
		return "0"
	}

	if bytes < 1000 {
		return fmt.Sprintf("%db", bytes)
	}

	val := float64(bytes)
	units := []string{"k", "M", "G", "T", "P"}

	for _, u := range units {
		val /= float64(unit)

		// For values < 10, show 1 decimal place (1.5k = 4 chars)
		if val < 9.95 {
			return fmt.Sprintf("%.1f%s", val, u)
		}

		// For values 10-999, round to integer (10k = 3 chars, 100k = 4 chars)
		if val < 999.5 {
			return fmt.Sprintf("%.0f%s", val, u)
		}
	}

	return fmt.Sprintf("%.0fE", val)
}

// FormatCount formats a number with k/M suffixes (max 4 chars)
// Examples: 999, 1.5k, 10k, 12M, 100M, 1.5G
func FormatCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}

	val := float64(count)
	units := []string{"k", "M", "G", "T"}

	for _, u := range units {
		val /= 1000.0

		// For values < 10, show 1 decimal place (1.5k = 4 chars)
		if val < 9.95 {
			return fmt.Sprintf("%.1f%s", val, u)
		}

		// For values 10-999, round to integer (10k = 3 chars, 100k = 4 chars)
		if val < 999.5 {
			return fmt.Sprintf("%.0f%s", val, u)
		}
	}

	return fmt.Sprintf("%.0fP", val)
}
