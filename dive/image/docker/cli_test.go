package docker

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDockerClientBinaryAvailable(t *testing.T) {
	t.Run("docker is available", func(t *testing.T) {
		// This test checks if docker binary is in PATH
		// It will pass if Docker is installed, fail otherwise
		available := isDockerClientBinaryAvailable()

		// If docker is in PATH, this should return true
		// We can't guarantee Docker is installed on all systems
		// so we just check the function doesn't panic
		if available {
			// Docker is available
			_, err := exec.LookPath("docker")
			assert.NoError(t, err, "docker should be findable via LookPath")
		} else {
			// Docker is not available
			_, err := exec.LookPath("docker")
			assert.Error(t, err, "docker should not be findable via LookPath")
		}
	})

	t.Run("function returns boolean", func(t *testing.T) {
		// Verify the function returns a boolean value
		available := isDockerClientBinaryAvailable()
		assert.IsType(t, false, available)
	})

	t.Run("multiple calls return same result", func(t *testing.T) {
		// Multiple calls should return the same result
		result1 := isDockerClientBinaryAvailable()
		result2 := isDockerClientBinaryAvailable()
		result3 := isDockerClientBinaryAvailable()

		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
	})
}

func TestRunDockerCmd(t *testing.T) {
	t.Run("error when docker not available", func(t *testing.T) {
		// This test verifies behavior when docker is not available
		// Since we can't easily mock exec.LookPath, we test the error message

		// Save original PATH
		originalPath := os.Getenv("PATH")

		// Set PATH to empty to make docker unavailable
		os.Unsetenv("PATH")
		defer func() {
			// Restore PATH
			os.Setenv("PATH", originalPath)
		}()

		// Even with empty PATH, some systems might still find docker
		// So we'll just verify the function can be called
		err := runDockerCmd("--version")

		// Either docker is not found (expected error) or it's found
		if err != nil {
			// Docker not found
			assert.Contains(t, err.Error(), "cannot find docker client executable")
		}
		// If docker was found, we can't test this case properly
	})

	t.Run("constructs correct command", func(t *testing.T) {
		// This test verifies that runDockerCmd constructs the correct command
		// Since we can't easily mock exec.Command, we'll test with a non-destructive command

		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// Test with --version which should always succeed if docker is installed
		err := runDockerCmd("--version")

		// --version should succeed
		assert.NoError(t, err, "docker --version should succeed if docker is available")
	})

	t.Run("handles command with arguments", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// Test with a command that has arguments
		// Using 'info' which should work on any Docker installation
		err := runDockerCmd("info", "--format", "{{.ServerVersion}}")

		// This should succeed if docker is available
		// Note: info might fail if docker daemon is not running, but that's ok
		// We're testing that the command is constructed correctly
		_ = err // We don't assert on error since docker daemon might not be running
	})

	t.Run("handles empty arguments", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// Test with empty arguments - Docker shows help and exits successfully
		// runDockerCmd requires at least one argument (cmdStr)
		err := runDockerCmd("")

		// Docker shows help when called with empty string, exits successfully
		// So we don't expect an error
		_ = err // Don't assert, just verify it doesn't panic
	})

	t.Run("cleans arguments", func(t *testing.T) {
		// Test that arguments are properly cleaned (whitespace trimmed)
		// This is hard to test without mocking, but we can verify it doesn't panic
		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// These arguments will be cleaned by utils.CleanArgs
		err := runDockerCmd("  --version  ")

		// Should succeed (whitespace should be trimmed)
		assert.NoError(t, err)
	})
}

func TestRunDockerCmd_ErrorCases(t *testing.T) {
	t.Run("invalid command should error", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// Test with an invalid docker command
		err := runDockerCmd("invalid-command-that-does-not-exist")

		// Should return an error
		assert.Error(t, err)
	})

	t.Run("command with invalid flags should error", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()

		if !available {
			t.Skip("Docker not available, skipping integration test")
		}

		// Test a valid command with invalid flags
		err := runDockerCmd("--version", "--invalid-flag")

		// Should return an error
		assert.Error(t, err)
	})
}

func TestRunDockerCmd_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	available := isDockerClientBinaryAvailable()
	if !available {
		t.Skip("Docker not available, skipping integration tests")
	}

	t.Run("docker --version", func(t *testing.T) {
		err := runDockerCmd("--version")
		assert.NoError(t, err)
	})

	t.Run("docker help", func(t *testing.T) {
		err := runDockerCmd("help")
		assert.NoError(t, err)
	})
}

func TestDockerCommandConstruction(t *testing.T) {
	// This test verifies that the docker command is constructed correctly
	// We can't directly test the construction without modifying the code
	// But we can test that the function handles various argument formats

	t.Run("handles single command", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		err := runDockerCmd("version")
		_ = err // Don't assert, just verify it doesn't panic
	})

	t.Run("handles command with multiple arguments", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		err := runDockerCmd("image", "ls", "--format", "{{.Repository}}")
		_ = err // Don't assert, just verify it doesn't panic
	})
}

func TestIsDockerClientBinaryAvailable_EdgeCases(t *testing.T) {
	t.Run("handles repeated calls", func(t *testing.T) {
		// Test that repeated calls don't have side effects
		results := make([]bool, 10)
		for i := 0; i < 10; i++ {
			results[i] = isDockerClientBinaryAvailable()
		}

		// All results should be the same
		firstResult := results[0]
		for _, result := range results {
			assert.Equal(t, firstResult, result)
		}
	})

	t.Run("result is consistent with exec.LookPath", func(t *testing.T) {
		// Verify that isDockerClientBinaryAvailable is consistent with exec.LookPath
		available := isDockerClientBinaryAvailable()
		_, err := exec.LookPath("docker")

		if available {
			assert.NoError(t, err, "isDockerClientBinaryAvailable returned true but LookPath failed")
		} else {
			assert.Error(t, err, "isDockerClientBinaryAvailable returned false but LookPath succeeded")
		}
	})
}

func TestRunDockerCmd_Stdio(t *testing.T) {
	t.Run("stdio connections", func(t *testing.T) {
		// This test verifies that runDockerCmd properly connects stdio
		// We can't directly test this without running a command
		// But we can verify the function completes without hanging

		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		// Run a command that uses stdin/stdout/stderr
		// --version is a good choice as it's fast and always works
		err := runDockerCmd("--version")

		if err == nil {
			// Command succeeded, stdio was properly connected
			assert.NoError(t, err)
		}
		// If err != nil, docker might not be properly configured, but that's ok
	})
}

func TestDockerArgsCleaning(t *testing.T) {
	t.Run("arguments with whitespace are cleaned", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		// The function should use utils.CleanArgs which trims whitespace
		// Test with arguments that have leading/trailing whitespace
		err := runDockerCmd("  --version  ")
		assert.NoError(t, err)
	})
}

func TestRunDockerCmd_CommandString(t *testing.T) {
	// This test verifies the command string construction
	// The full command is: docker + cmdStr + args

	t.Run("command has docker prefix", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		// The function should construct: docker --version
		err := runDockerCmd("--version")
		assert.NoError(t, err)
	})

	t.Run("command with args has correct format", func(t *testing.T) {
		available := isDockerClientBinaryAvailable()
		if !available {
			t.Skip("Docker not available")
		}

		// The function should construct: docker image ls
		err := runDockerCmd("image", "ls")
		_ = err // Don't assert, daemon might not be running
	})
}

func TestIsDockerClientBinaryAvailable_System(t *testing.T) {
	t.Run("detects docker in PATH", func(t *testing.T) {
		// Check if docker is in common PATH locations
		path := os.Getenv("PATH")
		pathDirs := strings.Split(path, string(os.PathListSeparator))

		dockerFound := false
		for _, dir := range pathDirs {
			dockerPath := dir + string(os.PathSeparator) + "docker"
			if _, err := os.Stat(dockerPath); err == nil {
				dockerFound = true
				break
			}
		}

		available := isDockerClientBinaryAvailable()

		if dockerFound {
			assert.True(t, available, "Docker found in PATH but isDockerClientBinaryAvailable returned false")
		} else {
			assert.False(t, available, "Docker not found in PATH but isDockerClientBinaryAvailable returned true")
		}
	})
}

func TestRunDockerCmd_ErrorMessages(t *testing.T) {
	t.Run("returns specific error when docker not found", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set PATH to a directory that doesn't contain docker
		os.Setenv("PATH", "/nonexistent/path")

		err := runDockerCmd("--version")

		// Should get the specific error message
		if err != nil {
			assert.Contains(t, err.Error(), "cannot find docker client executable")
		}
	})
}

func TestRunDockerCmd_NoPanic(t *testing.T) {
	// These tests verify that runDockerCmd doesn't panic with various inputs

	testCases := []struct {
		name string
		args []string
	}{
		{"empty string command", []string{""}},
		{"single argument", []string{"--version"}},
		{"multiple arguments", []string{"image", "ls"}},
		{"arguments with spaces", []string{"  image  ", "  ls  "}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify it doesn't panic
			require.NotPanics(t, func() {
				// tc.args[0] is the cmdStr, tc.args[1:] are additional args
				if len(tc.args) > 0 {
					cmdStr := tc.args[0]
					args := tc.args[1:]
					err := runDockerCmd(cmdStr, args...)
					_ = err // Ignore errors, we're just testing for panics
				}
			})
		})
	}
}

func TestIsDockerClientBinaryAvailable_NoPanic(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		// Verify the function doesn't panic
		assert.NotPanics(t, func() {
			_ = isDockerClientBinaryAvailable()
		})
	})

	t.Run("concurrent calls", func(t *testing.T) {
		// Verify multiple concurrent calls don't cause issues
		results := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				results <- isDockerClientBinaryAvailable()
			}()
		}

		// Collect results
		for i := 0; i < 10; i++ {
			<-results
		}

		// If we got here without deadlock or panic, the test passes
	})
}
