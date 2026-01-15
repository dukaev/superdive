package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	v1 "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1/viewmodel"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app/layout"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/keys"
	filetreepane "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/filetree"
	imagepane "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/image"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/details"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/panes/layers"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/common"
	filetree "github.com/wagoodman/dive/dive/filetree"
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
	treePane    filetreepane.Pane

	// Active pane state
	activePane Pane

	// Filter state
	filter FilterModel

	// Layer detail modal
	layerDetailModal LayerDetailModal

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
	var comparer filetree.Comparer
	if len(analysis.RefTrees) > 0 {
		v1cfg := v1.Config{
			Analysis:    analysis,
			Content:     content,
			Preferences: prefs,
		}
		// Get comparer for layer tree comparison
		comparer, _ = v1cfg.TreeComparer()
		// Note: we ignore the error here since treeVM.Update() will be called in Init()
		treeVM, _ = viewmodel.NewFileTreeViewModel(v1cfg, 0)
	}

	h := help.New()
	h.Width = 80
	h.Styles.ShortKey = styles.HelpStyle
	h.Styles.ShortDesc = styles.HelpStyle
	h.Styles.Ellipsis = styles.HelpStyle

	f := NewFilterModel()
	layerDetailModal := NewLayerDetailModal()

	// Create pane components
	layersPane := layers.New(layerVM, comparer)
	detailsPane := details.New()
	imagePane := imagepane.New(&analysis)
	treePane := filetreepane.New(treeVM)

	// Create model with initial dimensions
	model := Model{
		analysis:         analysis,
		content:          content,
		prefs:            prefs,
		ctx:              ctx,
		layerVM:          layerVM,
		treeVM:           treeVM,
		layersPane:       layersPane,
		detailsPane:      detailsPane,
		imagePane:        imagePane,
		treePane:         treePane,
		width:            80,
		height:           24,
		quitting:         false,
		activePane:       PaneLayer,
		keys:             keys.Keys,
		help:             h,
		filter:           f,
		layerDetailModal: layerDetailModal,
	}

	// CRITICAL: Calculate initial layout immediately
	// This ensures panes have correct dimensions before first render
	model.recalculateLayout()

	// Send LayoutMsg to set initial pane sizes via message passing
	layoutMsg := common.LayoutMsg{
		LeftWidth:     model.layout.LeftWidth,
		LayersHeight:  model.layout.LayersHeight,
		DetailsHeight: model.layout.DetailsHeight,
		ImageHeight:   model.layout.ImageHeight,
		RightWidth:    model.layout.RightWidth,
		TreeHeight:    model.layout.TreeHeight,
	}

	// Update panes with initial layout
	newLayers, _ := model.layersPane.Update(layoutMsg)
	model.layersPane = newLayers.(layers.Pane)

	newDetails, _ := model.detailsPane.Update(layoutMsg)
	model.detailsPane = newDetails.(details.Pane)

	newImage, _ := model.imagePane.Update(layoutMsg)
	model.imagePane = newImage.(imagepane.Pane)

	newTree, _ := model.treePane.Update(layoutMsg)
	model.treePane = newTree.(filetreepane.Pane)

	return model
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// IMPORTANT: Initialize tree on startup so it's not empty!
	if m.layerVM != nil && m.treeVM != nil {
		m.updateTreeForCurrentLayer()
	}

	// Initialize details pane with current layer via message
	if m.layerVM != nil && len(m.layerVM.Layers) > 0 {
		layerIndex := m.layerVM.LayerIndex
		if layerIndex >= 0 && layerIndex < len(m.layerVM.Layers) {
			layerMsg := common.LayerSelectedMsg{
				Layer:      m.layerVM.Layers[layerIndex],
				LayerIndex: layerIndex,
			}
			newDetails, _ := m.detailsPane.Update(layerMsg)
			m.detailsPane = newDetails.(details.Pane)
		}
	}

	// CRITICAL: Set initial focus state
	// Parent tells children which pane is focused via FocusStateMsg
	m.sendFocusStates()

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
		// If layer detail modal is visible, let it handle keys first
		if m.layerDetailModal.IsVisible() {
			var cmd tea.Cmd
			m.layerDetailModal, cmd = m.layerDetailModal.Update(msg)
			cmds = append(cmds, cmd)
			break
		}

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
			m.treePane = newPane.(filetreepane.Pane)
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
		// Layer changed - update details pane and tree via messages
		if m.layerVM != nil && msg.LayerIndex >= 0 && msg.LayerIndex < len(m.layerVM.Layers) {
			layerMsg := common.LayerSelectedMsg{
				Layer:      m.layerVM.Layers[msg.LayerIndex],
				LayerIndex: msg.LayerIndex,
			}
			newDetails, _ := m.detailsPane.Update(layerMsg)
			m.detailsPane = newDetails.(details.Pane)
		}
		m.updateTreeForCurrentLayer()

	case filetreepane.NodeToggledMsg:
		// Forward message to tree pane to refresh its visibleNodes cache
		// CRITICAL: This fixes the copy-on-write issue. The InputHandler's callback
		// modified the collapsed flag in the tree data, but the visible copy of
		// treePane (stored in this Model) needs to refresh its cache to show changes.
		newPane, cmd := m.treePane.Update(msg)
		m.treePane = newPane.(filetreepane.Pane)
		cmds = append(cmds, cmd)

	case filetreepane.RefreshTreeContentMsg:
		// Request to refresh tree content
		m.treePane.SetTreeVM(m.treeVM)

	case layers.ShowLayerDetailMsg:
		// Show layer detail modal
		m.layerDetailModal.Show(msg.Layer)

	case tea.MouseMsg:
		// Route mouse events to appropriate pane with coordinate transformation
		// Parent handles ALL coordinate math - children receive simple local coordinates
		x, y := msg.X, msg.Y
		l := m.layout

		inLeftCol := x >= 0 && x < l.LeftWidth
		inRightCol := x >= l.LeftWidth && x < m.width

		if inLeftCol {
			// Determine which pane in left column
			layersEndY := l.ContentStartY + l.LayersHeight
			detailsEndY := layersEndY + l.DetailsHeight

			if y < layersEndY {
				// Layers pane - transform to local coordinates
				// X: relative to pane border (will be adjusted by child for content area)
				// Y: relative to content area (accounting for ContentVisualOffset)
				localX := x
				localY := y - l.ContentStartY
				localMsg := common.LocalMouseMsg{
					MouseMsg: msg,
					LocalX:   localX,
					LocalY:   localY,
				}
				newPane, cmd := m.layersPane.Update(localMsg)
				m.layersPane = newPane.(layers.Pane)
				cmds = append(cmds, cmd)
				if m.activePane != PaneLayer {
					m.activePane = PaneLayer
					m.sendFocusStates()
				}
			} else if y >= layersEndY && y < detailsEndY {
				// Details pane (read-only, no mouse handling)
				if m.activePane != PaneDetails {
					m.activePane = PaneDetails
					m.sendFocusStates()
				}
			} else {
				// Image pane - transform to local coordinates
				localX := x
				localY := y - detailsEndY
				localMsg := common.LocalMouseMsg{
					MouseMsg: msg,
					LocalX:   localX,
					LocalY:   localY,
				}
				newPane, cmd := m.imagePane.Update(localMsg)
				m.imagePane = newPane.(imagepane.Pane)
				cmds = append(cmds, cmd)
				if m.activePane != PaneImage {
					m.activePane = PaneImage
					m.sendFocusStates()
				}
			}
		} else if inRightCol {
			// Tree pane - transform to local coordinates
			localX := x - l.LeftWidth
			localY := y - l.ContentStartY
			localMsg := common.LocalMouseMsg{
				MouseMsg: msg,
				LocalX:   localX,
				LocalY:   localY,
			}
			newPane, cmd := m.treePane.Update(localMsg)
			m.treePane = newPane.(filetreepane.Pane)
			cmds = append(cmds, cmd)
			if m.activePane != PaneTree {
				m.activePane = PaneTree
				m.sendFocusStates()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.recalculateLayout()

		// Send LayoutMsg to all panes with their new dimensions
		// This replaces direct SetSize() calls with message passing
		layoutMsg := common.LayoutMsg{
			LeftWidth:     m.layout.LeftWidth,
			LayersHeight:  m.layout.LayersHeight,
			DetailsHeight: m.layout.DetailsHeight,
			ImageHeight:   m.layout.ImageHeight,
			RightWidth:    m.layout.RightWidth,
			TreeHeight:    m.layout.TreeHeight,
		}

		// Broadcast to all panes - they will extract what they need
		var layoutCmds []tea.Cmd

		newLayers, cmd := m.layersPane.Update(layoutMsg)
		m.layersPane = newLayers.(layers.Pane)
		layoutCmds = append(layoutCmds, cmd)

		newDetails, cmd := m.detailsPane.Update(layoutMsg)
		m.detailsPane = newDetails.(details.Pane)
		layoutCmds = append(layoutCmds, cmd)

		newImage, cmd := m.imagePane.Update(layoutMsg)
		m.imagePane = newImage.(imagepane.Pane)
		layoutCmds = append(layoutCmds, cmd)

		newTree, cmd := m.treePane.Update(layoutMsg)
		m.treePane = newTree.(filetreepane.Pane)
		layoutCmds = append(layoutCmds, cmd)

		cmds = append(cmds, layoutCmds...)
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

	m.sendFocusStates()
}

func (m *Model) sendFocusStates() {
	// Send FocusStateMsg to all panes based on current active pane
	// Parent is the Single Source of Truth - children receive focus state via messages
	switch m.activePane {
	case PaneLayer:
		newPane, _ := m.layersPane.Update(layers.FocusStateMsg{Focused: true})
		m.layersPane = newPane.(layers.Pane)
	case PaneDetails:
		newPane, _ := m.detailsPane.Update(details.FocusStateMsg{Focused: true})
		m.detailsPane = newPane.(details.Pane)
	case PaneImage:
		newPane, _ := m.imagePane.Update(imagepane.FocusStateMsg{Focused: true})
		m.imagePane = newPane.(imagepane.Pane)
	case PaneTree:
		newPane, _ := m.treePane.Update(filetreepane.FocusStateMsg{Focused: true})
		m.treePane = newPane.(filetreepane.Pane)
	}

	// Blur all other panes
	if m.activePane != PaneLayer {
		newPane, _ := m.layersPane.Update(layers.FocusStateMsg{Focused: false})
		m.layersPane = newPane.(layers.Pane)
	}
	if m.activePane != PaneDetails {
		newPane, _ := m.detailsPane.Update(details.FocusStateMsg{Focused: false})
		m.detailsPane = newPane.(details.Pane)
	}
	if m.activePane != PaneImage {
		newPane, _ := m.imagePane.Update(imagepane.FocusStateMsg{Focused: false})
		m.imagePane = newPane.(imagepane.Pane)
	}
	if m.activePane != PaneTree {
		newPane, _ := m.treePane.Update(filetreepane.FocusStateMsg{Focused: false})
		m.treePane = newPane.(filetreepane.Pane)
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

	// Overlay layer detail modal if visible
	if m.layerDetailModal.IsVisible() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.layerDetailModal.View(m.width, m.height))
	}

	// Overlay filter modal if visible
	if m.filter.IsVisible() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.filter.View(m.width, m.height))
	}

	return base
}

