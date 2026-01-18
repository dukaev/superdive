package filetree

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/filetree"
)

// RenderNodeWithCursor renders a node with tree guides and improved visual design
func RenderNodeWithCursor(sb *strings.Builder, node *filetree.FileNode, prefix string, isSelected bool, width int) {
	if node == nil {
		return
	}
	row := RenderRow(node, prefix, "", isSelected, width, nil)
	sb.WriteString(row)
	sb.WriteString("\n")
}

// RenderRow renders a single tree node row with DYNAMIC column hiding
// displayName is optional - if empty, node.Name will be used
// filterRegex is optional - if provided, matching text will be highlighted
func RenderRow(node *filetree.FileNode, prefix string, displayName string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	// 1. Determine visibility based on width
	showSize, showUID, showPerm := getColumnVisibility(width)

	// --- Standard Rendering Logic (Icon, Color, etc) ---
	icon := styles.IconFile
	color := styles.DiffNormalColor
	if node.Data.FileInfo.IsDir() {
		if node.Data.ViewInfo.Collapsed {
			icon = styles.IconDirClosed
		} else {
			icon = styles.IconDirOpen
		}
	} else if node.Data.FileInfo.TypeFlag == 16 {
		icon = styles.IconSymlink
	}

	// Diff status color
	switch node.Data.DiffType {
	case filetree.Added:
		color = styles.DiffAddedColor
	case filetree.Removed:
		color = styles.DiffRemovedColor
	case filetree.Modified:
		color = styles.DiffModifiedColor
	}

	// --- Metadata Formatting ---
	var sizeStr, uidGid, perm string

	// Only compute strings if they will be shown (optimization)
	if showSize && !node.Data.FileInfo.IsDir() {
		size := node.Data.FileInfo.Size
		if size >= 0 {
			sizeStr = utils.FormatSize(uint64(size))
		}
	}
	if showUID {
		if node.Data.FileInfo.Uid != 0 || node.Data.FileInfo.Gid != 0 {
			uidGid = FormatUIDGid(node.Data.FileInfo.Uid, node.Data.FileInfo.Gid)
		} else {
			uidGid = "-"
		}
	}
	if showPerm {
		perm = FormatPermissions(node.Data.FileInfo.Mode)
	}

	// Name resolution
	name := displayName
	if name == "" {
		name = node.Name
	}
	if name == "" {
		name = "/"
	}
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// Background
	var bg lipgloss.TerminalColor = lipgloss.Color("") // Default empty color (transparent)
	if isSelected {
		bg = styles.SelectionBgColor // CompleteAdaptiveColor
	}

	// Prefix & Icon Styles
	prefixStyle := lipgloss.NewStyle().Foreground(styles.DarkGray).Background(bg)
	styledPrefix := prefixStyle.Render(prefix)
	iconStyle := lipgloss.NewStyle().Background(bg)
	styledIcon := iconStyle.Render(icon)

	// --- Build Meta Block Dynamically ---
	metaColor := styles.MutedTextColor
	gapStyle := lipgloss.NewStyle().Width(len(MetaGap)).Background(bg)

	var metaCells []string

	// 1. Size (Priority 3 - Last to hide)
	if showSize {
		sizeCell := lipgloss.NewStyle().Width(SizeWidth).Align(lipgloss.Right).Foreground(metaColor).Background(bg).Render(sizeStr)
		metaCells = append(metaCells, sizeCell)
	}

	// 2. UID:GID (Priority 2)
	if showUID {
		if len(metaCells) > 0 {
			metaCells = append(metaCells, gapStyle.Render(MetaGap))
		}
		uidGidCell := lipgloss.NewStyle().Width(UIDGidWidth).Align(lipgloss.Right).Foreground(metaColor).Background(bg).Render(uidGid)
		metaCells = append(metaCells, uidGidCell)
	}

	// 3. Permissions (Priority 1 - First to hide)
	if showPerm {
		if len(metaCells) > 0 {
			metaCells = append(metaCells, gapStyle.Render(MetaGap))
		}
		permCell := lipgloss.NewStyle().Width(PermWidth).Align(lipgloss.Right).Foreground(metaColor).Background(bg).Render(perm)
		metaCells = append(metaCells, permCell)
	}

	// Join meta block
	metaBlock := lipgloss.JoinHorizontal(lipgloss.Top, metaCells...)
	metaBlockWidth := lipgloss.Width(metaBlock)

	// --- Name Rendering & Truncation ---

	// Calculate available width for name
	fixedPartWidth := lipgloss.Width(styledPrefix) + lipgloss.Width(styledIcon)

	// -2 for spacing/padding
	availableForName := width - fixedPartWidth - metaBlockWidth - 2

	// Absolute minimum safety
	if availableForName < 5 {
		availableForName = 5
	}

	// Truncate name logic
	var finalStyledName string
	truncatedName := name

	// Render name with highlight or normal style
	if filterRegex != nil && filterRegex.MatchString(name) {
		// Highlight matching portions
		if runewidth.StringWidth(name) > availableForName {
			truncatedName = smartTruncatePath(name, availableForName)
		}
		// Note: Highlight matching on truncated string is tricky, simplified here:
		if filterRegex.MatchString(truncatedName) {
			finalStyledName = highlightMatches(truncatedName, filterRegex, color, isSelected, bg)
		} else {
			// Fallback if truncation cut off the match
			nameStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
			if isSelected {
				nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
			}
			finalStyledName = nameStyle.Render(truncatedName)
		}
	} else {
		if runewidth.StringWidth(name) > availableForName {
			truncatedName = smartTruncatePath(name, availableForName)
		}
		nameStyle := lipgloss.NewStyle().Foreground(color).Background(bg)
		if isSelected {
			nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
		}
		finalStyledName = nameStyle.Render(truncatedName)
	}

	// --- Final Assembly ---

	// Calculate flexible padding
	contentWidth := fixedPartWidth + lipgloss.Width(finalStyledName) + metaBlockWidth
	paddingNeeded := width - contentWidth
	if paddingNeeded < 1 {
		paddingNeeded = 1
	}

	padding := lipgloss.NewStyle().Width(paddingNeeded).Background(bg).Render(strings.Repeat(" ", paddingNeeded))

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styledPrefix,
		styledIcon,
		finalStyledName,
		padding,
		metaBlock,
	)
}

// highlightMatches applies regex highlighting to matching portions of the text
// baseColor and bg can be lipgloss.Color or lipgloss.CompleteAdaptiveColor (both implement TerminalColor)
func highlightMatches(text string, filter *regexp.Regexp, baseColor lipgloss.TerminalColor, isSelected bool, bg lipgloss.TerminalColor) string {
	// Find all matches
	matches := filter.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// Should not happen since we check MatchString before calling
		nameStyle := lipgloss.NewStyle().Foreground(baseColor).Background(bg)
		if isSelected {
			nameStyle = nameStyle.Bold(true).Foreground(styles.PrimaryColor)
		}
		return nameStyle.Render(text)
	}

	// Build highlighted string
	var result strings.Builder
	lastEnd := 0

	// Highlight color: use centralized colors
	highlightColor := styles.HighlightColor
	if isSelected {
		highlightColor = styles.SelectionHighlightColor
	}

	normalStyle := lipgloss.NewStyle().Foreground(baseColor).Background(bg)
	if isSelected {
		normalStyle = normalStyle.Bold(true).Foreground(styles.PrimaryColor)
	}

	highlightStyle := lipgloss.NewStyle().Foreground(highlightColor).Background(bg)
	if isSelected {
		highlightStyle = highlightStyle.Bold(true)
	}

	for _, match := range matches {
		// Add non-matching text before this match
		if match[0] > lastEnd {
			result.WriteString(normalStyle.Render(text[lastEnd:match[0]]))
		}

		// Add matching text with highlight
		result.WriteString(highlightStyle.Render(text[match[0]:match[1]]))

		lastEnd = match[1]
	}

	// Add remaining text after last match
	if lastEnd < len(text) {
		result.WriteString(normalStyle.Render(text[lastEnd:]))
	}

	return result.String()
}

// RenderNodeLine renders a single node line for viewport.
// This is a convenience wrapper around RenderRow.
func RenderNodeLine(node *filetree.FileNode, prefix string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	return RenderRow(node, prefix, "", isSelected, width, filterRegex)
}

// RenderNodeLineWithDisplayName renders a single node line with a custom display name.
// This is used for flat view where the full path is shown instead of just the name.
func RenderNodeLineWithDisplayName(node *filetree.FileNode, prefix string, displayName string, isSelected bool, width int, filterRegex *regexp.Regexp) string {
	return RenderRow(node, prefix, displayName, isSelected, width, filterRegex)
}

// smartTruncatePath сокращает путь, отдавая приоритет имени файла.
// Пример: "/very/long/path/to/file.txt" -> "…/path/to/file.txt"
func smartTruncatePath(path string, width int) string {
	if runewidth.StringWidth(path) <= width {
		return path
	}

	// 1. Пытаемся разделить на директорию и файл
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		// Если слэшей нет, это просто длинное имя файла.
		// Используем стандартное обрезание
		return runewidth.Truncate(path, width, "…")
	}

	dir := path[:lastSlash]
	name := path[lastSlash+1:]
	nameWidth := runewidth.StringWidth(name)

	// 2. Если имя файла + "/" занимает почти всю ширину (осталось < 5 символов для директории)
	// то просто обрезаем весь путь стандартным способом
	if nameWidth+1 > width-5 {
		return runewidth.Truncate(path, width, "…")
	}

	// 3. Имя файла влезает, вычисляем место для директории
	// Нужно место под name + 1 символ слэша
	availableForDir := width - nameWidth - 1

	if availableForDir < 3 {
		// Совсем нет места под директорию
		return "…/" + name
	}

	// 4. Если dir влезает целиком, возвращаем как есть
	if runewidth.StringWidth(dir) <= availableForDir {
		return dir + "/" + name
	}

	// 5. Иначе обрезаем начало директории, оставляя "…" в начале
	// Берём руну за руной с конца, пока не наберём нужную ширину
	dirRunes := []rune(dir)
	var truncatedRunes []rune
	currentWidth := 1 // Ширина "…"

	// Идём с конца директории
	for i := len(dirRunes) - 1; i >= 0; i-- {
		r := dirRunes[i]
		rWidth := runewidth.RuneWidth(r)

		if currentWidth+rWidth > availableForDir {
			break
		}

		currentWidth += rWidth
		truncatedRunes = append([]rune{r}, truncatedRunes...)
	}

	truncatedDir := "…" + string(truncatedRunes)

	return truncatedDir + "/" + name
}
