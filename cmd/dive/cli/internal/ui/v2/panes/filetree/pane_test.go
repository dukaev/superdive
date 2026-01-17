package filetree

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/testutils"
)

func TestPane_View_EmptyTree(t *testing.T) {
	// Test with nil treeVM
	pane := New(nil)
	pane.Resize(50, 20)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_WithTree(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(50, 20)

	// Initialize the pane
	cmd := pane.Init()
	require.Nil(t, cmd)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_Focused(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(50, 20)

	// Send focus message
	updatedPane, _ := pane.Update(FocusStateMsg{Focused: true})

	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallWidth(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(30, 20) // Very narrow width

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallHeight(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(50, 8) // Very short height

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_LargeSize(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(120, 40) // Large dimensions

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_Update_WithLayoutMsg(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)

	// Send layout message
	layoutMsg := testutils.TestLayout()
	updatedPane, _ := pane.Update(layoutMsg)

	// Verify size was updated
	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_Update_TreeNavigation(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)
	pane.Resize(50, 20)

	// Focus the pane
	pane.Update(FocusStateMsg{Focused: true})

	// Move down a few items
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_ShortHelp(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.TreeVM)

	// Test that ShortHelp returns tree-specific key bindings
	keys := pane.ShortHelp()
	require.NotNil(t, keys)
	require.Len(t, keys, 5) // Enter, Space, CollapseAll, ToggleUnmodified, ToggleView

	// Extract key descriptions for verification
	keyHelp := make([]string, len(keys))
	for i, key := range keys {
		keyHelp[i] = key.Help().Key
	}

	// Verify expected keys are present: Enter, Space, CollapseAll, ToggleUnmodified, ToggleView
	require.Contains(t, keyHelp, "enter/spc")
	require.Contains(t, keyHelp, "space")
	require.Contains(t, keyHelp, "shift+c")
	require.Contains(t, keyHelp, "u")
	require.Contains(t, keyHelp, "f")
}
