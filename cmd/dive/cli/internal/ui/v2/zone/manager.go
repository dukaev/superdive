// Package zone provides a simple zone manager for handling mouse clicks
// This is a lightweight alternative to external zone libraries
package zone

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Manager tracks clickable regions in the UI
type Manager struct {
	mu    sync.RWMutex
	zones map[string]Rect
}

// Rect represents a rectangular region
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// New creates a new zone manager
func New() *Manager {
	return &Manager{
		zones: make(map[string]Rect),
	}
}

// Set defines a zone with the given ID and boundaries
func (m *Manager) Set(id string, x, y, width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones[id] = Rect{X: x, Y: y, Width: width, Height: height}
}

// Get retrieves a zone by ID
func (m *Manager) Get(id string) (Rect, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rect, ok := m.zones[id]
	return rect, ok
}

// Contains checks if a point is within a zone
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// At the given coordinates returns all zone IDs that contain this point
func (m *Manager) At(x, y int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ids []string
	for id, rect := range m.zones {
		if rect.Contains(x, y) {
			ids = append(ids, id)
		}
	}
	return ids
}

// Clear removes all zones
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones = make(map[string]Rect)
}

// QueryMsg is a tea.Msg that requests zone information at coordinates
type QueryMsg struct {
	X, Y int
}

// ResponseMsg contains the zone IDs at the queried coordinates
type ResponseMsg struct {
	IDs []string
}

// Handler creates a tea.Cmd that responds to QueryMsg
func (m *Manager) Handler() func(tea.Msg) *ResponseMsg {
	return func(msg tea.Msg) *ResponseMsg {
		query, ok := msg.(QueryMsg)
		if !ok {
			return nil
		}
		ids := m.At(query.X, query.Y)
		return &ResponseMsg{IDs: ids}
	}
}
