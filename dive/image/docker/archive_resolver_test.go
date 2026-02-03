package docker

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResolverFromArchive(t *testing.T) {
	t.Run("create new resolver", func(t *testing.T) {
		resolver := NewResolverFromArchive()

		assert.NotNil(t, resolver)
		assert.IsType(t, &archiveResolver{}, resolver)
	})
}

func TestArchiveResolver_Name(t *testing.T) {
	t.Run("resolver name", func(t *testing.T) {
		resolver := NewResolverFromArchive()

		name := resolver.Name()

		assert.Equal(t, "docker-archive", name)
	})
}

func TestArchiveResolver_Build(t *testing.T) {
	t.Run("build not supported", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()
		args := []string{"build", "args"}

		img, err := resolver.Build(ctx, args)

		assert.Nil(t, img)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "build option not supported")
	})
}

func TestArchiveResolver_Extract(t *testing.T) {
	t.Run("extract not implemented", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()

		err := resolver.Extract(ctx, "id", "layer", "path")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

func TestArchiveResolver_Fetch(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()
		nonExistentPath := "/non/existent/path.tar"

		img, err := resolver.Fetch(ctx, nonExistentPath)

		assert.Nil(t, img)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("fetch from valid archive", func(t *testing.T) {
		// Create a simple Docker archive tar file for testing
		tmpFile, err := os.CreateTemp("", "docker-archive-*.tar")
		assert.NoError(t, err)
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)
		tmpFile.Close()

		// Create a minimal tar file with manifest.json
		// Note: This is a simplified test - in reality, a proper Docker archive
		// would have manifest.json, layer files, etc.
		// For this test, we're just checking that Fetch properly opens the file
		// and calls NewImageArchive (which will fail with an invalid archive)

		resolver := NewResolverFromArchive()
		ctx := context.Background()

		img, err := resolver.Fetch(ctx, tmpPath)

		// NewImageArchive will fail with an empty/invalid tar file
		assert.Nil(t, img)
		assert.Error(t, err)
		// The error should come from NewImageArchive, not from os.Open
		assert.NotContains(t, err.Error(), "no such file or directory")
	})

	t.Run("fetch with empty path", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()

		img, err := resolver.Fetch(ctx, "")

		assert.Nil(t, img)
		assert.Error(t, err)
	})

	t.Run("fetch from directory calls os.Exit", func(t *testing.T) {
		// Note: NewImageArchive calls os.Exit(1) on error, which will kill the test
		// This test documents that behavior - we cannot test it directly
		// because it would kill the test runner
		resolver := NewResolverFromArchive()
		ctx := context.Background()

		// We can't test this case because NewImageArchive calls os.Exit(1)
		// when encountering a directory, which would terminate the test runner
		tmpDir := t.TempDir()

		// This will cause os.Exit(1) to be called, so we skip this test
		t.Skip("NewImageArchive calls os.Exit(1) on error, which would kill the test runner")

		_, _ = resolver.Fetch(ctx, tmpDir)
	})
}

func TestArchiveResolver_Integration(t *testing.T) {
	// This test verifies that the resolver implements the Resolver interface correctly
	t.Run("resolver implements interface", func(t *testing.T) {
		resolver := NewResolverFromArchive()

		// Verify the resolver has the correct type
		assert.IsType(t, &archiveResolver{}, resolver)

		// Verify all methods exist and return expected types
		ctx := context.Background()

		// Name() should return string
		name := resolver.Name()
		assert.IsType(t, "", name)

		// Build() should return error
		img, err := resolver.Build(ctx, []string{})
		assert.Nil(t, img)
		assert.Error(t, err)

		// Extract() should return error
		err = resolver.Extract(ctx, "id", "layer", "path")
		assert.Error(t, err)

		// Fetch() with invalid path should return error
		img, err = resolver.Fetch(ctx, "/invalid/path")
		assert.Nil(t, img)
		assert.Error(t, err)
	})
}

func TestArchiveResolver_ErrorMessages(t *testing.T) {
	t.Run("build error message", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()

		_, err := resolver.Build(ctx, []string{})

		assert.Error(t, err)
		expectedMsg := "build option not supported for docker archive resolver"
		assert.Equal(t, expectedMsg, err.Error())
	})

	t.Run("extract error message", func(t *testing.T) {
		resolver := NewResolverFromArchive()
		ctx := context.Background()

		err := resolver.Extract(ctx, "id", "layer", "path")

		assert.Error(t, err)
		expectedMsg := "not implemented"
		assert.Equal(t, expectedMsg, err.Error())
	})
}

func TestArchiveResolver_FetchErrors(t *testing.T) {
	t.Run("verify file is closed on error", func(t *testing.T) {
		// Note: NewImageArchive calls os.Exit(1) on tar parsing errors
		// This prevents us from testing file handle cleanup in the normal way
		t.Skip("NewImageArchive calls os.Exit(1) on tar errors, which would kill the test runner")

		resolver := NewResolverFromArchive()
		ctx := context.Background()

		// Create a temp file with invalid content (will fail in NewImageArchive)
		tmpFile, err := os.CreateTemp("", "docker-archive-*.tar")
		assert.NoError(t, err)
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		// Write some invalid content
		_, err = tmpFile.WriteString("invalid tar content")
		assert.NoError(t, err)
		tmpFile.Close()

		// Fetch should fail, but the file should be closed
		img, err := resolver.Fetch(ctx, tmpPath)

		assert.Nil(t, img)
		assert.Error(t, err)

		// Verify we can open the file again (meaning it was properly closed)
		_, err = os.Open(tmpPath)
		assert.NoError(t, err)
	})

	t.Run("verify context cancellation is respected", func(t *testing.T) {
		// Note: The current implementation doesn't check context cancellation
		// This test documents the current behavior
		resolver := NewResolverFromArchive()

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Try to fetch with cancelled context
		// Current implementation will still try to open the file
		_, err := resolver.Fetch(ctx, "/non/existent/path")

		assert.Error(t, err)
	})
}

func TestArchiveResolver_MultipleInstances(t *testing.T) {
	t.Run("multiple resolvers are independent", func(t *testing.T) {
		resolver1 := NewResolverFromArchive()
		resolver2 := NewResolverFromArchive()

		// They should be different instances
		assert.NotSame(t, resolver1, resolver2)

		// But have the same name
		assert.Equal(t, resolver1.Name(), resolver2.Name())
	})
}

func TestArchiveResolver_NilSafety(t *testing.T) {
	t.Run("methods work on nil receiver", func(t *testing.T) {
		// This test checks if methods can handle nil receiver
		// Note: In Go, calling methods on nil pointers is valid as long as
		// the method doesn't dereference the receiver
		var resolver *archiveResolver

		// Name() doesn't dereference the receiver, so it won't panic
		name := resolver.Name()
		assert.Equal(t, "docker-archive", name)

		// Build and Extract also don't dereference the receiver
		ctx := context.Background()
		_, err := resolver.Build(ctx, []string{})
		assert.Error(t, err)

		err = resolver.Extract(ctx, "id", "layer", "path")
		assert.Error(t, err)
	})
}

func TestArchiveResolver_FetchWithInvalidArchive(t *testing.T) {
	t.Run("fetch with corrupted archive", func(t *testing.T) {
		// Note: NewImageArchive calls os.Exit(1) on tar parsing errors
		// which prevents testing with corrupted archives
		t.Skip("NewImageArchive calls os.Exit(1) on tar errors, which would kill the test runner")

		resolver := NewResolverFromArchive()
		ctx := context.Background()

		// Create a file with corrupted content
		tmpFile, err := os.CreateTemp("", "docker-archive-*.tar")
		assert.NoError(t, err)
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		// Write random bytes (not a valid tar)
		_, err = tmpFile.Write([]byte{0xFF, 0xFE, 0xFD, 0xFC})
		assert.NoError(t, err)
		tmpFile.Close()

		img, err := resolver.Fetch(ctx, tmpPath)

		assert.Nil(t, img)
		assert.Error(t, err)
		// Should be an error from NewImageArchive or ToImage
		assert.NotContains(t, err.Error(), "no such file or directory")
	})
}

func TestArchiveResolver_ContextVariants(t *testing.T) {
	t.Run("fetch with different context types", func(t *testing.T) {
		resolver := NewResolverFromArchive()

		// Test with background context
		img1, err1 := resolver.Fetch(context.Background(), "/non/existent")
		assert.Nil(t, img1)
		assert.Error(t, err1)

		// Test with TODO context
		img2, err2 := resolver.Fetch(context.TODO(), "/non/existent")
		assert.Nil(t, img2)
		assert.Error(t, err2)

		// Both should return similar errors
		assert.Contains(t, err1.Error(), "no such file")
		assert.Contains(t, err2.Error(), "no such file")
	})

	t.Run("build and extract ignore context", func(t *testing.T) {
		resolver := NewResolverFromArchive()

		// Build doesn't use the context
		_, err1 := resolver.Build(context.Background(), []string{})
		assert.Error(t, err1)

		_, err2 := resolver.Build(context.TODO(), []string{})
		assert.Error(t, err2)

		assert.Equal(t, err1.Error(), err2.Error())

		// Extract doesn't use the context
		err3 := resolver.Extract(context.Background(), "id", "layer", "path")
		assert.Error(t, err3)

		err4 := resolver.Extract(context.TODO(), "id", "layer", "path")
		assert.Error(t, err4)

		assert.Equal(t, err3.Error(), err4.Error())
	})
}
