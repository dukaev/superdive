package styles

import "github.com/charmbracelet/lipgloss"

// --- 1. PALETTE (Базовая палитра) ---
// Определяем цвета здесь, чтобы менять их в одном месте

var (
	// Brand Colors
	Blue   = lipgloss.Color("#007AFF")
	Purple = lipgloss.Color("#5856D6")

	// Status Colors
	Green  = lipgloss.Color("#34C759")
	Orange = lipgloss.Color("#FF9500")
	Red    = lipgloss.Color("#FF3B30")
	Yellow = lipgloss.Color("#FFFF00")
	Gold   = lipgloss.Color("#FFD700")

	// Grayscale & Backgrounds
	White          = lipgloss.Color("#FFFFFF")
	Gray           = lipgloss.Color("#8E8E93")
	LightGray      = lipgloss.Color("#C7C7CC")
	MediumGray     = lipgloss.Color("#6e6e73") // Был захардкожен как #6e6e73
	DarkGray       = lipgloss.Color("#48484A")
	DarkerGray     = lipgloss.Color("#3A3A3C")
	BackgroundDark = lipgloss.Color("#1C1C1E") // Был захардкожен как #1C1C1E
)

// --- 2. SEMANTIC ROLES (Назначение цветов) ---
// Используем переменные палитры для назначения ролей

var (
	// Main UI
	PrimaryColor   = Blue
	SecondaryColor = Purple
	BorderColor    = DarkerGray

	// Text
	MainTextColor  = White
	MutedTextColor = MediumGray // Для метаданных, заголовков таблиц

	// Backgrounds
	PanelBgColor          = BackgroundDark
	SelectionBgColor      = DarkGray
	InputBgColor          = DarkerGray
	StatusBarBgColor      = DarkerGray

	// Statuses
	SuccessColor             = Green
	WarningColor             = Orange
	ErrorColor               = Red
	HighlightColor           = Yellow
	SelectionHighlightColor  = Gold
)

// --- 3. LEGACY SUPPORT ---
// Сохраняем старые имена для обратной совместимости

var (
	// Deprecated: Use MainTextColor instead
	GrayColor      = Gray
	LightGrayColor = LightGray
	DarkGrayColor  = DarkGray
)

// --- 4. FILE TREE DIFF COLORS ---
var (
	DiffAddedColor    = lipgloss.Color("#A3BE8C") // Оставляем специфичные оттенки
	DiffRemovedColor  = lipgloss.Color("#BF616A")
	DiffModifiedColor = lipgloss.Color("#EBCB8B")
	DiffNormalColor   = lipgloss.Color("#D8DEE9")
)
