//go:build darwin

package apple

import (
	"os"
)

func buildImageFromCli(buildArgs []string) (string, error) {
	iidfile, err := os.CreateTemp("", "dive-apple-*.iid")
	if err != nil {
		return "", err
	}
	defer os.Remove(iidfile.Name())
	defer iidfile.Close()

	allArgs := append([]string{"--iidfile", iidfile.Name()}, buildArgs...)
	err = runContainerCmd("build", allArgs...)
	if err != nil {
		return "", err
	}

	imageId, err := os.ReadFile(iidfile.Name())
	if err != nil {
		return "", err
	}

	return string(imageId), nil
}
