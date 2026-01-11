package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	v1 "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/dive/image"
	v2styles "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// Pane represents a UI pane
type Pane int

const (
	PaneLayer Pane = iota
	PaneDetails
	PaneImage
	PaneTree
)

// Layout constants for viewport calculations
const (
	borderHeight         = 2 // Top + Bottom border lines
	headerHeight         = 2 // Title line + newline/padding separator
	boxContentPadding    = borderHeight + headerHeight // Total padding inside RenderBox
	contentVisualOffset  = 3 // Offset for mouse hit testing: 1 border + 1 title + 1 padding
)

func (p Pane) String() string {
	switch p {
	case PaneLayer:
		return "Layers"
	case PaneDetails:
		return "Details"
	case PaneImage:
		return "Image"
	case PaneTree:
		return "Tree"
	}
	return "Unknown"
}

// LayoutCache stores calculated pane dimensions to avoid recalculating in View and Mouse
type LayoutCache struct {
	ContentStartY int
	LeftWidth     int
	RightWidth    int
	LayersHeight  int
	DetailsHeight int
	ImageHeight   int
	TreeHeight    int
}

// Model is the bubbletea Model for V2UI
type Model struct {
	// Configuration
	analysis image.Analysis
	content  image.ContentReader
	prefs    v1.Preferences
	ctx      context.Context

	// Viewmodels (SHARED with V1UI!)
	layerVM *viewmodel.LayerSetState
	treeVM  *viewmodel.FileTreeViewModel

	// UI state
	width    int
	height   int
	quitting bool
	layout   LayoutCache

	// Viewports for scrolling
	layersViewport viewport.Model
	treeViewport   viewport.Model
	imageViewport  viewport.Model

	// Active pane state
	activePane Pane

	// Tree navigation state
	treeIndex int // Current selection in tree (0-based)

	// Filter state
	filter FilterModel

	// Help and key bindings
	keys KeyMap
	help help.Model
}

// NewModel creates a new bubbletea model with configuration
func NewModel(analysis image.Analysis, content image.ContentReader, prefs v1.Preferences, ctx context.Context) Model {
	// Initialize layer viewmodel
	var layerVM *viewmodel.LayerSetState
	if analysis.Layers != nil && len(analysis.Layers) > 0 {
		layerVM = viewmodel.NewLayerSetState(
			analysis.Layers,
			viewmodel.CompareSingleLayer,
		)
	}

	// Initialize filetree viewmodel
	var treeVM *viewmodel.FileTreeViewModel
	if len(analysis.RefTrees) > 0 {
		v1cfg := v1.Config{
			Analysis:    analysis,
			Content:     content,
			Preferences: prefs,
		}
		// Note: we ignore the error here since treeVM.Update() will be called in Init()
		treeVM, _ = viewmodel.NewFileTreeViewModel(v1cfg, 0)
	}

	h := help.New()
	h.Width = 80
	h.Styles.ShortKey = v2styles.StatusStyle
	h.Styles.ShortDesc = v2styles.StatusStyle
	h.Styles.Ellipsis = v2styles.StatusStyle

	f := NewFilterModel()

	return Model{
		analysis:       analysis,
		content:        content,
		prefs:          prefs,
		ctx:            ctx,
		layerVM:        layerVM,
		treeVM:         treeVM,
		width:          80,
		height:         24,
		quitting:       false,
		layersViewport: viewport.New(80, 20),
		treeViewport:   viewport.New(80, 20),
		imageViewport:  viewport.New(80, 20),
		activePane:     PaneLayer,
		keys:           Keys,
		help:           h,
		filter:         f,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// IMPORTANT: Initialize tree on startup so it's not empty!
	if m.layerVM != nil && m.treeVM != nil {
		m.updateTreeForCurrentLayer()
	}
	return nil
}

// recalculateLayout is the single source of truth for pane dimensions
func (m *Model) recalculateLayout() {
	titleContent := v2styles.TitleStyle.Render(" Dive V2UI ")
	titleHeight := lipgloss.Height(titleContent)

	statusBarHeight := 1

	m.layout.ContentStartY = titleHeight

	// Subtract 1 extra line for safety to prevent terminal auto-scroll
	availableHeight := m.height - titleHeight - statusBarHeight - 1
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Widths
	m.layout.LeftWidth = (m.width - 4) / 2
	if m.layout.LeftWidth < 20 {
		m.layout.LeftWidth = 20
	}
	m.layout.RightWidth = m.width - m.layout.LeftWidth - 2
	if m.layout.RightWidth < 20 {
		m.layout.RightWidth = 20
	}

	// Heights (left column: 40%, 20%, 40%)
	m.layout.LayersHeight = availableHeight * 2 / 5
	if m.layout.LayersHeight < 5 {
		m.layout.LayersHeight = 5
	}

	m.layout.DetailsHeight = availableHeight * 1 / 5
	if m.layout.DetailsHeight < 3 {
		m.layout.DetailsHeight = 3
	}

	m.layout.ImageHeight = availableHeight - m.layout.LayersHeight - m.layout.DetailsHeight
	if m.layout.ImageHeight < 5 {
		m.layout.ImageHeight = 5
	}

	m.layout.TreeHeight = availableHeight
}

// updateViewportsContent generates and sets content for all viewports
// CRITICAL: This must be called from Update(), NOT from View()
// View() must be pure (read-only) according to BubbleTea architecture
func (m *Model) updateViewportsContent() {
	// Update layers viewport content
	if m.layerVM != nil && len(m.layerVM.Layers) > 0 {
		content := m.generateLayersContent()
		m.layersViewport.SetContent(content)
	}

	// Update tree viewport content
	if m.treeVM != nil {
		content := m.renderTreeContent()
		if content == "" {
			content = "(File tree rendering in progress...)"
		}
		m.treeViewport.SetContent(content)
	}

	// Update image viewport content
	content := m.generateImageContent()
	m.imageViewport.SetContent(content)
}

// generateLayersContent creates the full layers content string (for SetContent)
func (m *Model) generateLayersContent() string {
	if m.layerVM == nil || len(m.layerVM.Layers) == 0 {
		return "No layer data"
	}

	width := m.layout.LeftWidth - 2 // Subtract borders

	const (
		idWidth   = 12 // ID image (truncate to 12 chars)
		sizeWidth = 9  // Size (e.g., "999.9 MB")
		spaces    = 4  // Spaces between columns (2 after ID, 2 after Size)
	)

	var fullContent strings.Builder

	for i, layer := range m.layerVM.Layers {
		// 1. Selection marker
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.layerVM.LayerIndex {
			prefix = "● "
			style = v2styles.SelectedLayerStyle
		}

		// 2. Prepare ID (truncate if too long)
		id := layer.Id
		if len(id) > idWidth {
			id = id[:idWidth]
		}

		// 3. Prepare Size
		size := formatSize(layer.Size)

		// 4. Prepare Command (IMPORTANT: remove newlines!)
		rawCmd := strings.ReplaceAll(layer.Command, "\n", " ")
		rawCmd = strings.TrimSpace(rawCmd)

		// 5. Calculate available width for command
		availableCmdWidth := width - 2 - idWidth - spaces - sizeWidth

		// Minimum width protection for narrow terminals
		if availableCmdWidth < 5 {
			availableCmdWidth = 0
		}

		// Truncate command to fit available width
		cmd := ""
		if availableCmdWidth > 0 && rawCmd != "" {
			cmd = runewidth.Truncate(rawCmd, availableCmdWidth, "...")
		}

		// 6. Format string with fixed column widths
		text := fmt.Sprintf("%s%-*s  %*s  %s", prefix, idWidth, id, sizeWidth, size, cmd)

		// Protect against overflow
		maxLineWidth := width
		if runewidth.StringWidth(text) > maxLineWidth {
			text = runewidth.Truncate(text, maxLineWidth, "")
		}

		fullContent.WriteString(style.Render(text))
		fullContent.WriteString("\n")
	}

	return fullContent.String()
}

// generateImageContent creates the full image details content string (for SetContent)
func (m *Model) generateImageContent() string {
	width := m.layout.LeftWidth - 2 // Subtract borders

	// Header with statistics
	headerText := fmt.Sprintf(
		"Image name: %s\nTotal Image size: %s\nPotential wasted space: %s\nImage efficiency score: %.0f%%",
		m.analysis.Image,
		formatSize(m.analysis.SizeBytes),
		formatSize(m.analysis.WastedBytes),
		m.analysis.Efficiency*100,
	)

	// Table header
	tableHeader := fmt.Sprintf("\n%-5s %-12s %s", "Count", "Total Space", "Path")

	// Build full content with all rows
	var fullContent strings.Builder
	fullContent.WriteString(headerText)
	fullContent.WriteString("\n")
	fullContent.WriteString(v2styles.LayerHeaderStyle.Render(tableHeader))
	fullContent.WriteString("\n")

	if len(m.analysis.Inefficiencies) > 0 {
		for _, file := range m.analysis.Inefficiencies {
			row := fmt.Sprintf("%-5d %-12s %s", len(file.Nodes), formatSize(uint64(file.CumulativeSize)), file.Path)
			if lipgloss.Width(row) > width {
				row = runewidth.Truncate(row, width, "...")
			}
			fullContent.WriteString(v2styles.FileTreeModifiedStyle.Render(row))
			fullContent.WriteString("\n")
		}
	} else {
		fullContent.WriteString("No inefficiencies detected - great job!")
	}

	return fullContent.String()
}

// syncTreeScroll ensures the cursor (treeIndex) is always visible in the viewport
func (m *Model) syncTreeScroll() {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return
	}

	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)
	if len(visibleNodes) == 0 {
		return
	}

	// Adjust treeIndex if out of bounds
	if m.treeIndex >= len(visibleNodes) {
		m.treeIndex = len(visibleNodes) - 1
	}
	if m.treeIndex < 0 {
		m.treeIndex = 0
	}

	// Get viewport visible area
	visibleHeight := m.treeViewport.Height
	if visibleHeight <= 0 {
		return
	}

	// If cursor is above viewport, scroll up
	if m.treeIndex < m.treeViewport.YOffset {
		m.treeViewport.SetYOffset(m.treeIndex)
	}

	// If cursor is below visible area, scroll down
	if m.treeIndex >= m.treeViewport.YOffset+visibleHeight {
		m.treeViewport.SetYOffset(m.treeIndex - visibleHeight + 1)
	}
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Flags to control which viewports receive mouse events
	updateLayers := true
	updateTree := true
	updateImage := true

	switch msg := msg.(type) {
	case tea.KeyMsg:
		newM, newCmd := m.handleKeyPress(msg)
		m = newM.(Model)
		cmds = append(cmds, newCmd)

		// --- ИСПРАВЛЕНИЕ: Double Update Fix ---
		// Если нажата кнопка, мы сами управляем навигацией через handleKeyPress.
		// Запрещаем вьюпортам обрабатывать нажатия (чтобы они не скроллили сами по себе).
		updateLayers = false
		updateTree = false
		updateImage = false

	case tea.MouseMsg:
		// --- FIX 1: Smart scroll - only update the viewport under the mouse ---
		x, y := msg.X, msg.Y
		l := m.layout

		inLeftCol := x >= 0 && x < l.LeftWidth
		inRightCol := x >= l.LeftWidth && x < m.width

		// If this is a mouse wheel event, disable update for non-target panes
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			if inLeftCol {
				updateTree = false // Mouse on left side, don't scroll tree

				// Determine if mouse is over Layers or Image
				layersEndY := l.ContentStartY + l.LayersHeight
				detailsEndY := layersEndY + l.DetailsHeight

				if y < layersEndY {
					updateImage = false // Mouse over Layers
				} else if y >= detailsEndY {
					updateLayers = false // Mouse over Image
				}
			} else if inRightCol {
				updateLayers = false
				updateImage = false
			}
		}

		// Handle clicks and application logic
		newModel, newCmd := m.handleMouse(msg)
		if nm, ok := newModel.(Model); ok {
			m = nm
		}
		cmds = append(cmds, newCmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.recalculateLayout()

	default:
		// For regular messages (timers, etc.), update everything
	}

	// --- Recalculate viewport sizes ---
	safeLayersH := m.layout.LayersHeight - boxContentPadding
	if safeLayersH < 0 {
		safeLayersH = 0
	}
	m.layersViewport.Width = m.layout.LeftWidth - 2
	m.layersViewport.Height = safeLayersH

	safeImageH := m.layout.ImageHeight - boxContentPadding
	if safeImageH < 0 {
		safeImageH = 0
	}
	m.imageViewport.Width = m.layout.LeftWidth - 2
	m.imageViewport.Height = safeImageH

	safeTreeH := m.layout.TreeHeight - boxContentPadding
	if safeTreeH < 0 {
		safeTreeH = 0
	}
	m.treeViewport.Width = m.layout.RightWidth - 2
	m.treeViewport.Height = safeTreeH

	// --- Update viewports (only the ones that need it) ---
	if updateLayers {
		m.layersViewport, cmd = m.layersViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if updateImage {
		m.imageViewport, cmd = m.imageViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if updateTree {
		m.treeViewport, cmd = m.treeViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.help, cmd = m.help.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewports content (CRITICAL: Must be in Update(), NOT View())
	m.updateViewportsContent()

	// Sync tree scroll to keep cursor visible
	if m.activePane == PaneTree {
		m.syncTreeScroll()
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If filter is visible, let filter handle keys
	if m.filter.IsVisible() {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit

	case "tab", "shift+tab":
		m.togglePane()

	case "up", "k":
		m.navigateUp()

	case "down", "j":
		m.navigateDown()

	case "enter", " ":
		// Toggle folder collapse when tree is active
		if m.activePane == PaneTree && m.treeVM != nil {
			m.toggleCurrentNodeCollapse()
		}

	case "ctrl+f", "/":
		m.filter.Show()
	}

	return m, nil
}

func (m *Model) togglePane() {
	// Cycle through panes: Layer -> Details -> Image -> Tree -> Layer
	m.activePane++
	if m.activePane > PaneTree {
		m.activePane = PaneLayer
	}
}

func (m *Model) navigateUp() {
	switch m.activePane {
	case PaneLayer:
		m.layerUp()
	case PaneImage:
		m.imageViewport.LineUp(1)
	case PaneTree:
		m.treeUp()
	case PaneDetails:
		// Details has no scroll
	}
}

func (m *Model) navigateDown() {
	switch m.activePane {
	case PaneLayer:
		m.layerDown()
	case PaneImage:
		m.imageViewport.LineDown(1)
	case PaneTree:
		m.treeDown()
	case PaneDetails:
		// Details has no scroll
	}
}

func (m *Model) treeUp() {
	if m.treeIndex > 0 {
		m.treeIndex--
	}
}

func (m *Model) treeDown() {
	if m.treeVM != nil && m.treeVM.ViewTree != nil {
		visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)
		if m.treeIndex < len(visibleNodes)-1 {
			m.treeIndex++
		}
	}
}

func (m *Model) layerUp() {
	if m.layerVM != nil && m.layerVM.LayerIndex > 0 {
		m.layerVM.LayerIndex--
		m.updateTreeForCurrentLayer()
	}
}

func (m *Model) layerDown() {
	if m.layerVM != nil && m.layerVM.LayerIndex < len(m.layerVM.Layers)-1 {
		m.layerVM.LayerIndex++
		m.updateTreeForCurrentLayer()
	}
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Use cached layout for hit testing
	x, y := msg.X, msg.Y
	l := m.layout

	inLeftCol := x >= 0 && x < l.LeftWidth
	inRightCol := x >= l.LeftWidth && x < m.width

	// Calculate Y boundaries
	layersEndY := l.ContentStartY + l.LayersHeight
	detailsEndY := layersEndY + l.DetailsHeight
	imageEndY := detailsEndY + l.ImageHeight

	// Handle mouse wheel
	if msg.Action == tea.MouseActionPress {
		if msg.Button == tea.MouseButtonWheelUp {
			if inLeftCol {
				if y < layersEndY {
					m.layerUp()
				} else if y >= detailsEndY && y < imageEndY {
					m.imageViewport.LineUp(1)
				}
			} else if inRightCol {
				m.treeUp()
			}
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if inLeftCol {
				if y < layersEndY {
					m.layerDown()
				} else if y >= detailsEndY && y < imageEndY {
					m.imageViewport.LineDown(1)
				}
			} else if inRightCol {
				m.treeDown()
			}
			return m, nil
		}
	}

	// Handle left click for pane switching and item selection
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	// Hit testing with cached layout
	if inLeftCol {
		if y >= l.ContentStartY && y < layersEndY {
			m.activePane = PaneLayer
			// Calculate which layer was clicked
			relativeY := y - l.ContentStartY - contentVisualOffset
			if relativeY >= 0 && relativeY < m.layersViewport.Height && m.layerVM != nil {
				targetIndex := relativeY + m.layersViewport.YOffset
				if targetIndex < len(m.layerVM.Layers) {
					m.layerVM.LayerIndex = targetIndex
					m.updateTreeForCurrentLayer()
				}
			}
		} else if y >= layersEndY && y < detailsEndY {
			m.activePane = PaneDetails
		} else if y >= detailsEndY && y < imageEndY {
			m.activePane = PaneImage
		}
	} else if inRightCol {
		m.activePane = PaneTree
		// Calculate which tree item was clicked
		// IMPORTANT: Account for viewport scroll offset AND border AND title bar!
		relativeY := y - l.ContentStartY - contentVisualOffset  // -1 for top border, -2 for title+newline
		if relativeY >= 0 && relativeY < m.treeViewport.Height && m.treeVM != nil && m.treeVM.ViewTree != nil {
			visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)

			// FIX: Add YOffset to get the actual index in the full list
			targetIndex := relativeY + m.treeViewport.YOffset

			// Each node takes 1 line
			if targetIndex >= 0 && targetIndex < len(visibleNodes) {
				// If clicking on already selected item, toggle collapse
				if m.treeIndex == targetIndex {
					m.toggleCurrentNodeCollapse()
				} else {
					m.treeIndex = targetIndex
				}
			}
		}
	}

	return m, nil
}

func (m *Model) updateTreeForCurrentLayer() {
	if m.treeVM == nil || m.layerVM == nil {
		return
	}

	bottomTreeStart, _, topTreeStart, topTreeStop := m.layerVM.GetCompareIndexes()
	_ = m.treeVM.SetTreeByLayer(bottomTreeStart, bottomTreeStart, topTreeStart, topTreeStop)

	// IMPORTANT: Call Update() to create ViewTree from ModelTree!
	// This applies filters and creates the visible tree
	_ = m.treeVM.Update(nil, m.layout.RightWidth, m.layout.TreeHeight)

	// Reset scroll when changing layers
	m.treeViewport.GotoTop()

	// Reset tree index to first item
	m.treeIndex = 0
}

func (m *Model) toggleCurrentNodeCollapse() {
	if m.treeVM == nil || m.treeVM.ViewTree == nil {
		return
	}

	// Collect visible nodes
	visibleNodes := collectVisibleNodes(m.treeVM.ViewTree.Root)

	// Adjust treeIndex if out of bounds
	if m.treeIndex >= len(visibleNodes) {
		m.treeIndex = len(visibleNodes) - 1
	}
	if m.treeIndex < 0 {
		m.treeIndex = 0
	}

	// Get the selected node
	if m.treeIndex < len(visibleNodes) {
		selectedNode := visibleNodes[m.treeIndex].Node

		// Only directories can be collapsed
		if selectedNode.Data.FileInfo.IsDir() {
			// Toggle collapse state
			selectedNode.Data.ViewInfo.Collapsed = !selectedNode.Data.ViewInfo.Collapsed

			// Regenerate ViewTree after toggling
			_ = m.treeVM.Update(nil, m.layout.RightWidth, m.layout.TreeHeight)
		}
	}
}

// View implements tea.Model (PURE FUNCTION - no side effects!)
func (m Model) View() string {
	if m.quitting {
		return v2styles.TitleStyle.Foreground(v2styles.SuccessColor).Render("Thanks for using Dive V2UI!")
	}

	// Calculate layout if not yet calculated (first run)
	if m.layout.LayersHeight == 0 {
		m.recalculateLayout()
	}

	// Render UI components
	title := v2styles.TitleStyle.Render(" Dive V2UI ")
	statusBar := m.help.View(m.keys)

	// Add active pane indicator to status bar
	paneName := v2styles.StatusStyle.Render(fmt.Sprintf(" Active: %s ", m.activePane))
	statusBar = lipgloss.JoinHorizontal(lipgloss.Top, statusBar, strings.Repeat(" ", 5), paneName)

	// Render panes using cached layout (pure - no SetContent calls!)
	layersPane := m.renderLayersPane()
	detailsPane := m.renderDetailsPane()
	imagePane := m.renderImageDetailsPane()

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, layersPane, detailsPane, imagePane)
	treePane := m.renderTreePane()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, treePane)

	base := lipgloss.JoinVertical(lipgloss.Left,
		title,
		mainContent,
		statusBar,
	)

	// Overlay filter modal if visible
	if m.filter.IsVisible() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.filter.View(m.width, m.height))
	}

	return base
}

func (m Model) renderLayersPane() string {
	width := m.layout.LeftWidth
	height := m.layout.LayersHeight

	if m.layerVM == nil || len(m.layerVM.Layers) == 0 {
		return v2styles.RenderBox("Layers", width, height, "No layer data", m.activePane == PaneLayer)
	}

	// Get viewport content (already set in Update())
	content := m.layersViewport.View()
	return v2styles.RenderBox("Layers", width, height, content, m.activePane == PaneLayer)
}

func (m Model) renderDetailsPane() string {
	width := m.layout.LeftWidth
	height := m.layout.DetailsHeight

	// 1. Вычисляем доступное место: Высота - Рамки(2) - Заголовок(2)
	maxLines := height - 4
	if maxLines < 0 {
		maxLines = 0
	}

	if m.layerVM == nil || len(m.layerVM.Layers) == 0 || m.layerVM.LayerIndex >= len(m.layerVM.Layers) {
		return v2styles.RenderBox("Layer Details", width, height, "No details", m.activePane == PaneDetails)
	}

	layer := m.layerVM.Layers[m.layerVM.LayerIndex]
	var lines []string

	// Хелпер: добавляет строку только если есть место
	addLine := func(s string) bool {
		if len(lines) < maxLines {
			lines = append(lines, s)
			return true
		}
		return false
	}

	// Tags
	if len(layer.Names) > 0 {
		tags := strings.Join(layer.Names, ", ")
		if lipgloss.Width(tags) > width-8 {
			tags = runewidth.Truncate(tags, width-8, "...")
		}
		if !addLine(v2styles.LayerHeaderStyle.Render(fmt.Sprintf("Tags: %s", tags))) {
			goto finish
		}
	}

	// ID & Size
	if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Id: %s", layer.Id))) {
		goto finish
	}
	if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Size: %s", formatSize(layer.Size)))) {
		goto finish
	}

	// Digest
	if layer.Digest != "" {
		digest := layer.Digest
		if len(digest) > 71 {
			digest = digest[:71] + "..."
		}
		if !addLine(v2styles.LayerValueStyle.Render(fmt.Sprintf("Digest: %s", digest))) {
			goto finish
		}
	}

	// Spacer
	if !addLine("") {
		goto finish
	}

	// Command
	if layer.Command != "" {
		if !addLine(v2styles.LayerHeaderStyle.Render("Command:")) {
			goto finish
		}

		maxWidth := width - 4
		if maxWidth < 20 {
			maxWidth = 20
		}

		// Разбиваем команду на строки
		wrappedCmd := lipgloss.NewStyle().Width(maxWidth).Render(layer.Command)
		wrappedLines := strings.Split(wrappedCmd, "\n")

		// Добавляем строки команды, пока есть место
		remainingLines := maxLines - len(lines)
		if remainingLines > 0 {
			for i, line := range wrappedLines {
				if i >= remainingLines {
					// Если места нет, заменяем последнюю добавленную строку на "..."
					lines[len(lines)-1] = v2styles.LayerValueStyle.Render("...")
					break
				}
				addLine(v2styles.LayerValueStyle.Render(line))
			}
		}
	} else {
		addLine(v2styles.LayerHeaderStyle.Render("Command:"))
		addLine(v2styles.LayerValueStyle.Render("(unavailable)"))
	}

finish:
	content := strings.Join(lines, "\n")
	return v2styles.RenderBox("Layer Details", width, height, content, m.activePane == PaneDetails)
}

func (m Model) renderTreePane() string {
	width := m.layout.RightWidth
	height := m.layout.TreeHeight

	if m.treeVM == nil {
		return v2styles.RenderBox("Current Layer Contents", width, height, "No tree data", m.activePane == PaneTree)
	}

	// Get viewport content (already set in Update())
	content := m.treeViewport.View()
	return v2styles.RenderBox("Current Layer Contents", width, height, content, m.activePane == PaneTree)
}

func (m Model) renderImageDetailsPane() string {
	width := m.layout.LeftWidth
	height := m.layout.ImageHeight
	isActive := m.activePane == PaneImage

	// Get viewport content (already set in Update())
	content := m.imageViewport.View()
	return v2styles.RenderBox("Image Details", width, height, content, isActive)
}

func (m Model) renderStatusBar(width int) string {
	left := fmt.Sprintf("Tab: %s", m.activePane.String())
	center := "↑↓: Navigate"
	right := "Q: Quit"

	leftStyle := v2styles.StatusStyle.Width(width/3 - 1)
	centerStyle := v2styles.StatusStyle.Width(width/3 - 1)
	rightStyle := v2styles.StatusStyle.Width(width - 2*(width/3-1) - 2)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(left),
		centerStyle.Render(center),
		rightStyle.Render(right),
	)
}

func (m Model) renderFilterBar(width int) string {
	content := " Filter: " + strings.Repeat(" ", width-12)
	return v2styles.FilterStyle.Width(width).Render(content)
}

func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncate is deprecated - use runewidth.Truncate instead
func truncate(s string, maxLen int) string {
	return runewidth.Truncate(s, maxLen, "...")
}
