package docker

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsConfig(t *testing.T) {
	t.Run("valid config with layers type", func(t *testing.T) {
		configJSON := `{
			"history": [],
			"rootfs": {
				"type": "layers",
				"diff_ids": []
			}
		}`
		result := isConfig([]byte(configJSON))
		assert.True(t, result)
	})

	t.Run("config with different rootfs type", func(t *testing.T) {
		configJSON := `{
			"history": [],
			"rootfs": {
				"type": "not-layers",
				"diff_ids": []
			}
		}`
		result := isConfig([]byte(configJSON))
		assert.False(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidJSON := `{invalid json}`
		result := isConfig([]byte(invalidJSON))
		assert.False(t, result)
	})

	t.Run("empty JSON object", func(t *testing.T) {
		emptyJSON := `{}`
		result := isConfig([]byte(emptyJSON))
		assert.False(t, result)
	})

	t.Run("missing rootfs field", func(t *testing.T) {
		configJSON := `{"history": []}`
		result := isConfig([]byte(configJSON))
		assert.False(t, result)
	})

	t.Run("config with history entries", func(t *testing.T) {
		configJSON := `{
			"history": [
				{
					"created": "2023-01-01T00:00:00Z",
					"author": "test",
					"created_by": "test command",
					"empty_layer": false
				}
			],
			"rootfs": {
				"type": "layers",
				"diff_ids": ["sha256:abc123"]
			}
		}`
		result := isConfig([]byte(configJSON))
		assert.True(t, result)
	})
}

func TestNewConfig(t *testing.T) {
	t.Run("valid config with non-empty layers", func(t *testing.T) {
		configJSON := `{
			"history": [
				{
					"created": "2023-01-01T00:00:00Z",
					"created_by": "CMD /bin/sh",
					"empty_layer": false
				},
				{
					"created": "2023-01-01T00:00:01Z",
					"created_by": "RUN apt-get update",
					"empty_layer": false
				}
			],
			"rootfs": {
				"type": "layers",
				"diff_ids": [
					"sha256:layer1",
					"sha256:layer2"
				]
			}
		}`
		config := newConfig([]byte(configJSON))

		require.Len(t, config.History, 2)
		assert.Equal(t, "sha256:layer1", config.History[0].ID)
		assert.Equal(t, "sha256:layer2", config.History[1].ID)
		assert.False(t, config.History[0].EmptyLayer)
		assert.False(t, config.History[1].EmptyLayer)
	})

	t.Run("config with empty layers", func(t *testing.T) {
		configJSON := `{
			"history": [
				{
					"created": "2023-01-01T00:00:00Z",
					"created_by": "FROM scratch",
					"empty_layer": true
				},
				{
					"created": "2023-01-01T00:00:01Z",
					"created_by": "RUN echo test",
					"empty_layer": false
				}
			],
			"rootfs": {
				"type": "layers",
				"diff_ids": [
					"sha256:layer1"
				]
			}
		}`
		config := newConfig([]byte(configJSON))

		require.Len(t, config.History, 2)
		assert.Equal(t, "<missing>", config.History[0].ID)
		assert.True(t, config.History[0].EmptyLayer)
		assert.Equal(t, "sha256:layer1", config.History[1].ID)
		assert.False(t, config.History[1].EmptyLayer)
	})

	t.Run("config with mixed empty and non-empty layers", func(t *testing.T) {
		configJSON := `{
			"history": [
				{"empty_layer": true},
				{"empty_layer": false},
				{"empty_layer": true},
				{"empty_layer": false}
			],
			"rootfs": {
				"type": "layers",
				"diff_ids": [
					"sha256:layer1",
					"sha256:layer2"
				]
			}
		}`
		config := newConfig([]byte(configJSON))

		require.Len(t, config.History, 4)
		assert.Equal(t, "<missing>", config.History[0].ID)
		assert.Equal(t, "sha256:layer1", config.History[1].ID)
		assert.Equal(t, "<missing>", config.History[2].ID)
		assert.Equal(t, "sha256:layer2", config.History[3].ID)
	})

	t.Run("empty history and rootfs", func(t *testing.T) {
		configJSON := `{
			"history": [],
			"rootfs": {
				"type": "layers",
				"diff_ids": []
			}
		}`
		config := newConfig([]byte(configJSON))

		require.Len(t, config.History, 0)
		require.Len(t, config.RootFs.DiffIds, 0)
	})

	t.Run("invalid JSON panics", func(t *testing.T) {
		invalidJSON := `{invalid json}`
		assert.Panics(t, func() {
			newConfig([]byte(invalidJSON))
		})
	})

	t.Run("config with all history fields", func(t *testing.T) {
		configJSON := `{
			"history": [
				{
					"id": "original-id",
					"size": 1024,
					"created": "2023-01-01T00:00:00Z",
					"author": "test@example.com",
					"created_by": "CMD /bin/bash",
					"empty_layer": false
				}
			],
			"rootfs": {
				"type": "layers",
				"diff_ids": ["sha256:abc123"]
			}
		}`
		config := newConfig([]byte(configJSON))

		require.Len(t, config.History, 1)
		// ID should be replaced by diff_id
		assert.Equal(t, "sha256:abc123", config.History[0].ID)
		assert.Equal(t, uint64(1024), config.History[0].Size)
		assert.Equal(t, "2023-01-01T00:00:00Z", config.History[0].Created)
		assert.Equal(t, "test@example.com", config.History[0].Author)
		assert.Equal(t, "CMD /bin/bash", config.History[0].CreatedBy)
	})
}

func TestNewManifest(t *testing.T) {
	t.Run("valid manifest", func(t *testing.T) {
		manifestJSON := `[{
			"Config": "sha256:config.json",
			"RepoTags": ["test:latest"],
			"Layers": ["layer1.tar", "layer2.tar"]
		}]`
		manifest := newManifest([]byte(manifestJSON))

		assert.Equal(t, "sha256:config.json", manifest.ConfigPath)
		assert.Equal(t, []string{"test:latest"}, manifest.RepoTags)
		assert.Equal(t, []string{"layer1.tar", "layer2.tar"}, manifest.LayerTarPaths)
	})

	t.Run("manifest with multiple entries (returns first)", func(t *testing.T) {
		manifestJSON := `[{
			"Config": "config1.json",
			"RepoTags": ["test:1"],
			"Layers": ["layer1.tar"]
		}, {
			"Config": "config2.json",
			"RepoTags": ["test:2"],
			"Layers": ["layer2.tar"]
		}]`
		manifest := newManifest([]byte(manifestJSON))

		assert.Equal(t, "config1.json", manifest.ConfigPath)
		assert.Equal(t, []string{"test:1"}, manifest.RepoTags)
	})

	t.Run("manifest with empty arrays", func(t *testing.T) {
		manifestJSON := `[{
			"Config": "config.json",
			"RepoTags": [],
			"Layers": []
		}]`
		manifest := newManifest([]byte(manifestJSON))

		assert.Equal(t, "config.json", manifest.ConfigPath)
		assert.Empty(t, manifest.RepoTags)
		assert.Empty(t, manifest.LayerTarPaths)
	})

	t.Run("manifest with multiple repo tags", func(t *testing.T) {
		manifestJSON := `[{
			"Config": "config.json",
			"RepoTags": ["test:latest", "test:v1.0", "myrepo:test"],
			"Layers": ["layer1.tar"]
		}]`
		manifest := newManifest([]byte(manifestJSON))

		assert.Equal(t, []string{"test:latest", "test:v1.0", "myrepo:test"}, manifest.RepoTags)
	})

	t.Run("invalid JSON panics", func(t *testing.T) {
		invalidJSON := `[invalid json]`
		assert.Panics(t, func() {
			newManifest([]byte(invalidJSON))
		})
	})

	t.Run("empty manifest array panics", func(t *testing.T) {
		emptyJSON := `[]`
		assert.Panics(t, func() {
			newManifest([]byte(emptyJSON))
		})
	})

	t.Run("manifest with only required fields", func(t *testing.T) {
		manifestJSON := `[{
			"Config": "config.json"
		}]`
		manifest := newManifest([]byte(manifestJSON))

		assert.Equal(t, "config.json", manifest.ConfigPath)
		assert.Nil(t, manifest.RepoTags)
		assert.Nil(t, manifest.LayerTarPaths)
	})
}
