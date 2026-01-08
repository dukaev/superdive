package dive

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wagoodman/dive/dive/image"
	"testing"
)

func TestImageSource_String(t *testing.T) {
	t.Run("SourceUnknown", func(t *testing.T) {
		assert.Equal(t, "unknown", SourceUnknown.String())
	})

	t.Run("SourceDockerEngine", func(t *testing.T) {
		assert.Equal(t, "docker", SourceDockerEngine.String())
	})

	t.Run("SourcePodmanEngine", func(t *testing.T) {
		assert.Equal(t, "podman", SourcePodmanEngine.String())
	})

	t.Run("SourceDockerArchive", func(t *testing.T) {
		assert.Equal(t, "docker-archive", SourceDockerArchive.String())
	})
}

func TestParseImageSource(t *testing.T) {
	t.Run("parse docker", func(t *testing.T) {
		result := ParseImageSource("docker")
		assert.Equal(t, SourceDockerEngine, result)
	})

	t.Run("parse podman", func(t *testing.T) {
		result := ParseImageSource("podman")
		assert.Equal(t, SourcePodmanEngine, result)
	})

	t.Run("parse docker-archive", func(t *testing.T) {
		result := ParseImageSource("docker-archive")
		assert.Equal(t, SourceDockerArchive, result)
	})

	t.Run("parse docker-tar alias", func(t *testing.T) {
		result := ParseImageSource("docker-tar")
		assert.Equal(t, SourceDockerArchive, result)
	})

	t.Run("parse unknown source", func(t *testing.T) {
		result := ParseImageSource("unknown")
		assert.Equal(t, SourceUnknown, result)
	})

	t.Run("parse invalid source", func(t *testing.T) {
		result := ParseImageSource("invalid-source")
		assert.Equal(t, SourceUnknown, result)
	})

	t.Run("parse empty string", func(t *testing.T) {
		result := ParseImageSource("")
		assert.Equal(t, SourceUnknown, result)
	})
}

func TestDeriveImageSource(t *testing.T) {
	t.Run("docker scheme", func(t *testing.T) {
		source, img := DeriveImageSource("docker://my-image:tag")
		assert.Equal(t, SourceDockerEngine, source)
		assert.Equal(t, "my-image:tag", img)
	})

	t.Run("podman scheme", func(t *testing.T) {
		source, img := DeriveImageSource("podman://my-image:tag")
		assert.Equal(t, SourcePodmanEngine, source)
		assert.Equal(t, "my-image:tag", img)
	})

	t.Run("docker-archive scheme", func(t *testing.T) {
		// Note: DeriveImageSource may not support docker-archive scheme
		// This test documents current behavior
		source, img := DeriveImageSource("docker-archive:/path/to/image.tar")
		// Current implementation returns SourceUnknown for docker-archive
		assert.Equal(t, SourceUnknown, source)
		assert.Equal(t, "", img)
	})

	t.Run("docker-tar scheme alias", func(t *testing.T) {
		// Note: DeriveImageSource may not support docker-tar scheme
		// This test documents current behavior
		source, img := DeriveImageSource("docker-tar:/path/to/image.tar")
		// Current implementation returns SourceUnknown for docker-tar
		assert.Equal(t, SourceUnknown, source)
		assert.Equal(t, "", img)
	})

	t.Run("no scheme", func(t *testing.T) {
		source, img := DeriveImageSource("my-image:tag")
		assert.Equal(t, SourceUnknown, source)
		assert.Equal(t, "", img)
	})

	t.Run("unknown scheme", func(t *testing.T) {
		source, img := DeriveImageSource("unknown://my-image:tag")
		assert.Equal(t, SourceUnknown, source)
		assert.Equal(t, "", img)
	})

	t.Run("scheme with multiple colons", func(t *testing.T) {
		source, img := DeriveImageSource("docker://my-image:tag:latest")
		assert.Equal(t, SourceDockerEngine, source)
		assert.Equal(t, "my-image:tag:latest", img)
	})

	t.Run("empty string", func(t *testing.T) {
		source, img := DeriveImageSource("")
		assert.Equal(t, SourceUnknown, source)
		assert.Equal(t, "", img)
	})
}

func TestGetImageResolver(t *testing.T) {
	t.Run("docker engine resolver", func(t *testing.T) {
		resolver, err := GetImageResolver(SourceDockerEngine)
		require.NoError(t, err)
		assert.NotNil(t, resolver)
		// Check that resolver implements the interface
		assert.Implements(t, (*image.Resolver)(nil), resolver)
	})

	t.Run("podman engine resolver", func(t *testing.T) {
		resolver, err := GetImageResolver(SourcePodmanEngine)
		require.NoError(t, err)
		assert.NotNil(t, resolver)
		// Check that resolver implements the interface
		assert.Implements(t, (*image.Resolver)(nil), resolver)
	})

	t.Run("docker archive resolver", func(t *testing.T) {
		resolver, err := GetImageResolver(SourceDockerArchive)
		require.NoError(t, err)
		assert.NotNil(t, resolver)
		// Check that resolver implements the interface
		assert.Implements(t, (*image.Resolver)(nil), resolver)
	})

	t.Run("unknown source returns error", func(t *testing.T) {
		resolver, err := GetImageResolver(SourceUnknown)
		assert.Error(t, err)
		assert.Nil(t, resolver)
		assert.Contains(t, err.Error(), "unable to determine image resolver")
	})

	t.Run("invalid source returns error", func(t *testing.T) {
		invalidSource := ImageSource(99)
		resolver, err := GetImageResolver(invalidSource)
		assert.Error(t, err)
		assert.Nil(t, resolver)
		assert.Contains(t, err.Error(), "unable to determine image resolver")
	})
}

func TestImageSources(t *testing.T) {
	t.Run("contains all valid sources", func(t *testing.T) {
		assert.Contains(t, ImageSources, "docker")
		assert.Contains(t, ImageSources, "podman")
		assert.Contains(t, ImageSources, "docker-archive")
	})

	t.Run("does not contain unknown", func(t *testing.T) {
		assert.NotContains(t, ImageSources, "unknown")
	})
}
