package layout

// Layout constants for viewport calculations
const (
	BorderHeight        = 2                           // Top + Bottom border lines
	HeaderHeight        = 2                           // Title line + newline/padding separator
	BoxContentPadding   = BorderHeight + HeaderHeight // Total padding inside RenderBox
	ContentVisualOffset = 3                           // Offset for mouse hit testing: 1 border + 1 title + 1 padding

	// Additional header heights for specific panes
	TreeTableHeaderHeight = 3 // "Name   Size   Permissions" table header
)

// Result stores calculated pane dimensions
type Result struct {
	ContentStartY int
	LeftWidth     int
	RightWidth    int
	LayersHeight  int
	DetailsHeight int
	ImageHeight   int
	TreeHeight    int
}

// Engine calculates and caches layout dimensions
type Engine struct {
	cache Result
}

// NewEngine creates a new layout engine
func NewEngine() *Engine {
	return &Engine{}
}

// Calculate computes pane dimensions based on terminal size
func (e *Engine) Calculate(width, height int) Result {
	statusBarHeight := 1

	result := Result{}
	result.ContentStartY = 0 // No title, content starts from top

	// Subtract 1 extra line for safety to prevent terminal auto-scroll
	availableHeight := height - statusBarHeight - 1
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Calculate widths (50/50 split with gap)
	result.LeftWidth = (width - 4) / 2
	if result.LeftWidth < 20 {
		result.LeftWidth = 20
	}
	result.RightWidth = width - result.LeftWidth - 2
	if result.RightWidth < 20 {
		result.RightWidth = 20
	}

	// Calculate heights (left column)
	// Layers: flexible, Image: flexible, Details: at least 12 lines
	result.DetailsHeight = 12 // Minimum for command display

	if result.DetailsHeight > availableHeight/3 {
		result.DetailsHeight = availableHeight / 3
	}

	remainingHeight := availableHeight - result.DetailsHeight
	result.LayersHeight = remainingHeight / 2
	if result.LayersHeight < 5 {
		result.LayersHeight = 5
	}

	result.ImageHeight = remainingHeight - result.LayersHeight
	if result.ImageHeight < 5 {
		result.ImageHeight = 5
	}

	result.TreeHeight = availableHeight

	e.cache = result
	return result
}

// GetCached returns the last calculated layout
func (e *Engine) GetCached() Result {
	return e.cache
}

// GetViewportDimensions returns the width and height for a viewport
// accounting for borders and title bar
func (r Result) GetViewportDimensions(paneWidth, paneHeight int) (width, height int) {
	width = paneWidth - 2 // Side borders

	safeHeight := paneHeight - BoxContentPadding
	if safeHeight < 0 {
		safeHeight = 0
	}
	height = safeHeight

	return width, height
}

// PaneID represents a pane identifier (matches app.Pane values)
type PaneID int

const (
	PaneIDLayer   PaneID = 0
	PaneIDDetails PaneID = 1
	PaneIDImage   PaneID = 2
	PaneIDTree    PaneID = 3
)

// GetPaneAt determines which pane is at the given global coordinates.
// Returns: (paneID, localX, localY, found)
//
// Parameters:
//   - x, y: Global screen coordinates
//   - totalWidth: Total screen width
//
// The returned localX, localY are coordinates relative to the pane's content area
// (accounting for borders and titles as appropriate).
func (r Result) GetPaneAt(x, y, totalWidth int) (PaneID, int, int, bool) {
	// Check bounds
	if x < 0 || y < r.ContentStartY {
		return 0, 0, 0, false
	}

	// Determine which column
	inLeftCol := x < r.LeftWidth
	inRightCol := x >= r.LeftWidth && x < totalWidth

	if inLeftCol {
		// Determine which pane in left column
		layersEndY := r.ContentStartY + r.LayersHeight
		detailsEndY := layersEndY + r.DetailsHeight

		if y < layersEndY {
			// Layers pane
			localX := x
			localY := y - r.ContentStartY
			return PaneIDLayer, localX, localY, true
		} else if y >= layersEndY && y < detailsEndY {
			// Details pane (read-only)
			localX := x
			localY := y - layersEndY
			return PaneIDDetails, localX, localY, true
		} else {
			// Image pane
			localX := x
			localY := y - detailsEndY
			return PaneIDImage, localX, localY, true
		}
	} else if inRightCol {
		// Tree pane
		localX := x - r.LeftWidth
		localY := y - r.ContentStartY
		return PaneIDTree, localX, localY, true
	}

	return 0, 0, 0, false
}
