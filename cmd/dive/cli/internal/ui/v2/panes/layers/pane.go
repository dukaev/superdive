package layers

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/components"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/domain"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/keys"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
)

// LayerChangedMsg is sent when the active layer changes
type LayerChangedMsg struct {
	LayerIndex int
}

// ShowLayerDetailMsg is sent to show the layer detail modal
type ShowLayerDetailMsg struct {
	Layer *image.Layer
}

// FocusStateMsg is sent by parent to tell the pane whether it's focused or not
type FocusStateMsg struct {
	Focused bool
}

// Define layout constants to ensure click detection matches rendering
const (
	ColWidthPrefix = 7  // "[1/n] " format (max 6 chars + space)
	ColWidthID     = 12
	ColWidthSize   = 9
	ColWidthDigest = 13 // "sha256:abc12" format (12 chars + space)
	ColPadding     = 1
	// Calculation: Prefix(7) + ID(12) + Pad(1) + Size(9) + Pad(1)
	StatsStartOffset = ColWidthPrefix + ColWidthID + ColPadding + ColWidthSize + ColPadding
)

// Pane manages the layers list
type Pane struct {
	focused          bool // Set by parent via FocusStateMsg, not by Focus()/Blur() methods
	width            int
	height           int
	layerVM          *viewmodel.LayerSetState
	comparer         *filetree.Comparer // For computing layer comparison trees
	viewport         viewport.Model
	layerIndex       int
	statsRows        []components.FileStatsRow // Stats row for each layer
	statsCache       []domain.FileStats         // Cached statistics for each layer (calculated once)
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
		m.statsCache[i] = domain.CalculateFileStats(treeToCompare)
	}
}

// Resize updates the pane dimensions
func (m *Pane) Resize(width, height int) {
	m.width = width
	m.height = height

	viewportWidth := width - 2
	viewportHeight := height - layout.BoxContentPadding
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
		case " ":
			// Show layer detail modal
			if m.layerVM != nil && m.layerIndex >= 0 && m.layerIndex < len(m.layerVM.Layers) {
				cmds = append(cmds, func() tea.Msg {
					return ShowLayerDetailMsg{Layer: m.layerVM.Layers[m.layerIndex]}
				})
			}
		}

	case common.LocalMouseMsg:
		// Mouse coordinates are relative to the marked zone (includes borders + title)
		// We need to subtract visual offsets to get content coordinates
		if msg.Action == tea.MouseActionPress {
			// Content offsets relative to the panel:
			// Y: 1 (top border) + 1 (box title) + 1 (space padding) = 3
			// X: 1 (left border)
			const contentOffsetY = 3
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
	content := m.viewport.View()
	return styles.RenderBox("Layers", m.width, m.height, content, m.focused)
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

	// Check if click is in stats area
	if targetIndex < len(m.statsRows) {
		partType, found := m.statsRows[targetIndex].GetPartAtPosition(x, StatsStartOffset)
		if found {
			// Click on a stats part - toggle that specific part
			part := m.statsRows[targetIndex].GetPart(partType)
			if part != nil {
				part.ToggleActive()
				m.updateContent()
				return nil
			}
		}
	}

	// Click outside stats - select layer
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

	var fullContent strings.Builder

	for i, layer := range m.layerVM.Layers {
		// Format: [current/total]
		totalLayers := len(m.layerVM.Layers)
		prefix := fmt.Sprintf("[%d/%d] ", i+1, totalLayers)
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
		if i < len(m.statsRows) && i < len(m.statsCache) {
			// PERFOMANCE: Use cached stats instead of recalculating on every render
			// This avoids expensive tree traversal (CalculateFileStats) during scrolling
			stats := m.statsCache[i]
			m.statsRows[i].SetStats(stats)

			// Use plain rendering for selected layer to allow background highlight
			// Colors would interfere with the row background color
			if i == m.layerIndex {
				statsStr = m.statsRows[i].RenderPlain()
			} else {
				statsStr = m.statsRows[i].Render()
			}
		}

		// Clean command from newlines
		rawCmd := strings.ReplaceAll(layer.Command, "\n", " ")
		rawCmd = strings.TrimSpace(rawCmd)

		// Truncate command to fixed width (15 chars + "...")
		const maxCmdWidth = 15
		cmd := ""
		if rawCmd != "" {
			cmd = runewidth.Truncate(rawCmd, maxCmdWidth, "...")
		}

		// Format digest (short version: first 12 chars after "sha256:")
		digest := ""
		if layer.Digest != "" {
			// Remove "sha256:" prefix if present and take first 12 chars
			shortDigest := strings.TrimPrefix(layer.Digest, "sha256:")
			if len(shortDigest) > 12 {
				shortDigest = shortDigest[:12]
			}
			// Add gray color styling for digest
			digest = styles.MetaDataStyle.Render(shortDigest)
		}

		// Build the line using strict column widths
		// %-*s  = Prefix (left align, width 7) "[1/n] "
		// %-*s  = ID (left align, width 12)
		// " "   = Padding
		// %*s   = Size (right align, width 9)
		// " "   = Padding
		// %s    = Stats
		// " "   = Padding
		// %s    = Command
		// " "   = Padding before digest
		// %s    = Digest (gray color)
		var text string
		if digest != "" {
			text = fmt.Sprintf("%-*s%-*s %*s %s %s %s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
				cmd,
				digest,
			)
		} else {
			text = fmt.Sprintf("%-*s%-*s %*s %s %s",
				ColWidthPrefix, prefix,
				ColWidthID, id,
				ColWidthSize, size,
				statsStr,
				cmd,
			)
		}

		// Pad to full width for selected layer to ensure background fills entire row
		if i == m.layerIndex {
			textWidth := runewidth.StringWidth(text)
			padding := (width - 2) - textWidth
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
// Layers pane has navigation keys and a special Space key to show layer details.
func (m *Pane) ShortHelp() []key.Binding {
	return []key.Binding{
		keys.Keys.Space,  // Show layer detail modal
	}
}
