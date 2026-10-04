package plasma

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrVersionUnavailable = errors.New("plasma version unavailable")

func DetectVersion(ctx context.Context) (string, error) {
	path, err := exec.LookPath("plasmashell")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrVersionUnavailable
		}
		return "", fmt.Errorf("find plasmashell: %w", err)
	}

	output, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("run plasmashell --version: %w", err)
	}
	version := ParseVersionOutput(string(output))
	if version == "" {
		return "", ErrVersionUnavailable
	}
	return version, nil
}

func ParseVersionOutput(output string) string {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
