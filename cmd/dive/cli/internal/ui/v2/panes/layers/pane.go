package layers

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/components"
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

// Define layout constants to ensure click detection matches rendering
const (
	ColWidthPrefix = 1  // " "
	ColWidthID     = 12
	ColWidthSize   = 9
	ColPadding     = 1
	// Calculation: Prefix(1) + ID(12) + Pad(1) + Size(9) + Pad(1)
	StatsStartOffset = ColWidthPrefix + ColWidthID + ColPadding + ColWidthSize + ColPadding
)

// Pane manages the layers list
type Pane struct {
	focused          bool
	width            int
	height           int
	layerVM          *viewmodel.LayerSetState
	comparer         *filetree.Comparer // For computing layer comparison trees
	viewport         viewport.Model
	layerIndex       int
	statsRows        []components.FileStatsRow // Stats row for each layer
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
	p.updateContent()
	return p
}

// SetSize updates the pane dimensions
func (m *Pane) SetSize(width, height int) {
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

// Focus sets the pane as active
func (m *Pane) Focus() {
	m.focused = true
}

// Blur sets the pane as inactive
func (m *Pane) Blur() {
	m.focused = false
}

// IsFocused returns true if the pane is focused
func (m *Pane) IsFocused() bool {
	return m.focused
}

// Init initializes the pane
func (m Pane) Init() tea.Cmd {
	m.updateContent()
	return nil
}

// Update handles messages
func (m Pane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			cmds = append(cmds, m.moveUp())
		case "down", "j":
			cmds = append(cmds, m.moveDown())
		case " ":
			// Show layer detail modal
			if m.layerVM != nil && m.layerIndex >= 0 && m.layerIndex < len(m.layerVM.Layers) {
				cmds = append(cmds, func() tea.Msg {
					return ShowLayerDetailMsg{Layer: m.layerVM.Layers[m.layerIndex]}
				})
			}
		}

	case tea.MouseMsg:
		// Mouse wheel
		if msg.Action == tea.MouseActionPress {
			if msg.Button == tea.MouseButtonWheelUp {
				cmds = append(cmds, m.moveUp())
			} else if msg.Button == tea.MouseButtonWheelDown {
				cmds = append(cmds, m.moveDown())
			}
		}

		// Left click - select layer
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if cmd := m.handleClick(msg.X, msg.Y); cmd != nil {
				cmds = append(cmds, cmd)
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

// handleClick processes a mouse click
func (m *Pane) handleClick(x, y int) tea.Cmd {
	// 1. Basic Bounds Check
	if x < 0 || x >= m.width || y < 0 {
		return nil
	}

	// 2. Adjust Y for Viewport scrolling and Header
	relativeY := y - layout.ContentVisualOffset
	if relativeY < 0 || relativeY >= m.viewport.Height {
		return nil
	}

	targetIndex := relativeY + m.viewport.YOffset
	if targetIndex < 0 || targetIndex >= len(m.layerVM.Layers) {
		return nil
	}

	// 3. Adjust X for Border
	// The pane is rendered with RenderBox, which adds 1 char border on the left.
	// So the content technically starts at X=1 relative to the pane.
	// We subtract 1 to get the X coordinate relative to the *content*.
	contentX := x - 1
	if contentX < 0 {
		return nil
	}

	// 4. Check if click is in stats area
	// We use the shared constant StatsStartOffset to ensure math matches GenerateContent
	if targetIndex < len(m.statsRows) {
		partType, found := m.statsRows[targetIndex].GetPartAtPosition(contentX, StatsStartOffset)
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

	// 5. Click outside stats - select layer
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
		prefix := " "
		style := lipgloss.NewStyle()

		if i == m.layerIndex {
			// No bullet, just color highlighting
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
		statsVisualWidth := 9 // Default approximate width
		if i < len(m.statsRows) {
			// Use comparer to get the comparison tree for this layer
			// For layer i, we want to show changes from layer i-1 to i (or 0 to i for first layer)
			var treeToCompare *filetree.FileTree
			if m.comparer != nil {
				// Get tree for comparing previous layer (or 0) to current layer
				// This follows the CompareSingleLayer mode logic
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

			stats := utils.CalculateFileStats(treeToCompare)
			m.statsRows[i].SetStats(stats)
			statsStr = m.statsRows[i].Render()

			// Calculate total visual width for command truncation math
			addedW := m.statsRows[i].GetAdded().GetVisualWidth()
			modW := m.statsRows[i].GetModified().GetVisualWidth()
			remW := m.statsRows[i].GetRemoved().GetVisualWidth()
			statsVisualWidth = addedW + 1 + modW + 1 + remW // +1 for spaces
		}

		// Clean command from newlines
		rawCmd := strings.ReplaceAll(layer.Command, "\n", " ")
		rawCmd = strings.TrimSpace(rawCmd)

		// Calculate available space for command
		// Logic must match StatsStartOffset constants
		// Used = Prefix(1) + ID(12) + Pad(1) + Size(9) + Pad(1) + StatsWidth + Pad(1)
		usedWidth := StatsStartOffset + statsVisualWidth + 1

		availableCmdWidth := width - usedWidth
		if availableCmdWidth < 0 {
			availableCmdWidth = 0
		}

		cmd := ""
		if availableCmdWidth > 0 && rawCmd != "" {
			cmd = runewidth.Truncate(rawCmd, availableCmdWidth, "...")
		}

		// Build the line using strict column widths
		// %-1s  = Prefix
		// %-*s  = ID (left align, width 12)
		// " "   = Padding
		// %*s   = Size (right align, width 9)
		// " "   = Padding
		// %s    = Stats
		// " "   = Padding
		// %s    = Command
		text := fmt.Sprintf("%-1s%-*s %*s %s %s",
			prefix,
			ColWidthID, id,
			ColWidthSize, size,
			statsStr,
			cmd,
		)

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
