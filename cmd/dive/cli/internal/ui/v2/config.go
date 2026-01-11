package v2

import (
	"context"
	"sync"

	"github.com/wagoodman/dive/cmd/dive/cli/internal/ui/v1"
	"github.com/wagoodman/dive/dive/filetree"
	"github.com/wagoodman/dive/dive/image"
)

// Config holds the configuration for V2UI
type Config struct {
	Analysis    image.Analysis
	Content     ContentReader
	Preferences v1.Preferences
	stack       filetree.Comparer
	stackErr    error
}

type ContentReader interface {
	Extract(ctx context.Context, id string, layer string, path string) error
}

// TreeComparer returns the filetree comparer (lazy loaded)
func (c *Config) TreeComparer() (filetree.Comparer, error) {
	var once sync.Once
	once.Do(func() {
		treeStack := filetree.NewComparer(c.Analysis.RefTrees)
		errs := treeStack.BuildCache()
		if errs != nil && !c.Preferences.IgnoreErrors {
			c.stackErr = errs[0]
		}
		c.stack = treeStack
	})
	return c.stack, c.stackErr
}

// NewConfig creates a new V2UI config
func NewConfig(analysis image.Analysis, content ContentReader, prefs v1.Preferences) Config {
	return Config{
		Analysis:    analysis,
		Content:     content,
		Preferences: prefs,
	}
}
