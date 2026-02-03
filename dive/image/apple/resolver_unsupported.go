//go:build !darwin

package apple

import (
	"context"
	"fmt"

	"github.com/wagoodman/dive/dive/image"
)

type resolver struct{}

func NewResolverFromEngine() *resolver {
	return &resolver{}
}

// Name returns the name of the resolver to display to the user.
func (r *resolver) Name() string {
	return "apple"
}

func (r *resolver) Build(ctx context.Context, args []string) (*image.Image, error) {
	return nil, fmt.Errorf("Apple Containers are only supported on macOS")
}

func (r *resolver) Fetch(ctx context.Context, id string) (*image.Image, error) {
	return nil, fmt.Errorf("Apple Containers are only supported on macOS")
}

func (r *resolver) Extract(ctx context.Context, id string, l string, p string) error {
	return fmt.Errorf("Apple Containers are only supported on macOS")
}
