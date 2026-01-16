package image

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/testutils"
)

func TestPane_View_NoAnalysis(t *testing.T) {
	// Test with nil analysis
	pane := New(nil)
	pane.Resize(80, 20)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_WithAnalysis(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)
	pane.Resize(80, 20)

	// Initialize the pane
	cmd := pane.Init()
	require.Nil(t, cmd)

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_Focused(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)
	pane.Resize(80, 20)

	// Send focus message
	updatedPane, _ := pane.Update(FocusStateMsg{Focused: true})

	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallWidth(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)
	pane.Resize(40, 20) // Narrow width

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_SmallHeight(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)
	pane.Resize(80, 8) // Short height

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_View_LargeSize(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)
	pane.Resize(120, 40) // Large dimensions

	view := pane.View()
	snaps.MatchSnapshot(t, view)
}

func TestPane_Update_WithLayoutMsg(t *testing.T) {
	// Load test image data
	testData := testutils.LoadTestImage(t)

	pane := New(testData.Analysis)

	// Send layout message
	layoutMsg := testutils.TestLayout()
	updatedPane, _ := pane.Update(layoutMsg)

	// Verify size was updated
	view := updatedPane.(*Pane).View()
	snaps.MatchSnapshot(t, view)
}
