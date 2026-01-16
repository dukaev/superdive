package domain

import (
	"github.com/wagoodman/dive/dive/image"
)

// ImageStats holds calculated image statistics for display
type ImageStats struct {
	ImageName         string
	TotalSizeBytes    uint64
	WastedBytes       uint64
	EfficiencyScore   float64
	FilesAboveZeroKB  int
	InefficiencyCount int
}

// CalculateImageStats computes statistics from an image analysis.
// This is a pure function that extracts business logic from the UI layer.
func CalculateImageStats(analysis *image.Analysis) ImageStats {
	if analysis == nil {
		return ImageStats{}
	}

	filesAboveZeroKB := countFilesAboveZeroBytes(analysis)
	inefficiencyCount := countInefficiencies(analysis)

	return ImageStats{
		ImageName:         analysis.Image,
		TotalSizeBytes:    analysis.SizeBytes,
		WastedBytes:       analysis.WastedBytes,
		EfficiencyScore:   analysis.Efficiency * 100, // Convert to percentage
		FilesAboveZeroKB:  filesAboveZeroKB,
		InefficiencyCount: inefficiencyCount,
	}
}

// countFilesAboveZeroBytes counts the total number of files with size > 0 bytes across all inefficiencies
func countFilesAboveZeroBytes(analysis *image.Analysis) int {
	if analysis == nil {
		return 0
	}
	count := 0
	for _, ineff := range analysis.Inefficiencies {
		for _, node := range ineff.Nodes {
			if node.Data.FileInfo.Size > 0 {
				count++
			}
		}
	}
	return count
}

// countInefficiencies counts the number of inefficiencies with cumulative size > 0
func countInefficiencies(analysis *image.Analysis) int {
	if analysis == nil {
		return 0
	}
	count := 0
	for _, ineff := range analysis.Inefficiencies {
		if ineff.CumulativeSize > 0 {
			count++
		}
	}
	return count
}
