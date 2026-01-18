// Package layers provides the layers pane
package layers

import (
	"fmt"
	"strings"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// imax returns the maximum of two integers
func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RenderHeader creates a column header row for the layers panel
// The header shows: [#] ID Size A M D Digest Command
func RenderHeader(width int, wA, wM, wD int) string {
	// FIX: Calculate inner width first (viewport width without panel borders)
	viewportWidth := width - 2

	// FIX: Use viewportWidth for visibility check to match generateContent logic
	showCommand, showDigest, showStats := GetColumnVisibility(viewportWidth)

	// Step 1: Build header text WITHOUT colors (same as data rows)
	prefix := "#"
	id := "ID"
	size := "Size"
	const digestWidth = 6
	digest := "Digest"
	// Pad digest to fixed width for alignment
	for len(digest) < digestWidth {
		digest += " "
	}
	cmd := "Command"

	// Build stats header string with dynamic alignment (A M D)
	// wA-1 because "A" occupies 1 character, rest is spaces
	// Use imax(0, w-1) to avoid negative repeat count
	var statsStr string
	if showStats {
		addedHeader := strings.Repeat(" ", imax(0, wA-1)) + "A"
		modifiedHeader := strings.Repeat(" ", imax(0, wM-1)) + "M"
		removedHeader := strings.Repeat(" ", imax(0, wD-1)) + "D"
		statsStr = fmt.Sprintf("%s %s %s", addedHeader, modifiedHeader, removedHeader)
	}

	// Build header using the SAME format as data rows
	// Order: Prefix | ID | Size | Stats | Digest | Command
	var text string
	switch {
	case showDigest && showCommand:
		// All columns
		text = fmt.Sprintf("%-*s%-*s %*s %s %s %s",
			ColWidthPrefix, prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			digest,
			cmd,
		)
	case showDigest:
		// Without Command
		text = fmt.Sprintf("%-*s%-*s %*s %s %s",
			ColWidthPrefix, prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			digest,
		)
	case showCommand:
		// Without Digest
		text = fmt.Sprintf("%-*s%-*s %*s %s %s",
			ColWidthPrefix, prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			cmd,
		)
	default:
		// Only Stats
		text = fmt.Sprintf("%-*s%-*s %*s %s",
			ColWidthPrefix, prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
		)
	}

	// Step 2: Pad header to full width to match data rows
	// This ensures header fills the entire available width
	textWidth := len(text) // Simple len since text is plain ASCII
	padding := viewportWidth - textWidth
	if padding > 0 {
		text += strings.Repeat(" ", padding)
	}

	// Step 3: Apply color to ENTIRE header at once (not per-cell)
	// This matches how data rows handle colors
	headerStyle := styles.MetaDataStyle // Use same style as digest in data rows
	return headerStyle.Render(text)
}
