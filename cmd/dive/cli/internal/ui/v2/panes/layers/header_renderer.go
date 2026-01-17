package layers

import (
	"fmt"
	"strings"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/components"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// RenderHeader creates a column header row for the layers panel
// The header shows: [#] ID Size A M D Command Digest
func RenderHeader(width int) string {
	// Step 1: Build header text WITHOUT colors (same as data rows)
	prefix := "#"
	id := "ID"
	size := "Size"
	digest := "Digest"
	cmd := "Command"

	// Build stats header string with proper alignment (A M D)
	// Each stat column is StatsColWidth (4) with space between
	// Format: "   A    M    D" (right-aligned in 4-char columns)
	addedHeader := strings.Repeat(" ", components.StatsColWidth-1) + "A"
	modifiedHeader := strings.Repeat(" ", components.StatsColWidth-1) + "M"
	removedHeader := strings.Repeat(" ", components.StatsColWidth-1) + "D"
	statsStr := fmt.Sprintf("%s %s %s", addedHeader, modifiedHeader, removedHeader)

	// Build header using the SAME format as data rows
	// Order: Prefix | ID | Size | Stats | Command | Digest
	var text string
	if digest != "" {
		text = fmt.Sprintf("%-*s%-*s %*s %s %s %s",
			ColWidthPrefix-1, prefix,     // -1 because data has "[n/n] " but header just "#"
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			cmd,
			digest,
		)
	} else {
		text = fmt.Sprintf("%-*s%-*s %*s %s %s",
			ColWidthPrefix-1, prefix,
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
