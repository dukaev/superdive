// Package filetree provides a tree view component for displaying Docker image file hierarchies.
//
// The package is organized into focused components that work together:
//
// Core Components:
//   - Pane: Main coordinator implementing tea.Model interface
//   - Selection: Manages tree index selection state
//   - ViewportManager: Wrapper around bubbletea viewport
//
// Rendering Components:
//   - HeaderRenderer: Renders table header row
//   - NodeRenderer: Renders individual tree nodes
//   - MetadataFormatter: Formats file metadata (permissions, uid:gid, size)
//   - TreeTraversal: Collects visible nodes and handles tree traversal
//
// Logic Components:
//   - Navigation: Handles navigation movements (up/down/pageup/pagedown)
//   - InputHandler: Processes keyboard and mouse input
//
// Dependency Graph:
//
//	pane.go (coordinator)
//	├── input_handler.go
//	│   ├── navigation.go
//	│   │   ├── selection.go
//	│   │   └── viewport_manager.go
//	│   └── selection.go
//	├── node_renderer.go
//	│   ├── metadata_formatter.go
//	│   └── styles (external)
//	├── header_renderer.go
//	│   └── styles (external)
//	└── tree_traversal.go (pure functions)
//
// Usage:
//
//	treeVM := // ... obtain FileTreeViewModel
//	pane := filetree.New(treeVM)
//	pane.SetSize(width, height)
//	pane.Focus()
//
// Messages:
//   - NodeToggledMsg: Sent when a tree node is collapsed/expanded
//   - TreeSelectionChangedMsg: Sent when a tree node is selected
//   - RefreshTreeContentMsg: Requests tree content to be refreshed
package filetree
