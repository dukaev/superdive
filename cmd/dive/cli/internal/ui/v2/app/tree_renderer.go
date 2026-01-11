package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/dive/filetree"
)

// VisibleNode represents a node with its depth for rendering
type VisibleNode struct {
	Node  *filetree.FileNode
	Depth int
}

// renderTreeContent генерирует красивую строку дерева с иконками и цветами
func (m Model) renderTreeContent() string {
	if m.treeVM == nil {
		return "Tree viewer not initialized"
	}

	if m.treeVM.ViewTree == nil {
		return "Loading tree..."
	}

	if m.treeVM.ViewTree.Root == nil {
		return "Empty tree"
	}

	var sb strings.Builder

	// Collect all visible nodes
	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)

	// Adjust treeIndex if out of bounds
	if m.treeIndex >= len(visibleNodes) {
		m.treeIndex = len(visibleNodes) - 1
	}
	if m.treeIndex < 0 {
		m.treeIndex = 0
	}

	// Render nodes with cursor indicator
	for i, vn := range visibleNodes {
		isSelected := (i == m.treeIndex)
		// Pass viewport width for full-width selection highlight
		renderNodeWithCursor(&sb, vn.Node, vn.Depth, isSelected, m.treeViewport.Width)
	}

	return sb.String()
}

// collectVisibleNodes collects all visible nodes in a flat list
func collectVisibleNodes(root *filetree.FileNode) []VisibleNode {
	var nodes []VisibleNode

	var traverse func(*filetree.FileNode, int)
	traverse = func(node *filetree.FileNode, depth int) {
		if node == nil {
			return
		}

		// Skip root node itself, start from children
		if node.Parent != nil {
			nodes = append(nodes, VisibleNode{Node: node, Depth: depth})
		}

		// Recurse into children if directory and not collapsed
		if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed {
			sortedChildren := sortChildren(node.Children)
			for _, child := range sortedChildren {
				traverse(child, depth+1)
			}
		}
	}

	// Start from root's children if root is not collapsed
	if !root.Data.ViewInfo.Collapsed {
		sortedChildren := sortChildren(root.Children)
		for _, child := range sortedChildren {
			traverse(child, 0)
		}
	}

	return nodes
}

// renderNodeWithCursor renders a node with optional cursor indicator
func renderNodeWithCursor(sb *strings.Builder, node *filetree.FileNode, depth int, isSelected bool, width int) {
	if node == nil {
		return
	}

	// 1. Cursor indicator
	var cursor string
	if isSelected {
		cursor = "▸ "
	} else {
		cursor = "  "
	}

	// 2. Icon and color
	icon := v2styles.IconFile
	diffIcon := ""
	color := v2styles.DiffNormalColor

	if node.Data.FileInfo.IsDir() {
		if node.Data.ViewInfo.Collapsed {
			icon = v2styles.IconDirClosed
		} else {
			icon = v2styles.IconDirOpen
		}
	} else if node.Data.FileInfo.TypeFlag == 16 { // Symlink
		icon = v2styles.IconSymlink
	}

	// 3. Diff status
	switch node.Data.DiffType {
	case filetree.Added:
		color = v2styles.DiffAddedColor
		diffIcon = v2styles.IconAdded
	case filetree.Removed:
		color = v2styles.DiffRemovedColor
		diffIcon = v2styles.IconRemoved
	case filetree.Modified:
		color = v2styles.DiffModifiedColor
		diffIcon = v2styles.IconModified
	}

	// 4. Format name
	name := node.Name
	if name == "" {
		name = "/"
	}
	// Symlinks
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// 5. Indent
	indent := strings.Repeat("  ", depth)

	// 6. Build line (without styles yet)
	// Add space after diffIcon if present
	if diffIcon != "" {
		diffIcon += " "
	}

	rawText := fmt.Sprintf("%s%s%s%s", indent+cursor, diffIcon, icon, name)

	// ВАЖНО: Truncate to prevent line wrapping which breaks click detection
	// width is the viewport content width. Leave small margin to ensure no wrapping
	maxTextWidth := width
	if maxTextWidth < 10 {
		maxTextWidth = 10 // Protection
	}

	truncatedText := runewidth.Truncate(rawText, maxTextWidth, "…")

	// 7. Apply style
	style := lipgloss.NewStyle().Foreground(color)
	if isSelected {
		// --- FIX 3: Force width to 100% of viewport for selected items ---
		style = style.
			Background(v2styles.PrimaryColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Width(width).     // Explicitly set width to 100%
			MaxWidth(width)   // Prevent exceeding width
	}

	sb.WriteString(style.Render(truncatedText))
	sb.WriteString("\n")

	// Note: no recursion here since we're using collectVisibleNodes instead
}

// renderNode рекурсивно рендерит узел дерева с иконками и цветами
func renderNode(sb *strings.Builder, node *filetree.FileNode, depth int, prefix string) {
	if node == nil {
		return
	}

	// Не рендерим корневой элемент (он обычно пустой)
	if node.Parent == nil {
		// Рендерим детей корня
		if !node.Data.ViewInfo.Collapsed {
			sortedChildren := sortChildren(node.Children)
			for _, child := range sortedChildren {
				renderNode(sb, child, depth, "")
			}
		}
		return
	}

	// 1. Определяем иконку
	icon := v2styles.IconFile
	diffIcon := ""

	// Определяем тип файла
	if node.Data.FileInfo.IsDir() {
		if node.Data.ViewInfo.Collapsed {
			icon = v2styles.IconDirClosed
		} else {
			icon = v2styles.IconDirOpen
		}
	} else if node.Data.FileInfo.TypeFlag == 16 { // tar.TypeSymlink
		icon = v2styles.IconSymlink
	}

	// Определяем Diff (Добавлен/Удален/Изменен)
	color := v2styles.DiffNormalColor

	switch node.Data.DiffType {
	case filetree.Added:
		color = v2styles.DiffAddedColor
		diffIcon = v2styles.IconAdded
	case filetree.Removed:
		color = v2styles.DiffRemovedColor
		diffIcon = v2styles.IconRemoved
	case filetree.Modified:
		color = v2styles.DiffModifiedColor
		diffIcon = v2styles.IconModified
	}

	// 2. Формируем строку
	name := node.Name
	if name == "" {
		name = "/"
	}

	// Добавляем symlink target если есть
	if node.Data.FileInfo.TypeFlag == 16 && node.Data.FileInfo.Linkname != "" {
		name += " → " + node.Data.FileInfo.Linkname
	}

	// Собираем строку с префиксом (отступом)
	line := prefix + diffIcon + " " + icon + " " + name

	// Применяем цвет
	style := lipgloss.NewStyle().Foreground(color)
	sb.WriteString(style.Render(line))
	sb.WriteString("\n")

	// 3. Рекурсия для детей (если папка не свернута)
	if node.Data.FileInfo.IsDir() && !node.Data.ViewInfo.Collapsed && !node.IsLeaf() {
		// Вычисляем префикс для детей
		childPrefix := prefix + "  "

		// Сортируем и рендерим детей
		sortedChildren := sortChildren(node.Children)
		for _, child := range sortedChildren {
			renderNode(sb, child, depth+1, childPrefix)
		}
	}
}

// sortChildren сортирует детей узла: сначала папки, потом файлы, все по алфавиту
func sortChildren(children map[string]*filetree.FileNode) []*filetree.FileNode {
	if children == nil {
		return nil
	}

	// Разделяем на папки и файлы
	var dirs []*filetree.FileNode
	var files []*filetree.FileNode

	for _, child := range children {
		if child.Data.FileInfo.IsDir() {
			dirs = append(dirs, child)
		} else {
			files = append(files, child)
		}
	}

	// Сортируем папки
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name < dirs[j].Name
	})

	// Сортируем файлы
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Объединяем: сначала папки, потом файлы
	result := append(dirs, files...)
	return result
}
