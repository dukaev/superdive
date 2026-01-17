package styles

import "github.com/charmbracelet/lipgloss"

// --- 1. PALETTE (Adaptive Colors) ---
// CompleteAdaptiveColor automatically switches between Light/Dark based on terminal background

var (
	// Brand Colors
	// Blue: Primary action color
	Blue = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#0060DF", ANSI256: "27", ANSI: "4"}, // Darker blue for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#007AFF", ANSI256: "33", ANSI: "12"},
	}

	// Purple: Secondary/Accent color
	Purple = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#5856D6", ANSI256: "63", ANSI: "5"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#5E5CE6", ANSI256: "63", ANSI: "13"},
	}

	// Status Colors
	// Green: Success / Added
	Green = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#248A3D", ANSI256: "28", ANSI: "2"}, // Darker green for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#34C759", ANSI256: "41", ANSI: "10"},
	}

	// Orange: Warning / Modified
	Orange = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#C45C00", ANSI256: "130", ANSI: "3"}, // Darker orange for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF9500", ANSI256: "214", ANSI: "11"},
	}

	// Red: Error / Removed
	Red = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#D70015", ANSI256: "160", ANSI: "1"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#FF3B30", ANSI256: "196", ANSI: "9"},
	}

	// Yellow/Gold: Highlights
	Yellow = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#856404", ANSI256: "136", ANSI: "3"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFFF00", ANSI256: "226", ANSI: "11"},
	}
	Gold = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#997B00", ANSI256: "136", ANSI: "3"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFD700", ANSI256: "220", ANSI: "11"},
	}

	// Grayscale & Backgrounds
	// White/Black: Main Text
	MainText = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#1C1C1E", ANSI256: "233", ANSI: "0"}, // Almost black for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#FFFFFF", ANSI256: "255", ANSI: "15"},
	}

	// Muted Text
	MutedText = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#6C6C70", ANSI256: "242", ANSI: "8"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#8E8E93", ANSI256: "246", ANSI: "7"},
	}

	// Panel Backgrounds
	PanelBg = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#FFFFFF", ANSI256: "255", ANSI: "15"}, // Pure white for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#1C1C1E", ANSI256: "234", ANSI: "0"},
	}

	// Status Bar / Inputs Backgrounds
	SurfaceBg = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#F2F2F7", ANSI256: "254", ANSI: "7"}, // Light Gray for light bg
		Dark:  lipgloss.CompleteColor{TrueColor: "#3A3A3C", ANSI256: "236", ANSI: "8"},
	}

	// Borders
	Border = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#D1D1D6", ANSI256: "250", ANSI: "7"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#48484A", ANSI256: "240", ANSI: "8"},
	}
)

// --- 2. SEMANTIC ROLES ---

var (
	// Main UI
	PrimaryColor   = Blue
	SecondaryColor = Purple
	BorderColor    = Border

	// Text
	MainTextColor  = MainText
	MutedTextColor = MutedText

	// Backgrounds
	PanelBgColor     = PanelBg
	SelectionBgColor = SurfaceBg // Used for active rows/selections
	InputBgColor     = SurfaceBg
	StatusBarBgColor = SurfaceBg

	// Statuses
	SuccessColor = Green
	WarningColor = Orange
	ErrorColor   = Red

	// Highlights
	HighlightColor          = Yellow
	SelectionHighlightColor = Gold
)

// --- 3. LEGACY SUPPORT (Static Colors) ---
// These are kept for backward compatibility with non-adaptive code

var (
	// Deprecated: Use MainTextColor instead
	Gray           = lipgloss.Color("#8E8E93")
	LightGray      = lipgloss.Color("#C7C7CC")
	MediumGray     = lipgloss.Color("#6e6e73")
	DarkGray       = lipgloss.Color("#48484A")
	DarkerGray     = lipgloss.Color("#3A3A3C")
	BackgroundDark = lipgloss.Color("#1C1C1E")
)

// --- 4. FILE TREE DIFF COLORS ---
// Adaptive colors for file tree diff indicators

var (
	DiffAddedColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#248A3D", ANSI256: "28", ANSI: "2"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#A3BE8C", ANSI256: "108", ANSI: "10"},
	}
	DiffRemovedColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#D70015", ANSI256: "160", ANSI: "1"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#BF616A", ANSI256: "167", ANSI: "9"},
	}
	DiffModifiedColor = lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#C45C00", ANSI256: "130", ANSI: "3"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#EBCB8B", ANSI256: "179", ANSI: "11"},
	}
	DiffNormalColor = MutedText
)
