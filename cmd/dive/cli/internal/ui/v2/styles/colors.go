package styles

import "github.com/charmbracelet/lipgloss"

// --- UI Colors ---

var (
	// Primary theme colors
	PrimaryColor   = lipgloss.Color("#007AFF")
	SecondaryColor = lipgloss.Color("#5856D6")

	// Status colors
	SuccessColor = lipgloss.Color("#34C759")
	WarningColor = lipgloss.Color("#FF9500")
	ErrorColor   = lipgloss.Color("#FF3B30")

	// Gray scale
	GrayColor      = lipgloss.Color("#8E8E93")
	LightGrayColor = lipgloss.Color("#C7C7CC")
	DarkGrayColor  = lipgloss.Color("#48484A")

	// UI elements
	BorderColor = lipgloss.Color("#3A3A3C")
)

// --- File Tree Diff Colors ---

var (
	// Diff type colors
	DiffAddedColor    = lipgloss.Color("#A3BE8C") // Green for added files
	DiffRemovedColor  = lipgloss.Color("#BF616A") // Red for removed files
	DiffModifiedColor = lipgloss.Color("#EBCB8B") // Yellow for modified files
	DiffNormalColor   = lipgloss.Color("#D8DEE9") // Default color
)
