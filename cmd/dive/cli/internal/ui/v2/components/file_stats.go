package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/domain"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
)

// Re-export FileStats from domain package
type FileStats = domain.FileStats

// StatsPartType represents which part of the stats this is
type StatsPartType int

const (
	StatsPartAdded StatsPartType = iota
	StatsPartModified
	StatsPartRemoved
)

// StatsColWidth is the fixed width for each stats column
// 4 chars fits: "999", "1.5k", "100k", "1.5M", "100M"
const StatsColWidth = 4

// StatsPartRenderer handles rendering a single part of file statistics with interactive state
type StatsPartRenderer struct {
	partType StatsPartType
	value    int
	active   bool
	width    int // Dynamic width for this column
}

// NewStatsPartRenderer creates a new stats part renderer
func NewStatsPartRenderer(partType StatsPartType, value int) StatsPartRenderer {
	return StatsPartRenderer{
		partType: partType,
		value:    value,
		active:   false,
		width:    StatsColWidth, // Default width
	}
}

// SetValue updates the value
func (r *StatsPartRenderer) SetValue(value int) {
	r.value = value
}

// GetValue returns the current value
func (r *StatsPartRenderer) GetValue() int {
	return r.value
}

// SetActive sets the active state
func (r *StatsPartRenderer) SetActive(active bool) {
	r.active = active
}

// IsActive returns whether the component is active
func (r *StatsPartRenderer) IsActive() bool {
	return r.active
}

// ToggleActive toggles the active state
func (r *StatsPartRenderer) ToggleActive() {
	r.active = !r.active
}

// SetWidth sets the width for this column
func (r *StatsPartRenderer) SetWidth(width int) {
	r.width = width
}

// GetType returns the type of this stats part
func (r *StatsPartRenderer) GetType() StatsPartType {
	return r.partType
}

// Render renders the stats part as a string with alignment
func (r *StatsPartRenderer) Render() string {
	text := r.formatText()

	// Right-align by padding with spaces on the left
	text = r.alignText(text)

	if r.active {
		return r.activeStyle().Render(text)
	}

	// Use neutral style for zero values
	if r.value == 0 {
		return r.neutralStyle().Render(text)
	}

	return r.defaultStyle().Render(text)
}

// RenderPlain renders the stats part without any colors (for row highlight) with alignment
func (r *StatsPartRenderer) RenderPlain() string {
	text := r.formatText()

	// Right-align by padding with spaces on the left
	text = r.alignText(text)

	return text
}

// formatText formats the base text (value only, no prefix)
func (r *StatsPartRenderer) formatText() string {
	// Format value with k/M suffixes (max 4 chars)
	return utils.FormatCount(r.value)
}

// alignText right-aligns text to r.width by padding with spaces on the left
func (r *StatsPartRenderer) alignText(text string) string {
	if len(text) < r.width {
		return strings.Repeat(" ", r.width-len(text)) + text
	}
	return text
}

// defaultStyle returns the style for inactive state
func (r *StatsPartRenderer) defaultStyle() lipgloss.Style {
	switch r.partType {
	case StatsPartAdded:
		return styles.FileStatsAddedStyle
	case StatsPartModified:
		return styles.FileStatsModifiedStyle
	case StatsPartRemoved:
		return styles.FileStatsRemovedStyle
	default:
		return lipgloss.NewStyle()
	}
}

// neutralStyle returns the style for zero values (grayed out)
func (r *StatsPartRenderer) neutralStyle() lipgloss.Style {
	// Use same muted style as tree metadata
	return styles.MetaDataStyle
}

// activeStyle returns the style for active state
func (r *StatsPartRenderer) activeStyle() lipgloss.Style {
	var bgColor lipgloss.TerminalColor // Use TerminalColor to support both Color and CompleteAdaptiveColor
	switch r.partType {
	case StatsPartAdded:
		bgColor = styles.SuccessColor
	case StatsPartModified:
		bgColor = styles.WarningColor
	case StatsPartRemoved:
		bgColor = styles.ErrorColor
	default:
		bgColor = styles.PrimaryColor
	}

	// FIX: Removed Padding(0, 1) to prevent text shifting
	// The background color is enough indication of state
	return lipgloss.NewStyle().
		Foreground(styles.MainTextColor).
		Background(bgColor).
		Bold(true)
}

// GetVisualWidth returns the visual width of the rendered part
func (r *StatsPartRenderer) GetVisualWidth() int {
	return r.width
}

// FileStatsRow manages a row with three stats parts
type FileStatsRow struct {
	added    StatsPartRenderer
	modified StatsPartRenderer
	removed  StatsPartRenderer
}

// NewFileStatsRow creates a new file stats row
func NewFileStatsRow() FileStatsRow {
	return FileStatsRow{
		added:    NewStatsPartRenderer(StatsPartAdded, 0),
		modified: NewStatsPartRenderer(StatsPartModified, 0),
		removed:  NewStatsPartRenderer(StatsPartRemoved, 0),
	}
}

// SetStats updates all statistics
func (r *FileStatsRow) SetStats(stats FileStats) {
	r.added.SetValue(stats.Added)
	r.modified.SetValue(stats.Modified)
	r.removed.SetValue(stats.Removed)
}

// GetStats returns the current statistics
func (r *FileStatsRow) GetStats() FileStats {
	return FileStats{
		Added:    r.added.GetValue(),
		Modified: r.modified.GetValue(),
		Removed:  r.removed.GetValue(),
	}
}

// GetPart returns the specific part renderer
func (r *FileStatsRow) GetPart(partType StatsPartType) *StatsPartRenderer {
	switch partType {
	case StatsPartAdded:
		return &r.added
	case StatsPartModified:
		return &r.modified
	case StatsPartRemoved:
		return &r.removed
	default:
		return nil
	}
}

// GetAdded returns the added part renderer
func (r *FileStatsRow) GetAdded() *StatsPartRenderer {
	return &r.added
}

// GetModified returns the modified part renderer
func (r *FileStatsRow) GetModified() *StatsPartRenderer {
	return &r.modified
}

// GetRemoved returns the removed part renderer
func (r *FileStatsRow) GetRemoved() *StatsPartRenderer {
	return &r.removed
}

// DeactivateAll deactivates all parts
func (r *FileStatsRow) DeactivateAll() {
	r.added.SetActive(false)
	r.modified.SetActive(false)
	r.removed.SetActive(false)
}

// SetWidths sets the width for all three stats columns
func (r *FileStatsRow) SetWidths(wA, wM, wD int) {
	r.added.SetWidth(wA)
	r.modified.SetWidth(wM)
	r.removed.SetWidth(wD)
}

// Render renders the complete stats row as a string
func (r *FileStatsRow) Render() string {
	// Join with spaces
	addedStr := r.added.Render()
	modifiedStr := r.modified.Render()
	removedStr := r.removed.Render()

	return fmt.Sprintf("%s %s %s", addedStr, modifiedStr, removedStr)
}

// RenderPlain renders the complete stats row without any colors (for row highlight)
func (r *FileStatsRow) RenderPlain() string {
	// Join with spaces, no colors
	addedStr := r.added.RenderPlain()
	modifiedStr := r.modified.RenderPlain()
	removedStr := r.removed.RenderPlain()

	return fmt.Sprintf("%s %s %s", addedStr, modifiedStr, removedStr)
}

// GetPartPositions returns the X positions and widths of each part
// Returns: (addedX, addedWidth, modifiedX, modifiedWidth, removedX, removedWidth)
func (r *FileStatsRow) GetPartPositions(startX int) (int, int, int, int, int, int) {
	addedWidth := r.added.GetVisualWidth()
	modifiedWidth := r.modified.GetVisualWidth()
	removedWidth := r.removed.GetVisualWidth()

	addedX := startX
	modifiedX := addedX + addedWidth + 1 // +1 for space
	removedX := modifiedX + modifiedWidth + 1 // +1 for space

	return addedX, addedWidth, modifiedX, modifiedWidth, removedX, removedWidth
}

// GetPartAtPosition returns which part is at the given X position
// Returns (partType, found)
func (r *FileStatsRow) GetPartAtPosition(x int, startX int) (StatsPartType, bool) {
	addedX, addedWidth, modifiedX, modifiedWidth, removedX, removedWidth := r.GetPartPositions(startX)

	if x >= addedX && x < addedX+addedWidth {
		return StatsPartAdded, true
	}
	if x >= modifiedX && x < modifiedX+modifiedWidth {
		return StatsPartModified, true
	}
	if x >= removedX && x < removedX+removedWidth {
		return StatsPartRemoved, true
	}

	return -1, false
}
