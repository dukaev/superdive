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
	addedHeader := strings.Repeat(" ", imax(0, wA-1)) + "A"
	modifiedHeader := strings.Repeat(" ", imax(0, wM-1)) + "M"
	removedHeader := strings.Repeat(" ", imax(0, wD-1)) + "D"
	statsStr := fmt.Sprintf("%s %s %s", addedHeader, modifiedHeader, removedHeader)

	// Build header using the SAME format as data rows
	// Order: Prefix | ID | Size | Stats | Digest | Command
	var text string
	if digest != "" {
		text = fmt.Sprintf("%-*s%-*s %*s %s %s %s",
			ColWidthPrefix, prefix,   // Now "1/n" without brackets
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			digest,
			cmd,
		)
	} else {
		text = fmt.Sprintf("%-*s%-*s %*s %s %s",
			ColWidthPrefix, prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			cmd,
		)
	}

	// Step 2: Apply color to ENTIRE header at once (not per-cell)
	// This matches how data rows handle colors
	headerStyle := styles.MetaDataStyle // Use same style as digest in data rows
	return headerStyle.Render(text)
}
