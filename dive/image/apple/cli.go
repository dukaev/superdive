//go:build darwin

package apple

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/wagoodman/dive/internal/log"
	"github.com/wagoodman/dive/internal/utils"
)

// runContainerCmd runs a given Apple container command in the current tty
func runContainerCmd(cmdStr string, args ...string) error {
	if !isContainerClientBinaryAvailable() {
		return fmt.Errorf("cannot find Apple container client executable")
	}

	allArgs := utils.CleanArgs(append([]string{cmdStr}, args...))

	fullCmd := strings.Join(append([]string{"container"}, allArgs...), " ")
	log.WithFields("cmd", fullCmd).Trace("executing")

	cmd := exec.Command("container", allArgs...)
	cmd.Env = os.Environ()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// saveContainerImage saves the container image to a temporary file and returns a reader.
// Apple container CLI requires --output <file> and cannot stream directly to stdout.
func saveContainerImage(imageID string) (io.ReadCloser, func(), error) {
	if !isContainerClientBinaryAvailable() {
		return nil, nil, fmt.Errorf("cannot find Apple container client executable")
	}

	// Create temporary file for the image archive
	tmpFile, err := os.CreateTemp("", "dive-apple-*.tar")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Cleanup function to remove the temporary file
	cleanup := func() {
		os.Remove(tmpPath)
	}

	// Run container image save --output <path> <image>
	args := []string{"image", "save", "--output", tmpPath, imageID}
	fullCmd := strings.Join(append([]string{"container"}, args...), " ")
	log.WithFields("cmd", fullCmd).Trace("executing")

	cmd := exec.Command("container", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to save image: %w", err)
	}

	// Open the saved file for reading
	reader, err := os.Open(tmpPath)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to open saved image: %w", err)
	}

	return reader, cleanup, nil
}

func isContainerClientBinaryAvailable() bool {
	_, err := exec.LookPath("container")
	return err == nil
}
