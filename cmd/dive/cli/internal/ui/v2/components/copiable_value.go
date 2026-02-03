package components

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v2/styles"
)

// tickMsg is sent when the copy icon display timer expires
type tickMsg time.Time

// CopyValueTimeout is the duration to show the copy icon
const CopyValueTimeout = time.Second

// CopiableValue represents a value that can be copied to clipboard on double click
type CopiableValue struct {
	value                string
	width                int
	showingCopy          bool           // true when showing copy icon
	style                lipgloss.Style // optional custom style
	truncateWithEllipsis bool           // if true, add "…" when truncating (default: true)
}

// NewCopiableValue creates a new copiable value component
func NewCopiableValue(value string) CopiableValue {
	return CopiableValue{
		value:                value,
		width:                0,
		showingCopy:          false,
		style:                lipgloss.NewStyle(),
		truncateWithEllipsis: true, // default: show ellipsis for truncation
	}
}

// SetValue updates the value
func (c *CopiableValue) SetValue(value string) {
	c.value = value
}

// GetValue returns the current value
func (c *CopiableValue) GetValue() string {
	return c.value
}

// SetWidth sets the display width
func (c *CopiableValue) SetWidth(width int) {
	c.width = width
}

// SetStyle sets a custom style for the value display
func (c *CopiableValue) SetStyle(style lipgloss.Style) {
	c.style = style
}

// SetTruncateWithEllipsis sets whether to add "…" when truncating text
func (c *CopiableValue) SetTruncateWithEllipsis(enabled bool) {
	c.truncateWithEllipsis = enabled
}

// startCopyTimer starts a timer to reset the copy icon display
func startCopyTimer() tea.Cmd {
	return tea.Tick(CopyValueTimeout, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages for the copiable value
func (c CopiableValue) Update(msg tea.Msg) (CopiableValue, tea.Cmd) {
	_, ok := msg.(tickMsg)
	if ok {
		// Timer expired, hide copy icon
		c.showingCopy = false
		return c, nil
	}

	return c, nil
}

// triggerCopy copies the value to clipboard and shows the copy icon
func (c *CopiableValue) triggerCopy() tea.Cmd {
	c.showingCopy = true
	return tea.Batch(
		copyToClipboard(c.value),
		startCopyTimer(),
	)
}

// TriggerCopy publicly accessible method to trigger copy
func (c *CopiableValue) TriggerCopy() tea.Cmd {
	return c.triggerCopy()
}

// View renders the copiable value
func (c CopiableValue) View() string {
	if c.showingCopy {
		// Show copy icon with success color
		// IMPORTANT: Preserve the original width to avoid column width changes
		iconText := styles.IconCopy

		// FIX: Use runewidth for correct visual width calculation
		// len() counts bytes (icon = 3-4 bytes), which breaks layout
		visualWidth := runewidth.StringWidth(iconText)

		if c.width > 0 && visualWidth < c.width {
			// Pad icon to match the original value width
			paddingNeeded := c.width - visualWidth
			if paddingNeeded > 0 {
				iconText += strings.Repeat(" ", paddingNeeded)
			}
		}
		return lipgloss.NewStyle().
			Foreground(styles.PrimaryColor.Dark).
			Render(iconText)
	}

	// Show the actual value
	text := c.value
	if c.width > 0 { //nolint:nestif
		// Truncate if necessary (use visual width, not byte length)
		textWidth := runewidth.StringWidth(text)
		if textWidth > c.width {
			if c.truncateWithEllipsis {
				// Truncate with ellipsis using runewidth
				text = runewidth.Truncate(text, c.width, "…")
			} else {
				// Simple truncation without ellipsis
				text = runewidth.Truncate(text, c.width, "")
			}
		} else {
			// Pad with spaces on the right to match exact width
			paddingNeeded := c.width - textWidth
			if paddingNeeded > 0 {
				text += strings.Repeat(" ", paddingNeeded)
			}
		}
	}

	return c.style.Render(text)
}

// GetVisualWidth returns the visual width of the rendered value
func (c CopiableValue) GetVisualWidth() int {
	if c.showingCopy {
		// Return the configured width (same as original value)
		// This prevents column width changes when showing copy icon
		if c.width > 0 {
			return c.width
		}
		return runewidth.StringWidth(styles.IconCopy)
	}

	if c.width > 0 {
		return c.width
	}
	return runewidth.StringWidth(c.value)
}

// copyToClipboard copies text to the system clipboard
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		ctx := context.Background()

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.CommandContext(ctx, "pbcopy")
		case "linux":
			// Try xclip first, then wl-copy, then xsel
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.CommandContext(ctx, "wl-copy")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.CommandContext(ctx, "xsel", "--clipboard", "--input")
			}
		case "windows":
			cmd = exec.CommandContext(ctx, "clip")
		}

		if cmd != nil {
			cmd.Stdin = bytes.NewBufferString(text)
			_ = cmd.Run()
		}

		return nil
	}
}

// IsShowingCopy returns whether the copy icon is currently being shown
func (c CopiableValue) IsShowingCopy() bool {
	return c.showingCopy
}
