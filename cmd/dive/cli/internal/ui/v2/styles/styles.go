package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// --- Colors ---
var (
	PrimaryColor   = lipgloss.Color("#007AFF")
	SecondaryColor = lipgloss.Color("#5856D6")
	SuccessColor   = lipgloss.Color("#34C759")
	WarningColor   = lipgloss.Color("#FF9500")
	ErrorColor     = lipgloss.Color("#FF3B30")
	GrayColor      = lipgloss.Color("#8E8E93")
	LightGrayColor = lipgloss.Color("#C7C7CC")
	DarkGrayColor  = lipgloss.Color("#48484A")
	BorderColor    = lipgloss.Color("#3A3A3C")
)

// --- Base Styles ---
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Background(lipgloss.Color("#1C1C1E")).
			Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(SecondaryColor).
			Padding(0, 1)

	FilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(DarkGrayColor).
			Padding(0, 1)
)

// --- Component Styles ---

// RenderBox создает рамку с заголовком.
// ИСПРАВЛЕНО: Обрезает заголовок чтобы гарантировать одну строку
func RenderBox(title string, width, height int, content string, isSelected bool) string {
	// 1. Защита минимальных размеров
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}

	borderColor := BorderColor
	if isSelected {
		borderColor = PrimaryColor
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2)

	if title == "" {
		if content == "" {
			content = " "
		}
		return boxStyle.Render(content)
	}

	// 2. ИСПРАВЛЕНИЕ: Обрезаем заголовок, чтобы он не переносился на 2 строки
	// Ширина заголовка: Ширина окна - 2 (рамки) - 2 (запас)
	maxTitleWidth := width - 4
	if maxTitleWidth < 0 {
		maxTitleWidth = 0
	}

	truncatedTitle := runewidth.Truncate(title, maxTitleWidth, "…")

	// 3. Рендерим заголовок
	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	titleLine := titleStyle.Render(truncatedTitle)

	// 4. Собираем контент: Заголовок + Пробел + Данные
	// Используем " " (пробел), чтобы гарантировать высоту отступа в 1 строку
	innerContent := lipgloss.JoinVertical(lipgloss.Left, titleLine, " ", content)

	return boxStyle.Render(innerContent)
}

// --- Specific Styles for content ---

var (
	SelectedLayerStyle = lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Background(lipgloss.Color("#1C1C1E"))
	LayerHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(SecondaryColor)
	LayerValueStyle    = lipgloss.NewStyle().Foreground(LightGrayColor)

	FileTreeDirStyle      = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true)
	FileTreeModifiedStyle = lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
)

// --- Icons ---
var (
	IconDirOpen   = "📂 "
	IconDirClosed = "📁 "
	IconFile      = "📄 "
	IconSymlink   = "🔗 "
	IconAdded     = "✨ "
	IconRemoved   = "❌ "
	IconModified  = "✏️ "

	DiffAddedColor    = lipgloss.Color("#A3BE8C")
	DiffRemovedColor  = lipgloss.Color("#BF616A")
	DiffModifiedColor = lipgloss.Color("#EBCB8B")
	DiffNormalColor   = lipgloss.Color("#D8DEE9")
)

// TruncateString обрезает строку по визуальной ширине
func TruncateString(s string, maxLen int) string {
	return runewidth.Truncate(s, maxLen, "...")
}
