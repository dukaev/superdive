//go:build darwin

package apple

import (
	"context"
	"fmt"

	"github.com/wagoodman/dive/dive/image"
	"github.com/wagoodman/dive/dive/image/docker"
	"github.com/wagoodman/dive/internal/log"
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
	id, err := buildImageFromCli(args)
	if err != nil {
		return nil, err
	}
	return r.Fetch(ctx, id)
}

func (r *resolver) Fetch(ctx context.Context, id string) (*image.Image, error) {
	reader, cleanup, err := saveContainerImage(id)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve image %q: %w", id, err)
	}
	defer cleanup()
	defer reader.Close()

	// Parse OCI archive and select only layers for current platform
	img, arch, err := parseOCIArchiveForPlatform(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to parse image %q: %w", id, err)
	}

	img.Request = id
	log.WithFields("image", id, "arch", arch, "layers", len(img.Layers)).Debug("resolved image for platform")

	return img, nil
}

func (r *resolver) Extract(ctx context.Context, id string, l string, p string) error {
	reader, cleanup, err := saveContainerImage(id)
	if err != nil {
		return fmt.Errorf("unable to extract from image %q: %w", id, err)
	}
	defer cleanup()
	defer reader.Close()

	if err := docker.ExtractFromImage(reader, l, p); err != nil {
		return fmt.Errorf("unable to extract from image %q: %w", id, err)
	}

	return nil
}
