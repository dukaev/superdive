package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	v1 "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/keys"
	filetree "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/filetree"
	imagepane "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/image"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/details"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/layers"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/dive/image"
)

// Pane represents a UI pane
type Pane int

const (
	PaneLayer Pane = iota
	PaneDetails
	PaneImage
	PaneTree
)

// Use layout package constants
const (
	BorderHeight         = layout.BorderHeight
	HeaderHeight         = layout.HeaderHeight
	BoxContentPadding    = layout.BoxContentPadding
	ContentVisualOffset  = layout.ContentVisualOffset
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

	// Pane components (independent tea.Models)
	layersPane  layers.Pane
	detailsPane details.Pane
	imagePane   imagepane.Pane
	treePane    filetree.Pane

	// Active pane state
	activePane Pane

	// Filter state
	filter FilterModel

	// Help and key bindings
	keys keys.KeyMap
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
	h.Styles.ShortKey = styles.StatusStyle
	h.Styles.ShortDesc = styles.StatusStyle
	h.Styles.Ellipsis = styles.StatusStyle

	f := NewFilterModel()

	// Create pane components
	layersPane := layers.New(layerVM)
	detailsPane := details.New()
	imagePane := imagepane.New(&analysis)
	treePane := filetree.New(treeVM)

	// Set initial focus
	layersPane.Focus()

	// Create model with initial dimensions
	model := Model{
		analysis:    analysis,
		content:     content,
		prefs:       prefs,
		ctx:         ctx,
		layerVM:     layerVM,
		treeVM:      treeVM,
		layersPane:  layersPane,
		detailsPane: detailsPane,
		imagePane:   imagePane,
		treePane:    treePane,
		width:       80,
		height:      24,
		quitting:    false,
		activePane:  PaneLayer,
		keys:        keys.Keys,
		help:        h,
		filter:      f,
	}

	// CRITICAL: Calculate initial layout and set pane sizes immediately
	// This ensures panes have correct dimensions before first render
	model.recalculateLayout()
	model.layersPane.SetSize(model.layout.LeftWidth, model.layout.LayersHeight)
	model.detailsPane.SetSize(model.layout.LeftWidth, model.layout.DetailsHeight)
	model.imagePane.SetSize(model.layout.LeftWidth, model.layout.ImageHeight)
	model.treePane.SetSize(model.layout.RightWidth, model.layout.TreeHeight)

	return model
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// IMPORTANT: Initialize tree on startup so it's not empty!
	if m.layerVM != nil && m.treeVM != nil {
		m.updateTreeForCurrentLayer()
	}

	// Initialize details pane with current layer
	if m.layerVM != nil && len(m.layerVM.Layers) > 0 {
		layerIndex := m.layerVM.LayerIndex
		if layerIndex >= 0 && layerIndex < len(m.layerVM.Layers) {
			m.detailsPane.SetLayer(m.layerVM.Layers[layerIndex])
		}
	}

	// Note: Pane sizes are already set in NewModel with initial layout calculation
	// Content is already generated in constructors (NewLayersPane, etc.)

	return tea.Batch(cmds...)
}

// recalculateLayout uses the layout engine to calculate pane dimensions
func (m *Model) recalculateLayout() {
	engine := layout.NewEngine()
	result := engine.Calculate(m.width, m.height)

	m.layout.ContentStartY = result.ContentStartY
	m.layout.LeftWidth = result.LeftWidth
	m.layout.RightWidth = result.RightWidth
	m.layout.LayersHeight = result.LayersHeight
	m.layout.DetailsHeight = result.DetailsHeight
	m.layout.ImageHeight = result.ImageHeight
	m.layout.TreeHeight = result.TreeHeight
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If filter is visible, let filter handle keys
		if m.filter.IsVisible() {
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			cmds = append(cmds, cmd)
			break
		}

		// Route keys to focused pane
		switch m.activePane {
		case PaneLayer:
			newPane, cmd := m.layersPane.Update(msg)
			m.layersPane = newPane.(layers.Pane)
			cmds = append(cmds, cmd)

		case PaneDetails:
			// Details pane is read-only, no keyboard handling

		case PaneImage:
			newPane, cmd := m.imagePane.Update(msg)
			m.imagePane = newPane.(imagepane.Pane)
			cmds = append(cmds, cmd)

		case PaneTree:
			newPane, cmd := m.treePane.Update(msg)
			m.treePane = newPane.(filetree.Pane)
			cmds = append(cmds, cmd)
		}

		// Global key bindings
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "tab", "shift+tab":
			m.togglePane()

		case "ctrl+f", "/":
			m.filter.Show()
		}

	case layers.LayerChangedMsg:
		// Layer changed - update details pane and tree
		if m.layerVM != nil && msg.LayerIndex >= 0 && msg.LayerIndex < len(m.layerVM.Layers) {
			m.detailsPane.SetLayer(m.layerVM.Layers[msg.LayerIndex])
		}
		m.updateTreeForCurrentLayer()

		// Update focus state
		m.layersPane.Focus()
		m.detailsPane.Blur()
		m.imagePane.Blur()
		m.treePane.Blur()

	case filetree.NodeToggledMsg:
		// Tree node was toggled - tree pane already updated its content
		// Nothing to do here

	case filetree.RefreshTreeContentMsg:
		// Request to refresh tree content
		m.treePane.SetTreeVM(m.treeVM)

	case tea.MouseMsg:
		// Route mouse events to appropriate pane
		x, y := msg.X, msg.Y
		l := m.layout

		inLeftCol := x >= 0 && x < l.LeftWidth
		inRightCol := x >= l.LeftWidth && x < m.width

		if inLeftCol {
			// Determine which pane in left column
			layersEndY := l.ContentStartY + l.LayersHeight
			detailsEndY := layersEndY + l.DetailsHeight

			if y < layersEndY {
				// Layers pane
				newPane, cmd := m.layersPane.Update(msg)
				m.layersPane = newPane.(layers.Pane)
				cmds = append(cmds, cmd)
				if m.activePane != PaneLayer {
					m.activePane = PaneLayer
					m.updateFocus()
				}
			} else if y >= layersEndY && y < detailsEndY {
				// Details pane (read-only, no mouse handling)
				if m.activePane != PaneDetails {
					m.activePane = PaneDetails
					m.updateFocus()
				}
			} else {
				// Image pane
				newPane, cmd := m.imagePane.Update(msg)
				m.imagePane = newPane.(imagepane.Pane)
				cmds = append(cmds, cmd)
				if m.activePane != PaneImage {
					m.activePane = PaneImage
					m.updateFocus()
				}
			}
		} else if inRightCol {
			// Tree pane
			newPane, cmd := m.treePane.Update(msg)
			m.treePane = newPane.(filetree.Pane)
			cmds = append(cmds, cmd)
			if m.activePane != PaneTree {
				m.activePane = PaneTree
				m.updateFocus()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.recalculateLayout()

		// Update pane sizes
		m.layersPane.SetSize(m.layout.LeftWidth, m.layout.LayersHeight)
		m.detailsPane.SetSize(m.layout.LeftWidth, m.layout.DetailsHeight)
		m.imagePane.SetSize(m.layout.LeftWidth, m.layout.ImageHeight)
		m.treePane.SetSize(m.layout.RightWidth, m.layout.TreeHeight)
	}

	// Update help
	var cmd tea.Cmd
	m.help, cmd = m.help.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) togglePane() {
	// Cycle through panes: Layer -> Details -> Image -> Tree -> Layer
	m.activePane++
	if m.activePane > PaneTree {
		m.activePane = PaneLayer
	}

	m.updateFocus()
}

func (m *Model) updateFocus() {
	// Update focus state based on current active pane
	// Blur all panes first
	m.layersPane.Blur()
	m.detailsPane.Blur()
	m.imagePane.Blur()
	m.treePane.Blur()

	// Focus only the active pane
	switch m.activePane {
	case PaneLayer:
		m.layersPane.Focus()
	case PaneDetails:
		m.detailsPane.Focus() // Show focus visually
	case PaneImage:
		m.imagePane.Focus()
	case PaneTree:
		m.treePane.Focus()
	}
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

	// Update tree pane with new tree data
	m.treePane.SetTreeVM(m.treeVM)
}

// View implements tea.Model (PURE FUNCTION - no side effects!)
func (m Model) View() string {
	if m.quitting {
		return styles.TitleStyle.Foreground(styles.SuccessColor).Render("Thanks for using Dive V2UI!")
	}

	// Calculate layout if not yet calculated (first run)
	if m.layout.LayersHeight == 0 {
		m.recalculateLayout()
	}

	// Render UI components
	statusBar := m.help.View(m.keys)

	// Add active pane indicator to status bar
	paneName := styles.StatusStyle.Render(fmt.Sprintf(" Active: %s ", m.activePane))
	statusBar = lipgloss.JoinHorizontal(lipgloss.Top, statusBar, strings.Repeat(" ", 5), paneName)

	// Render panes directly using their View() methods
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.layersPane.View(),
		m.detailsPane.View(),
		m.imagePane.View(),
	)
	treePane := m.treePane.View()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, treePane)

	base := lipgloss.JoinVertical(lipgloss.Left,
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

