package v2

import (
	"context"
	"fmt"
	"os"

	"github.com/anchore/clio"
	"github.com/anchore/go-logger/adapter/discard"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lrstanley/bubblezone"
	"github.com/muesli/termenv"
	v1 "github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/app"
	"github.com/wagoodman/dive/dive/image"
	"github.com/wagoodman/dive/internal/bus/event"
	"github.com/wagoodman/dive/internal/bus/event/parser"
	"github.com/wagoodman/dive/internal/log"
	"github.com/wagoodman/go-partybus"
)

var _ clio.UI = (*UI)(nil)

// UI is the V2 UI implementation
type UI struct {
	cfg          v1.Preferences
	out          *os.File
	err          *os.File
	subscription partybus.Unsubscribable
	quiet        bool
	verbosity    int
}

// NewUI creates a new UI instance
func NewUI(cfg v1.Preferences, out *os.File, quiet bool, verbosity int) *UI {
	return &UI{
		cfg:       cfg,
		out:       out,
		err:       os.Stderr,
		quiet:     quiet,
		verbosity: verbosity,
	}
}

// Setup sets up the UI with the given subscription
func (n *UI) Setup(subscription partybus.Unsubscribable) error {
	if n.verbosity == 0 || n.quiet {
		log.Set(discard.New())
	}

	// remove CI var from consideration when determining if we should use the UI
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(n.out, termenv.WithEnvironment(environWithoutCI{})))

	n.subscription = subscription
	return nil
}

var _ termenv.Environ = (*environWithoutCI)(nil)

type environWithoutCI struct{}

func (e environWithoutCI) Environ() []string {
	var out []string
	for _, s := range os.Environ() {
		if s == "CI=" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (e environWithoutCI) Getenv(s string) string {
	if s == "CI" {
		return ""
	}
	return os.Getenv(s)
}

// Handle handles the given event
func (n *UI) Handle(e partybus.Event) error {
	switch e.Type {
	case event.TaskStarted:
		if n.quiet {
			return nil
		}
		// TODO: handle task started
		return nil
	case event.Notification:
		if n.quiet {
			return nil
		}
		// TODO: handle notification
		return nil
	case event.Report:
		if n.quiet {
			return nil
		}
		// TODO: handle report
		return nil
	case event.ExploreAnalysis:
		analysis, content, err := parser.ParseExploreAnalysis(e)
		if err != nil {
			log.WithFields("error", err).Warnf("failed to parse event: %v", e)
			return nil
		}

		// ensure the logger will not interfere with the UI
		log.Set(discard.New())

		return n.runApp(context.Background(), analysis, content)
	}
	return nil
}

func (n *UI) runApp(ctx context.Context, analysis image.Analysis, content image.ContentReader) error {
	// Initialize global zone manager for mouse hit testing
	// This must be called once before starting the bubbletea program
	zone.NewGlobal()

	// Create bubbletea program with initial model
	model := app.NewModel(ctx, analysis, content, n.cfg)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running bubbletea program: %w", err)
	}

	return nil
}

// Teardown tears down the UI
func (n *UI) Teardown(_ bool) error {
	return nil
}
