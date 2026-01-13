package filetree

// Selection manages the tree index selection state
type Selection struct {
	treeIndex int
	maxIndex  int
}

// NewSelection creates a new selection with default values
func NewSelection() *Selection {
	return &Selection{
		treeIndex: 0,
		maxIndex:  0,
	}
}

// SetTreeIndex sets the current tree index directly
func (s *Selection) SetTreeIndex(index int) {
	s.treeIndex = index
}

// GetTreeIndex returns the current tree index
func (s *Selection) GetTreeIndex() int {
	return s.treeIndex
}

// MoveToIndex moves selection to the specified index
func (s *Selection) MoveToIndex(index int) {
	s.treeIndex = index
	s.ValidateBounds()
}

// SetMaxIndex updates the maximum valid index
func (s *Selection) SetMaxIndex(max int) {
	s.maxIndex = max
}

// GetMaxIndex returns the maximum valid index
func (s *Selection) GetMaxIndex() int {
	return s.maxIndex
}

// ValidateBounds ensures treeIndex is within [0, maxIndex]
func (s *Selection) ValidateBounds() {
	if s.treeIndex >= s.maxIndex && s.maxIndex > 0 {
		s.treeIndex = s.maxIndex - 1
	}
	if s.treeIndex < 0 {
		s.treeIndex = 0
	}
}
