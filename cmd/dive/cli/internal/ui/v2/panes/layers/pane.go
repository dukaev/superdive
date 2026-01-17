package layers

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/components"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/domain"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/filetree"
)

// LayerChangedMsg is sent when the active layer changes
type LayerChangedMsg struct {
	LayerIndex int
}

// FocusStateMsg is sent by parent to tell the pane whether it's focused or not
type FocusStateMsg struct {
	Focused bool
}

// Define layout constants to ensure click detection matches rendering
const (
	ColWidthPrefix = 5 // "1/n " format (max 5 chars + space)
	ColWidthID     = 4 // Short ID (first 4 chars of blob)
	ColWidthSize   = 7
	ColWidthDigest = 13 // "sha256:abc12" format (12 chars + space)
	ColPadding     = 1
	// Calculation: Prefix(5) + ID(4) + Pad(1) + Size(9) + Pad(1)
	StatsStartOffset = ColWidthPrefix + ColWidthID + ColPadding + ColWidthSize + ColPadding
	// Minimum width for columns (based on header text length)
	MinWidthPrefix   = ColWidthPrefix
	MinWidthID       = ColWidthID
	MinWidthSize     = ColWidthSize
	MinWidthDigest   = 6 // "Digest" (6 chars)
	MinWidthStats    = 5 // "A M D" (5 chars) - fixed from 3
	MinWidthCommand  = 7 // "Command" (7 chars) - new constant
)

// GetColumnVisibility determines which columns should be shown based on width
// Priority for hiding (first to hide): Command → Digest → Stats
// Priority for keeping (never hide): ID → Size → Prefix
func GetColumnVisibility(width int) (showCommand, showDigest, showStats bool) {
	// Base width: Prefix(5) + ID(4) + Size(7) + padding(2) = 18
	baseWidth := MinWidthPrefix + MinWidthID + ColPadding + MinWidthSize + ColPadding

	// Thresholds for showing columns (include header width)
	minWithStats := baseWidth + MinWidthStats + ColPadding         // 18 + 5 + 1 = 24
	minWithDigest := minWithStats + MinWidthDigest + ColPadding    // 24 + 6 + 1 = 31
	minWithCommand := minWithDigest + MinWidthCommand + ColPadding // 31 + 7 + 1 = 39

	showStats = width >= minWithStats
	showDigest = width >= minWithDigest
	showCommand = width >= minWithCommand

	return showCommand, showDigest, showStats
}

// Pane manages the layers list
type Pane struct {
	focused    bool // Set by parent via FocusStateMsg, not by Focus()/Blur() methods
	width      int
	height     int
	layerVM    *viewmodel.LayerSetState
	comparer   *filetree.Comparer // For computing layer comparison trees
	viewport   viewport.Model
	layerIndex int
	statsRows  []components.FileStatsRow // Stats row for each layer
	statsCache []domain.FileStats        // Cached statistics for each layer (calculated once)
	statsWidths [3]int                  // [0]=AddedWidth, [1]=ModifiedWidth, [2]=RemovedWidth
}

// New creates a new layers pane
func New(layerVM *viewmodel.LayerSetState, comparer filetree.Comparer) Pane {
	vp := viewport.New(80, 20)

	// Initialize stats rows
	var statsRows []components.FileStatsRow
	if layerVM != nil && len(layerVM.Layers) > 0 {
		statsRows = make([]components.FileStatsRow, len(layerVM.Layers))
		for i := range layerVM.Layers {
			statsRows[i] = components.NewFileStatsRow()
		}
	}

	p := Pane{
		layerVM:    layerVM,
		comparer:   &comparer,
		viewport:   vp,
		layerIndex: 0,
		width:      80,
		height:     20,
		statsRows:  statsRows,
	}
	// IMPORTANT: Generate content immediately so viewport is not empty on startup
	// BUT: First calculate stats to avoid heavy computation in View()
	p.precalculateStats()
	p.updateContent()
	return p
}

// precalculateStats calculates file statistics for all layers once
// This is called during initialization to avoid expensive tree traversal during rendering
func (m *Pane) precalculateStats() {
	if m.layerVM == nil || len(m.layerVM.Layers) == 0 {
		m.statsCache = nil
		return
	}

	// Pre-allocate cache for all layers
	m.statsCache = make([]domain.FileStats, len(m.layerVM.Layers))

	// Initialize with minimum width (1 for header "A", "M", "D")
	maxA, maxM, maxD := 1, 1, 1

	// Calculate stats for each layer
	for i, layer := range m.layerVM.Layers {
		var treeToCompare *filetree.FileTree

		// Use comparer to get the comparison tree for this layer
		// For layer i, we want to show changes from layer i-1 to i (or 0 to i for first layer)
		if m.comparer != nil {
			bottomTreeStart := 0
			bottomTreeStop := i - 1
			if bottomTreeStop < 0 {
				bottomTreeStop = i
			}
			topTreeStart := i
			topTreeStop := i

			key := filetree.NewTreeIndexKey(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop)
			comparisonTree, err := m.comparer.GetTree(key)
			if err == nil && comparisonTree != nil {
				treeToCompare = comparisonTree
			}
		}

		// Fallback to layer.Tree if comparer didn't work
		if treeToCompare == nil {
			treeToCompare = layer.Tree
		}

		// Calculate stats ONCE per layer (heavy tree traversal)
		stats := domain.CalculateFileStats(treeToCompare)
		m.statsCache[i] = stats

		// Calculate the string width of each stat value
		lenA := len(utils.FormatCount(stats.Added))
		lenM := len(utils.FormatCount(stats.Modified))
		lenD := len(utils.FormatCount(stats.Removed))

		if lenA > maxA {
			maxA = lenA
		}
		if lenM > maxM {
			maxM = lenM
		}
		if lenD > maxD {
			maxD = lenD
		}
	}

	// Store computed widths
	m.statsWidths = [3]int{maxA, maxM, maxD}
}

// Resize updates the pane dimensions
func (m *Pane) Resize(width, height int) {
	m.width = width
	m.height = height

	// Calculate available height for the viewport content
	// Layout Padding: 2 (Top Border) + 2 (Bottom Border/Title gap) = 4
	// Header visual height: 1
	// Total: 4 (BoxContentPadding) + 1 (visualHeaderHeight) = 5
	const visualHeaderHeight = 1
	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding - visualHeaderHeight
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Width = viewportWidth
	m.viewport.Height = viewportHeight

	// CRITICAL: Regenerate content with new width for proper truncation
	m.updateContent()
}

// SetFocused sets the focus state of the pane
func (m *Pane) SetFocused(focused bool) {
	m.focused = focused
}

// SetLayerVM updates the layer viewmodel
func (m *Pane) SetLayerVM(layerVM *viewmodel.LayerSetState) {
	m.layerVM = layerVM
	if layerVM != nil {
		m.layerIndex = layerVM.LayerIndex
	}
	m.updateContent()
}

// SetLayerIndex sets the current layer index
func (m *Pane) SetLayerIndex(index int) tea.Cmd {
	if m.layerVM == nil || index < 0 || index >= len(m.layerVM.Layers) {
		return nil
	}

	m.layerIndex = index
	m.layerVM.LayerIndex = index
	m.updateContent()

	return func() tea.Msg {
		return LayerChangedMsg{LayerIndex: index}
	}
}

// Init initializes the pane
func (m *Pane) Init() tea.Cmd {
	m.updateContent()
	return nil
}

// Update handles messages and returns the updated Pane
func (m *Pane) Update(msg tea.Msg) (common.Pane, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case common.LayoutMsg:
		// Parent sends layout info instead of calling Resize()
		// Extract what we need from the message
		m.Resize(msg.LeftWidth, msg.LayersHeight)
		return m, nil

	case FocusStateMsg:
		// Parent controls focus state - use SetFocused method
		m.SetFocused(msg.Focused)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "[":
			cmds = append(cmds, m.moveUp())
		case "down", "j", "]":
			cmds = append(cmds, m.moveDown())
		}

	case common.LocalMouseMsg:
		// Mouse coordinates are relative to the marked zone (includes borders + title)
		// We need to subtract visual offsets to get content coordinates
		if msg.Action == tea.MouseActionPress {
			// Content offsets relative to the panel:
			// Y: 1 (top border) + 1 (box title) + 1 (header) + 1 (space padding) = 4
			// X: 1 (left border)
			const contentOffsetY = 4
			const contentOffsetX = 1

			if msg.Button == tea.MouseButtonWheelUp {
				cmds = append(cmds, m.moveUp())
			} else if msg.Button == tea.MouseButtonWheelDown {
				cmds = append(cmds, m.moveDown())
			} else if msg.Button == tea.MouseButtonLeft {
				// Adjust coordinates to be relative to content area
				contentX := msg.LocalX - contentOffsetX
				contentY := msg.LocalY - contentOffsetY

				// Ignore clicks on headers/decorations (negative coordinates)
				if contentY >= 0 && contentX >= 0 {
					if cmd := m.handleClick(contentX, contentY); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the pane
func (m Pane) View() string {
	// 1. Get content from viewport
	content := m.viewport.View()

	// 2. Add table header with dynamic widths
	header := RenderHeader(m.width, m.statsWidths[0], m.statsWidths[1], m.statsWidths[2])

	// 3. Combine header and content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	return styles.RenderBox("Layers", m.width, m.height, fullContent, m.focused)
}

// moveUp moves selection up
func (m *Pane) moveUp() tea.Cmd {
	if m.layerVM == nil || m.layerIndex <= 0 {
		return nil
	}

	m.layerIndex--
	m.layerVM.LayerIndex = m.layerIndex
	m.updateContent()

	return func() tea.Msg {
		return LayerChangedMsg{LayerIndex: m.layerIndex}
	}
}

// moveDown moves selection down
func (m *Pane) moveDown() tea.Cmd {
	if m.layerVM == nil || m.layerIndex >= len(m.layerVM.Layers)-1 {
		return nil
	}

	m.layerIndex++
	m.layerVM.LayerIndex = m.layerIndex
	m.updateContent()

	return func() tea.Msg {
		return LayerChangedMsg{LayerIndex: m.layerIndex}
	}
}

// handleClick processes a mouse click with CONTENT-RELATIVE coordinates
// x, y are provided by the caller after adjusting for visual offsets:
// - x: relative to content area (X=0 is first column of content, after left border)
// - y: relative to content area (Y=0 is first line of content, after title+padding)
//
// The caller has already subtracted:
//   - contentOffsetX (left border)
//   - contentOffsetY (top border + title + padding)
func (m *Pane) handleClick(x, y int) tea.Cmd {
	// Account for viewport scrolling to get absolute layer index
	targetIndex := y + m.viewport.YOffset
	if targetIndex < 0 || targetIndex >= len(m.layerVM.Layers) {
		return nil
	}

	// TEMPORARILY DISABLED: Click on stats (mod, new, del) to toggle visibility
	// This functionality is disabled for now - clicks on stats will select the layer instead
	//
	// // Check if click is in stats area
	// if targetIndex < len(m.statsRows) {
	// 	partType, found := m.statsRows[targetIndex].GetPartAtPosition(x, StatsStartOffset)
	// 	if found {
	// 		// Click on a stats part - toggle that specific part
	// 		part := m.statsRows[targetIndex].GetPart(partType)
	// 		if part != nil {
	// 			part.ToggleActive()
	// 			m.updateContent()
	// 			return nil
	// 		}
	// 	}
	// }

	// All clicks (including stats) select the layer
	return m.SetLayerIndex(targetIndex)
}

// updateContent regenerates the viewport content
func (m *Pane) updateContent() {
	if m.layerVM == nil || len(m.layerVM.Layers) == 0 {
		m.viewport.SetContent("No layer data")
		return
	}

	content := m.generateContent()
	m.viewport.SetContent(content)
}

// generateContent creates the layers content
func (m *Pane) generateContent() string {
	width := m.width - 2 // Viewport width (without panel borders)

	// Determine column visibility based on width
	showCommand, showDigest, showStats := GetColumnVisibility(width)

	var fullContent strings.Builder

	for i, layer := range m.layerVM.Layers {
		// Format: current/total (without brackets)
		totalLayers := len(m.layerVM.Layers)
		prefix := fmt.Sprintf("%d/%d ", i+1, totalLayers)
		style := lipgloss.NewStyle()

		if i == m.layerIndex {
			// Highlight entire row width, not just text
			style = styles.SelectedLayerStyle
		}

		// Format ID
		id := layer.Id
		if len(id) > ColWidthID {
			id = id[:ColWidthID]
		}

		// Format Size
		size := utils.FormatSize(layer.Size)

		// Update and get stats from component
		statsStr := ""
		statsStrPlain := "" // Plain version without colors for width calculation
		if showStats && i < len(m.statsRows) && i < len(m.statsCache) {
			// PERFOMANCE: Use cached stats instead of recalculating on every render
			// This avoids expensive tree traversal (CalculateFileStats) during scrolling
			stats := m.statsCache[i]
			m.statsRows[i].SetStats(stats)

			// Set dynamic widths for stats columns
			m.statsRows[i].SetWidths(m.statsWidths[0], m.statsWidths[1], m.statsWidths[2])

			// Always get plain version for width calculation
			statsStrPlain = m.statsRows[i].RenderPlain()

			// Use plain rendering for selected layer to allow background highlight
			// Colors would interfere with the row background color
			if i == m.layerIndex {
				statsStr = statsStrPlain
			} else {
				statsStr = m.statsRows[i].Render()
			}
		}

		// Format digest (short version: 6 chars)
		digest := ""
		if showDigest && layer.Digest != "" {
			const digestWidth = 6
			// Remove "sha256:" prefix if present and take first 6 chars
			shortDigest := strings.TrimPrefix(layer.Digest, "sha256:")
			if len(shortDigest) > digestWidth {
				shortDigest = shortDigest[:digestWidth]
			}
			// For selected layer, use plain text to allow background highlight
			// For normal layers, add gray color styling
			if i == m.layerIndex {
				digest = shortDigest
			} else {
				digest = styles.MetaDataStyle.Render(shortDigest)
			}
		}

		// Format command to take ALL remaining space
		cmd := ""
		cmdWidth := 0 // Will store the actual width allocated to command
		if showCommand {
			// Clean command: replace tabs/newlines with spaces, collapse multiple spaces
			rawCmd := strings.ReplaceAll(layer.Command, "\t", " ")
			rawCmd = strings.ReplaceAll(rawCmd, "\n", " ")
			rawCmd = strings.ReplaceAll(rawCmd, "\r", " ")

			// Collapse multiple spaces into one
			words := strings.Fields(rawCmd)
			rawCmd = strings.Join(words, " ")

			// Calculate used width by fixed columns
			// Base: Prefix(5) + ID(4) + Size(7) + padding(2) = 18
			usedWidth := ColWidthPrefix + ColWidthID + ColPadding + ColWidthSize + ColPadding

			// Add stats width if visible (use PLAIN version to ignore ANSI colors)
			if showStats {
				usedWidth += runewidth.StringWidth(statsStrPlain) + ColPadding
			}

			// Add digest width if visible
			if showDigest && digest != "" {
				usedWidth += 6 + ColPadding // Digest is always 6 chars
			}

			// Command gets ALL remaining space
			cmdWidth = width - usedWidth
			if cmdWidth < 1 {
				cmdWidth = 1
			}

			// Truncate command to exactly fit available width
			cmd = rawCmd
			if runewidth.StringWidth(cmd) > cmdWidth {
				cmd = runewidth.Truncate(cmd, cmdWidth, "")
			}
		}

		// Build the line dynamically based on column visibility
		var text string
		if showDigest && showCommand {
			// All columns: Prefix ID Size Stats Digest Command
			// Command uses dynamic width to fill remaining space
			text = fmt.Sprintf("%-*s%-*s %*s %s %s %-*s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
				digest,
				cmdWidth, cmd,
			)
		} else if showDigest {
			// Without Command: Prefix ID Size Stats Digest
			text = fmt.Sprintf("%-*s%-*s %*s %s %s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
				digest,
			)
		} else if showCommand {
			// Without Digest: Prefix ID Size Stats Command
			// Command uses dynamic width to fill remaining space
			text = fmt.Sprintf("%-*s%-*s %*s %s %-*s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
				cmdWidth, cmd,
			)
		} else {
			// Only Stats: Prefix ID Size Stats
			text = fmt.Sprintf("%-*s%-*s %*s %s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
			)
		}

		// Pad to full width for selected layer to ensure background fills entire row
		if i == m.layerIndex {
			textWidth := runewidth.StringWidth(text)
			// FIX: width is already (m.width - 2), so don't subtract 2 again
			padding := width - textWidth
			if padding > 0 {
				text += strings.Repeat(" ", padding)
			}
		}

		fullContent.WriteString(style.Render(text))
		fullContent.WriteString("\n")
	}

	return fullContent.String()
}

// GetLayerIndex returns the current layer index
func (m *Pane) GetLayerIndex() int {
	return m.layerIndex
}

// GetViewport returns the underlying viewport
func (m *Pane) GetViewport() *viewport.Model {
	return &m.viewport
}

// ShortHelp returns key bindings specific to the layers pane.
func (m *Pane) ShortHelp() []key.Binding {
	return []key.Binding{}
}
