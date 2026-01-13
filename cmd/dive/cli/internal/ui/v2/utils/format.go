package utils

import "fmt"

// FormatSize formats bytes into human-readable size
// This is a shared utility used by all panes
func FormatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatCount formats a number with k/M suffixes to fit in 4 characters max
// Examples: 999, 1.2k, 100k, 1.5M, 100M, 1.2G
func FormatCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 10000 {
		// Show as "1.2k" format for 1000-9999
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	}
	if count < 1000000 {
		// Show as "100k" format for 10000-999999
		return fmt.Sprintf("%dk", count/1000)
	}
	if count < 10000000 {
		// Show as "1.5M" format for 1M-9.9M
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count < 1000000000 {
		// Show as "100M" format for 10M-999M
		return fmt.Sprintf("%dM", count/1000000)
	}
	// Show as "1.5G" for 1B+
	return fmt.Sprintf("%.1fG", float64(count)/1000000000)
}
