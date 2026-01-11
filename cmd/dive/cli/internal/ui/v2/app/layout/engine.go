package layout

// Layout constants for viewport calculations
const (
	BorderHeight        = 2 // Top + Bottom border lines
	HeaderHeight        = 2 // Title line + newline/padding separator
	BoxContentPadding   = BorderHeight + HeaderHeight // Total padding inside RenderBox
	ContentVisualOffset = 3 // Offset for mouse hit testing: 1 border + 1 title + 1 padding
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

	// Calculate heights (left column: 40%, 20%, 40%)
	result.LayersHeight = availableHeight * 2 / 5
	if result.LayersHeight < 5 {
		result.LayersHeight = 5
	}

	result.DetailsHeight = availableHeight * 1 / 5
	if result.DetailsHeight < 3 {
		result.DetailsHeight = 3
	}

	result.ImageHeight = availableHeight - result.LayersHeight - result.DetailsHeight
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
