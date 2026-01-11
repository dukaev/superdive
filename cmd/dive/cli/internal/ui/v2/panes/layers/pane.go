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
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/utils"
)

// LayerChangedMsg is sent when the active layer changes
type LayerChangedMsg struct {
	LayerIndex int
}

// Pane manages the layers list
type Pane struct {
	focused    bool
	width      int
	height     int
	layerVM    *viewmodel.LayerSetState
	viewport   viewport.Model
	layerIndex int
}

// New creates a new layers pane
func New(layerVM *viewmodel.LayerSetState) Pane {
	vp := viewport.New(80, 20)
	p := Pane{
		layerVM:    layerVM,
		viewport:   vp,
		layerIndex: 0,
		width:      80,
		height:     20,
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
	if x < 0 || x >= m.width || y < 0 {
		return nil
	}

	relativeY := y - layout.ContentVisualOffset
	if relativeY < 0 || relativeY >= m.viewport.Height {
		return nil
	}

	targetIndex := relativeY + m.viewport.YOffset
	if targetIndex < 0 || targetIndex >= len(m.layerVM.Layers) {
		return nil
	}

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
	width := m.width - 2

	const (
		idWidth   = 12
		sizeWidth = 9
		spaces    = 4
	)

	var fullContent strings.Builder

	for i, layer := range m.layerVM.Layers {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.layerIndex {
			prefix = "● "
			style = styles.SelectedLayerStyle
		}

		id := layer.Id
		if len(id) > idWidth {
			id = id[:idWidth]
		}

		size := utils.FormatSize(layer.Size)

		rawCmd := strings.ReplaceAll(layer.Command, "\n", " ")
		rawCmd = strings.TrimSpace(rawCmd)

		availableCmdWidth := width - 2 - idWidth - spaces - sizeWidth
		if availableCmdWidth < 5 {
			availableCmdWidth = 0
		}

		cmd := ""
		if availableCmdWidth > 0 && rawCmd != "" {
			cmd = runewidth.Truncate(rawCmd, availableCmdWidth, "...")
		}

		text := fmt.Sprintf("%s%-*s  %*s  %s", prefix, idWidth, id, sizeWidth, size, cmd)

		maxLineWidth := width
		if runewidth.StringWidth(text) > maxLineWidth {
			text = runewidth.Truncate(text, maxLineWidth, "")
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
