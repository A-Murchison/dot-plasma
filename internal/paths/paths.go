package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dot-plasma/internal/allowlist"
)

type LiveRoots struct {
	Config     string
	LocalShare string
}

func DefaultLiveRoots() (LiveRoots, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return LiveRoots{}, fmt.Errorf("find home directory: %w", err)
	}
	return LiveRoots{
		Config:     filepath.Join(home, ".config"),
		LocalShare: filepath.Join(home, ".local", "share"),
	}, nil
}

func (r LiveRoots) Root(root allowlist.Root) (string, error) {
	switch root {
	case allowlist.RootConfig:
		return r.Config, nil
	case allowlist.RootLocalShare:
		return r.LocalShare, nil
	default:
		return "", fmt.Errorf("unknown root %q", root)
	}
}

func (r LiveRoots) Join(root allowlist.Root, relativePath string) (string, error) {
	base, err := r.Root(root)
	if err != nil {
		return "", err
	}
	if relativePath == "" {
		return "", fmt.Errorf("relative path is empty")
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("path %q must be relative", relativePath)
	}
	clean := filepath.Clean(relativePath)
	if clean == "." || clean != relativePath || strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is unsafe", relativePath)
	}
	joined := filepath.Join(base, clean)
	if !isWithin(base, joined) {
		return "", fmt.Errorf("path %q escapes root %q", relativePath, base)
	}
	return joined, nil
}

func isWithin(base, candidate string) bool {
	baseClean, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	candidateClean, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseClean, candidateClean)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}
